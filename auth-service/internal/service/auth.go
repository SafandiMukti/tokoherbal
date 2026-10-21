// Package service berisi logika bisnis autentikasi. Tidak tahu apa-apa tentang
// HTTP maupun SQL: ia hanya memakai interface store.Store dan token.Manager.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tokoherbal/auth-service/internal/store"
	"tokoherbal/auth-service/internal/token"
)

var (
	ErrEmailTaken         = errors.New("email sudah terdaftar")
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrInvalidToken       = errors.New("token tidak valid")
)

// InvalidInputError dipakai untuk kesalahan input yang pesannya aman ditampilkan ke klien.
type InvalidInputError struct{ Message string }

func (e *InvalidInputError) Error() string { return e.Message }

type Config struct {
	BcryptCost int           // 0 = bcrypt.DefaultCost
	RefreshTTL time.Duration // 0 = 7 hari
}

type User struct {
	ID    int
	Email string
	Role  string
}

type Identity struct {
	UserID int
	Role   string
}

type TokenPair struct {
	AccessToken     string
	RefreshToken    string
	AccessExpiresAt time.Time
}

type Service struct {
	store  store.Store
	tokens *token.Manager
	cfg    Config
	// dummyHash dipakai agar login dengan email yang tidak ada memakan waktu
	// yang sama dengan email yang ada. Tanpa ini, penyerang bisa menebak email
	// terdaftar dari selisih waktu respons.
	dummyHash []byte
}

func New(s store.Store, tm *token.Manager, cfg Config) (*Service, error) {
	if cfg.BcryptCost == 0 {
		cfg.BcryptCost = bcrypt.DefaultCost
	}
	if cfg.RefreshTTL == 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("password-pura-pura"), cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("buat dummy hash: %w", err)
	}
	return &Service{store: s, tokens: tm, cfg: cfg, dummyHash: dummy}, nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(email) > 254 {
		return "", &InvalidInputError{"format email tidak valid"}
	}
	addr, err := mail.ParseAddress(email)
	// addr.Address != email menolak bentuk seperti "Nama <a@b.com>"
	if err != nil || addr.Address != email {
		return "", &InvalidInputError{"format email tidak valid"}
	}
	return email, nil
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return &InvalidInputError{"password minimal 8 karakter"}
	}
	// bcrypt hanya memakai 72 byte pertama; lebih dari itu ditolak agar tidak menyesatkan
	if len(p) > 72 {
		return &InvalidInputError{"password maksimal 72 karakter"}
	}
	return nil
}

// Register membuat akun baru. Role SELALU customer: tidak ada parameter role.
func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if err := validatePassword(password); err != nil {
		return User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.store.CreateUser(ctx, email, string(hash))
	if errors.Is(err, store.ErrEmailTaken) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return User{ID: u.ID, Email: u.Email, Role: u.Role}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	u, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	return s.issuePair(ctx, u)
}

// Refresh menukar refresh token dengan pasangan token baru (rotation):
// token lama langsung dicabut dan tidak bisa dipakai lagi.
func (s *Service) Refresh(ctx context.Context, rawRefresh string) (TokenPair, error) {
	if rawRefresh == "" {
		return TokenPair{}, ErrInvalidToken
	}
	hash := token.HashRefresh(rawRefresh)

	rt, err := s.store.ConsumeRefreshToken(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		// Token tidak ada, kedaluwarsa, atau sudah dipakai. Kalau ternyata token itu
		// ADA dan sudah dicabut, kemungkinan token curian dipakai ulang (reuse
		// detection): cabut semua sesi pengguna itu demi keamanan.
		if old, getErr := s.store.GetRefreshToken(ctx, hash); getErr == nil && old.RevokedAt != nil {
			if revErr := s.store.RevokeAllRefreshTokens(ctx, old.UserID); revErr != nil {
				return TokenPair{}, fmt.Errorf("revoke all: %w", revErr)
			}
		}
		return TokenPair{}, ErrInvalidToken
	}
	if err != nil {
		return TokenPair{}, fmt.Errorf("consume refresh token: %w", err)
	}

	u, err := s.store.GetUserByID(ctx, rt.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return TokenPair{}, ErrInvalidToken
	}
	if err != nil {
		return TokenPair{}, fmt.Errorf("get user: %w", err)
	}
	return s.issuePair(ctx, u)
}

// Logout mencabut refresh token. Idempotent: token yang tidak ada bukan error.
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	return s.store.RevokeRefreshToken(ctx, token.HashRefresh(rawRefresh))
}

// Verify memvalidasi access token. Dipanggil product-service lewat GET /verify.
// Role dibaca dari token, jadi perubahan role baru berlaku setelah token diperbarui.
func (s *Service) Verify(accessToken string) (Identity, error) {
	c, err := s.tokens.ParseAccess(accessToken)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	return Identity{UserID: c.UserID, Role: c.Role}, nil
}

func (s *Service) issuePair(ctx context.Context, u store.User) (TokenPair, error) {
	access, exp, err := s.tokens.IssueAccess(u.ID, u.Role)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}
	raw, hash, err := token.NewRefresh()
	if err != nil {
		return TokenPair{}, fmt.Errorf("new refresh token: %w", err)
	}
	if err := s.store.SaveRefreshToken(ctx, u.ID, hash, time.Now().Add(s.cfg.RefreshTTL)); err != nil {
		return TokenPair{}, fmt.Errorf("save refresh token: %w", err)
	}
	return TokenPair{AccessToken: access, RefreshToken: raw, AccessExpiresAt: exp}, nil
}

// Ping dipakai endpoint /health.
func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}
