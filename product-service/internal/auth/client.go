// Package auth adalah klien HTTP ke auth-service (GET /verify).
//
// Product-service TIDAK memegang JWT_SECRET. Ia bertanya ke auth-service apakah
// token valid. Konsekuensinya: setiap request yang butuh login menambah satu
// panggilan jaringan, dan product-service harus tahu apa yang dilakukan bila
// auth-service mati (jawab 503, jangan menggantung dan jangan crash).
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"tokoherbal/product-service/internal/breaker"
	"tokoherbal/product-service/internal/reqid"
)

var (
	// ErrUnauthorized: token tidak ada atau ditolak auth-service (balas 401).
	ErrUnauthorized = errors.New("token tidak valid")
	// ErrUnavailable: auth-service tidak bisa dihubungi, timeout, atau error 5xx (balas 503).
	ErrUnavailable = errors.New("auth service tidak tersedia")
)

type Identity struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
}

type Client struct {
	baseURL string
	http    *http.Client
	timeout time.Duration
	br      *breaker.Breaker
}

// NewClient membuat klien dengan batas waktu per permintaan. Tanpa timeout,
// satu auth-service yang lambat akan menahan semua request product-service.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		timeout: timeout,
		// 3 kegagalan beruntun -> anggap mati 5 detik, request berikutnya
		// langsung 503 tanpa menunggu timeout lagi.
		br: breaker.New(3, 5*time.Second),
	}
}

// Verify menerima nilai header Authorization utuh ("Bearer xxx").
func (c *Client) Verify(ctx context.Context, authHeader string) (Identity, error) {
	if authHeader == "" {
		return Identity{}, ErrUnauthorized
	}

	var id Identity
	var verr error
	// Hanya ErrUnavailable yang dihitung "gagal" oleh breaker. Token ditolak
	// (401) berarti auth-service sehat, jadi fn mengembalikan nil untuknya.
	err := c.br.Do(ctx, c.timeout+time.Second, func(ctx context.Context) error {
		id, verr = c.verify(ctx, authHeader)
		if errors.Is(verr, ErrUnavailable) {
			return verr
		}
		return nil
	})
	switch {
	case errors.Is(err, breaker.ErrOpen):
		return Identity{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	case err != nil:
		return Identity{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return id, verr
}

func (c *Client) verify(ctx context.Context, authHeader string) (Identity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/verify", nil)
	if err != nil {
		return Identity{}, err
	}
	req.Header.Set("Authorization", authHeader)
	// teruskan request id supaya satu permintaan bisa dilacak di log kedua service
	if id := reqid.From(ctx); id != "" {
		req.Header.Set("X-Request-ID", id)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return Identity{}, ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		return Identity{}, fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}

	var id Identity
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&id); err != nil {
		return Identity{}, fmt.Errorf("%w: respons tidak terbaca: %w", ErrUnavailable, err)
	}
	if id.UserID <= 0 {
		return Identity{}, ErrUnauthorized
	}
	return id, nil
}
