package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tokoherbal/product-service/internal/reqid"
)

func TestVerify(t *testing.T) {
	var gotAuth, gotReqID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotReqID = r.Header.Get("Authorization"), r.Header.Get("X-Request-ID")
		switch r.Header.Get("Authorization") {
		case "Bearer bagus":
			w.Write([]byte(`{"user_id": 7, "role": "admin"}`))
		case "Bearer rusak":
			w.WriteHeader(http.StatusInternalServerError)
		case "Bearer sampah":
			w.Write([]byte(`bukan json`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)

	ctx := reqid.With(context.Background(), "req-123")
	id, err := c.Verify(ctx, "Bearer bagus")
	if err != nil || id.UserID != 7 || id.Role != "admin" {
		t.Fatalf("id=%+v err=%v", id, err)
	}
	if gotAuth != "Bearer bagus" || gotReqID != "req-123" {
		t.Fatalf("header yang diteruskan: auth=%q reqid=%q", gotAuth, gotReqID)
	}

	cases := []struct {
		name, header string
		want         error
	}{
		{"token ditolak", "Bearer salah", ErrUnauthorized},
		{"header kosong", "", ErrUnauthorized},
		{"auth-service error 500", "Bearer rusak", ErrUnavailable},
		{"respons bukan JSON", "Bearer sampah", ErrUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := c.Verify(context.Background(), tc.header); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyServiceDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // simulasi: container auth-service dimatikan

	_, err := NewClient(url, time.Second).Verify(context.Background(), "Bearer x")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestVerifyTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // auth-service "menggantung"
	}))
	defer srv.Close()
	defer close(release)

	start := time.Now()
	_, err := NewClient(srv.URL, 100*time.Millisecond).Verify(context.Background(), "Bearer x")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("timeout tidak bekerja, menunggu %v", time.Since(start))
	}
}
