package breaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

func ok(context.Context) error   { return nil }
func fail(context.Context) error { return errBoom }

func TestPassThrough(t *testing.T) {
	b := New(3, time.Second)
	if err := b.Do(context.Background(), time.Second, ok); err != nil {
		t.Fatal(err)
	}
	if err := b.Do(context.Background(), time.Second, fail); !errors.Is(err, errBoom) {
		t.Fatalf("error asli harus diteruskan, dapat %v", err)
	}
}

func TestOpensAfterThresholdAndSkipsCalls(t *testing.T) {
	b := New(3, time.Minute)
	var calls atomic.Int32
	counting := func(ctx context.Context) error { calls.Add(1); return errBoom }

	for i := 0; i < 3; i++ {
		b.Do(context.Background(), time.Second, counting)
	}
	start := time.Now()
	err := b.Do(context.Background(), time.Second, counting)
	if !errors.Is(err, ErrOpen) {
		t.Fatalf("harus ErrOpen setelah 3 gagal, dapat %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("saat terbuka, fn tidak boleh dipanggil lagi: calls=%d", calls.Load())
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("penolakan saat terbuka harus instan")
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	b := New(3, time.Minute)
	b.Do(context.Background(), time.Second, fail)
	b.Do(context.Background(), time.Second, fail)
	b.Do(context.Background(), time.Second, ok) // memutus rentetan gagal
	b.Do(context.Background(), time.Second, fail)
	b.Do(context.Background(), time.Second, fail)
	if err := b.Do(context.Background(), time.Second, ok); err != nil {
		t.Fatalf("2 gagal setelah sukses belum boleh membuka breaker: %v", err)
	}
}

func fakeClock(b *Breaker) *time.Time {
	t := time.Unix(1_000_000, 0)
	b.now = func() time.Time { return t }
	return &t
}

func TestHalfOpenRecovery(t *testing.T) {
	b := New(2, 10*time.Second)
	clock := fakeClock(b)
	b.Do(context.Background(), time.Second, fail)
	b.Do(context.Background(), time.Second, fail)
	if err := b.Do(context.Background(), time.Second, ok); !errors.Is(err, ErrOpen) {
		t.Fatalf("masih dalam cooldown, harus ErrOpen: %v", err)
	}
	*clock = clock.Add(11 * time.Second) // cooldown lewat
	if err := b.Do(context.Background(), time.Second, ok); err != nil {
		t.Fatalf("percobaan setelah cooldown harus lolos: %v", err)
	}
	if err := b.Do(context.Background(), time.Second, ok); err != nil {
		t.Fatalf("setelah percobaan berhasil breaker harus tertutup: %v", err)
	}
}

func TestHalfOpenFailureReopens(t *testing.T) {
	b := New(2, 10*time.Second)
	clock := fakeClock(b)
	b.Do(context.Background(), time.Second, fail)
	b.Do(context.Background(), time.Second, fail)
	*clock = clock.Add(11 * time.Second)
	if err := b.Do(context.Background(), time.Second, fail); !errors.Is(err, errBoom) {
		t.Fatalf("percobaan harus dijalankan: %v", err)
	}
	if err := b.Do(context.Background(), time.Second, ok); !errors.Is(err, ErrOpen) {
		t.Fatalf("percobaan gagal harus membuka breaker lagi: %v", err)
	}
}

func TestOnlyOneProbeAtATime(t *testing.T) {
	b := New(1, 10*time.Second)
	clock := fakeClock(b)
	b.Do(context.Background(), time.Second, fail)
	*clock = clock.Add(11 * time.Second)

	release := make(chan struct{})
	started := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Do(context.Background(), 5*time.Second, func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	if err := b.Do(context.Background(), time.Second, ok); !errors.Is(err, ErrOpen) {
		t.Fatalf("saat percobaan berjalan, panggilan lain harus ditolak: %v", err)
	}
	close(release)
	wg.Wait()
}

// Inti perbaikan: fn yang menggantung belasan detik tidak boleh menahan pemanggil.
func TestHardTimeoutEvenIfFnIgnoresContext(t *testing.T) {
	b := New(3, time.Minute)
	start := time.Now()
	err := b.Do(context.Background(), 50*time.Millisecond, func(context.Context) error {
		time.Sleep(2 * time.Second) // meniru dial/DNS yang tidak menghormati context
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("harus DeadlineExceeded, dapat %v", err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Fatalf("Do harus kembali sekitar timeout, bukan menunggu fn: %v", d)
	}
}

func TestCallerCancelIsNotFailure(t *testing.T) {
	b := New(2, time.Minute)
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // klien putus
		b.Do(ctx, time.Second, func(ctx context.Context) error { return ctx.Err() })
	}
	if err := b.Do(context.Background(), time.Second, ok); err != nil {
		t.Fatalf("pembatalan oleh pemanggil tidak boleh membuka breaker: %v", err)
	}
}

func TestPanicBecomesError(t *testing.T) {
	b := New(3, time.Minute)
	err := b.Do(context.Background(), time.Second, func(context.Context) error { panic("kaboom") })
	if err == nil {
		t.Fatal("panic harus menjadi error, bukan menjatuhkan proses")
	}
}

func TestConcurrentUse(t *testing.T) {
	b := New(5, time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if (i+j)%3 == 0 {
					b.Do(context.Background(), 20*time.Millisecond, fail)
				} else {
					b.Do(context.Background(), 20*time.Millisecond, ok)
				}
			}
		}(i)
	}
	wg.Wait()
}
