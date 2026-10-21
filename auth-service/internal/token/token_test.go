package token

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "rahasia-untuk-test-minimal-32-karakter!!"

func TestAccessRoundTrip(t *testing.T) {
	m := NewManager(testSecret, 15*time.Minute)

	raw, exp, err := m.IssueAccess(42, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if !exp.After(time.Now()) {
		t.Fatal("waktu kedaluwarsa seharusnya di masa depan")
	}

	c, err := m.ParseAccess(raw)
	if err != nil {
		t.Fatalf("token valid ditolak: %v", err)
	}
	if c.UserID != 42 || c.Role != "admin" {
		t.Fatalf("claims = %+v", c)
	}
}

func TestParseAccessRejects(t *testing.T) {
	good := NewManager(testSecret, 15*time.Minute)
	valid, _, _ := good.IssueAccess(1, "customer")

	expired, _, _ := NewManager(testSecret, -time.Minute).IssueAccess(1, "customer")
	otherSecret, _, _ := NewManager("secret-lain-yang-juga-panjang-sekali!!", time.Minute).IssueAccess(1, "customer")

	// serangan klasik: token tanpa tanda tangan ("alg: none")
	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "1", "role": "admin", "iss": issuer,
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	// isi diubah (role jadi admin) tapi tanda tangan lama dipertahankan
	parts := strings.Split(valid, ".")
	tampered := parts[0] + ".eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIn0." + parts[2]

	cases := []struct{ name, tok string }{
		{"kosong", ""},
		{"sampah", "bukan-jwt"},
		{"kedaluwarsa", expired},
		{"secret berbeda", otherSecret},
		{"alg none", none},
		{"isi diubah", tampered},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := good.ParseAccess(c.tok); err == nil {
				t.Fatal("token seharusnya ditolak")
			}
		})
	}
}

func TestRefreshTokens(t *testing.T) {
	raw1, hash1, err := NewRefresh()
	if err != nil {
		t.Fatal(err)
	}
	raw2, _, _ := NewRefresh()

	if raw1 == raw2 {
		t.Fatal("token harus acak dan unik")
	}
	if HashRefresh(raw1) != hash1 {
		t.Fatal("hash harus deterministik")
	}
	if hash1 == raw1 {
		t.Fatal("hash tidak boleh sama dengan token asli")
	}
}
