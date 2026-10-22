// Package store adalah lapisan yang berbicara dengan PostgreSQL (pgx).
// Interface-nya didefinisikan di package service.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// kode error PostgreSQL
const (
	pgForeignKeyViolation = "23503"
)

// scanner dipenuhi oleh pgx.Row (QueryRow) maupun pgx.Rows (Query).
type scanner interface {
	Scan(dest ...any) error
}

// Connect membuat connection pool lalu mencoba ping beberapa kali.
// Retry berguna saat container database baru menyala dan belum siap menerima koneksi.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL tidak valid: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = pool.Ping(pingCtx)
		cancel()
		if lastErr == nil {
			return pool, nil
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("database tidak bisa dihubungi: %w", lastErr)
}
