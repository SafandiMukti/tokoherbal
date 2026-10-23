// Package breaker berisi circuit breaker sederhana plus batas waktu keras per operasi.
//
// Masalah yang diselesaikan: saat Redis mati, setiap perintah Redis bisa menunggu
// belasan detik (percobaan sambung ulang dan DNS timeout). Satu request memanggil
// Redis beberapa kali, sehingga total bisa melewati batas waktu server dan
// "Redis mati" berubah menjadi "seluruh aplikasi menggantung".
//
// Cara kerja (tiga keadaan):
//
//	tertutup  : semua panggilan diteruskan. Gagal berturut-turut sebanyak threshold -> terbuka.
//	terbuka   : panggilan langsung ditolak (ErrOpen) tanpa menyentuh layanan, selama cooldown.
//	setengah  : setelah cooldown, SATU panggilan percobaan boleh lewat. Berhasil -> tertutup.
//	            Gagal -> terbuka lagi.
package breaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrOpen dikembalikan saat breaker terbuka: layanan dianggap mati sementara.
var ErrOpen = errors.New("circuit breaker terbuka: layanan dianggap mati sementara")

type Breaker struct {
	threshold int
	cooldown  time.Duration
	now       func() time.Time // bisa diganti di test

	mu        sync.Mutex
	failures  int
	openUntil time.Time // nol = tertutup
	probing   bool      // true saat satu panggilan percobaan sedang berjalan (setengah terbuka)
}

func New(threshold int, cooldown time.Duration) *Breaker {
	return &Breaker{threshold: threshold, cooldown: cooldown, now: time.Now}
}

// allow memutuskan apakah panggilan boleh diteruskan.
func (b *Breaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.openUntil.IsZero() {
		return nil // tertutup
	}
	if b.now().Before(b.openUntil) {
		return ErrOpen // terbuka
	}
	if b.probing {
		return ErrOpen // setengah terbuka: percobaan lain sedang berjalan
	}
	b.probing = true
	return nil
}

func (b *Breaker) success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openUntil = time.Time{}
	b.probing = false
}

func (b *Breaker) failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.probing { // percobaan gagal: buka lagi
		b.probing = false
		b.openUntil = b.now().Add(b.cooldown)
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = b.now().Add(b.cooldown)
	}
}

// neutral: hasil tidak bisa dinilai (misalnya pemanggil membatalkan sendiri).
// Hanya melepas status percobaan supaya breaker tidak macet di "setengah terbuka".
func (b *Breaker) neutral() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.probing = false
}

// Do menjalankan fn dengan batas waktu KERAS. Walau fn mengabaikan context dan
// terus menunggu, Do tetap kembali tepat saat timeout lewat. fn yang masih
// berjalan di latar belakang berhenti sendiri kemudian; selama breaker terbuka
// tidak ada fn baru yang dimulai, jadi goroutine tidak menumpuk.
//
// fn harus mengembalikan nil untuk hasil yang BUKAN kegagalan layanan
// (misalnya key tidak ada di cache), supaya tidak dihitung sebagai gagal.
func (b *Breaker) Do(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	if err := b.allow(); err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1) // berpenyangga: goroutine tidak tertahan bila Do sudah pergi
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic di dalam operasi: %v", r)
			}
		}()
		done <- fn(cctx)
	}()

	var err error
	select {
	case err = <-done:
	case <-cctx.Done():
		err = cctx.Err()
	}

	switch {
	case err == nil:
		b.success()
	case ctx.Err() != nil:
		b.neutral() // pemanggil yang membatalkan (misal klien putus), bukan salah layanan
	default:
		b.failure()
	}
	return err
}
