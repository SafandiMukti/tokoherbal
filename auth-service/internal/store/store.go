// Package store adalah lapisan yang berbicara dengan database.
// Service hanya bergantung pada interface Store, bukan pada Postgres langsung,
// sehingga logika bisnis bisa diuji dengan store palsu (lihat memory.go).
package store

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("data tidak ditemukan")
	ErrEmailTaken = errors.New("email sudah terdaftar")
)

type User struct {
	ID           int
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type RefreshToken struct {
	ID        int
	UserID    int
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type Store interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id int) (User, error)

	SaveRefreshToken(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error

	// ConsumeRefreshToken mencabut token yang MASIH AKTIF (belum dicabut, belum
	// kedaluwarsa) dalam satu langkah atomik dan mengembalikannya. Jika ada dua
	// permintaan serentak dengan token yang sama, hanya satu yang berhasil.
	// Mengembalikan ErrNotFound bila token tidak ada, sudah dicabut, atau kedaluwarsa.
	ConsumeRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)

	// GetRefreshToken membaca token apa adanya (termasuk yang sudah dicabut).
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)

	// RevokeRefreshToken mencabut satu token. Tidak error bila token tidak ada.
	RevokeRefreshToken(ctx context.Context, tokenHash string) error

	// RevokeAllRefreshTokens mencabut semua token aktif milik satu pengguna.
	RevokeAllRefreshTokens(ctx context.Context, userID int) error

	Ping(ctx context.Context) error
}
