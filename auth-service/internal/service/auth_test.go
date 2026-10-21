package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tokoherbal/auth-service/internal/store"
	"tokoherbal/auth-service/internal/token"
)

const testSecret = "rahasia-untuk-test-minimal-32-karakter!!"

// newTestService memakai store di memori dan bcrypt termurah supaya test cepat.
func newTestService(t *testing.T, cfg Config) (*Service, *store.Memory) {
	t.Helper()
	mem := store.NewMemory()
	cfg.BcryptCost = bcrypt.MinCost
	svc, err := New(mem, token.NewManager(testSecret, 15*time.Minute), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return svc, mem
}

func TestRegisterValidation(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()

	cases := []struct {
		name     string
		email    string
		password string
		wantErr  bool
	}{
		{"valid", "andi@example.com", "rahasia123", false},
		{"email kosong", "", "rahasia123", true},
		{"email tanpa domain", "andi", "rahasia123", true},
		{"email dengan nama tampilan", "Andi <andi2@example.com>", "rahasia123", true},
		{"password terlalu pendek", "b@example.com", "pendek", true},
		{"password terlalu panjang", "c@example.com", strings.Repeat("a", 73), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Register(ctx, c.email, c.password)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if c.wantErr {
				var invalid *InvalidInputError
				if !errors.As(err, &invalid) {
					t.Fatalf("seharusnya InvalidInputError, dapat %T", err)
				}
			}
		})
	}
}

func TestRegisterAlwaysCustomerAndNormalizesEmail(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()

	u, err := svc.Register(ctx, "  Andi@Example.COM ", "rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != "customer" {
		t.Fatalf("role = %q, harus customer", u.Role)
	}
	if u.Email != "andi@example.com" {
		t.Fatalf("email = %q, harus huruf kecil dan tanpa spasi", u.Email)
	}
	// email yang sama dengan huruf berbeda dianggap duplikat
	if _, err := svc.Register(ctx, "ANDI@example.com", "rahasia123"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("err = %v, harus ErrEmailTaken", err)
	}
}

func TestLogin(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()
	if _, err := svc.Register(ctx, "andi@example.com", "rahasia123"); err != nil {
		t.Fatal(err)
	}

	t.Run("berhasil", func(t *testing.T) {
		pair, err := svc.Login(ctx, "Andi@example.com", "rahasia123")
		if err != nil {
			t.Fatal(err)
		}
		id, err := svc.Verify(pair.AccessToken)
		if err != nil || id.Role != "customer" {
			t.Fatalf("verify = %+v, %v", id, err)
		}
		if pair.RefreshToken == "" {
			t.Fatal("refresh token kosong")
		}
	})

	t.Run("password salah dan email tidak ada memberi error yang sama", func(t *testing.T) {
		_, err1 := svc.Login(ctx, "andi@example.com", "salah-banget")
		_, err2 := svc.Login(ctx, "tidak-ada@example.com", "rahasia123")
		if !errors.Is(err1, ErrInvalidCredentials) || !errors.Is(err2, ErrInvalidCredentials) {
			t.Fatalf("err1 = %v, err2 = %v", err1, err2)
		}
	})
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()
	svc.Register(ctx, "andi@example.com", "rahasia123")
	first, _ := svc.Login(ctx, "andi@example.com", "rahasia123")

	second, err := svc.Refresh(ctx, first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh pertama gagal: %v", err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token harus diganti setiap kali dipakai")
	}

	// memakai token LAMA lagi = mencurigakan: ditolak, dan semua sesi dicabut
	if _, err := svc.Refresh(ctx, first.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token lama seharusnya ditolak, err = %v", err)
	}
	if _, err := svc.Refresh(ctx, second.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("setelah reuse, token baru ikut dicabut, err = %v", err)
	}
}

func TestRefreshExpiredAndUnknown(t *testing.T) {
	svc, _ := newTestService(t, Config{RefreshTTL: -time.Second}) // langsung kedaluwarsa
	ctx := context.Background()
	svc.Register(ctx, "andi@example.com", "rahasia123")
	pair, _ := svc.Login(ctx, "andi@example.com", "rahasia123")

	for name, tok := range map[string]string{
		"kedaluwarsa": pair.RefreshToken,
		"tidak ada":   "token-ngawur",
		"kosong":      "",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.Refresh(ctx, tok); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, harus ErrInvalidToken", err)
			}
		})
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()
	svc.Register(ctx, "andi@example.com", "rahasia123")
	pair, _ := svc.Login(ctx, "andi@example.com", "rahasia123")

	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("setelah logout refresh harus gagal, err = %v", err)
	}
	// logout berulang dan logout token ngawur bukan error
	if err := svc.Logout(ctx, pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, "ngawur"); err != nil {
		t.Fatal(err)
	}
}

// Dua permintaan serentak dengan refresh token yang sama: hanya satu boleh menang.
// Jalankan dengan `go test -race` untuk membuktikan tidak ada race condition.
func TestRefreshConcurrentOnlyOneWins(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	ctx := context.Background()
	svc.Register(ctx, "andi@example.com", "rahasia123")
	pair, _ := svc.Login(ctx, "andi@example.com", "rahasia123")

	const workers = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Refresh(ctx, pair.RefreshToken); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("berhasil %d kali, seharusnya tepat 1", successes)
	}
}

func TestRoleChangeAppearsAfterNewLogin(t *testing.T) {
	svc, mem := newTestService(t, Config{})
	ctx := context.Background()
	svc.Register(ctx, "andi@example.com", "rahasia123")

	mem.SetRole("andi@example.com", "admin") // setara UPDATE users SET role='admin'

	pair, _ := svc.Login(ctx, "andi@example.com", "rahasia123")
	id, err := svc.Verify(pair.AccessToken)
	if err != nil || id.Role != "admin" {
		t.Fatalf("verify = %+v, %v", id, err)
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	svc, _ := newTestService(t, Config{})
	if _, err := svc.Verify("bukan-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v", err)
	}
}
