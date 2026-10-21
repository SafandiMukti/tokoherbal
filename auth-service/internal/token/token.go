// Package token mengurus dua jenis token:
//
//   - access token: JWT berumur pendek (ditandatangani HS256), dikirim di setiap request.
//   - refresh token: string acak opaque berumur panjang. Hanya hash-nya yang disimpan
//     di database. Dipakai untuk meminta access token baru.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalid = errors.New("token tidak valid")

const issuer = "tokoherbal-auth"

// Claims adalah isi token yang sudah divalidasi.
type Claims struct {
	UserID int
	Role   string
}

type jwtClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret    []byte
	accessTTL time.Duration
}

func NewManager(secret string, accessTTL time.Duration) *Manager {
	return &Manager{secret: []byte(secret), accessTTL: accessTTL}
}

// IssueAccess membuat JWT dan mengembalikan waktu kedaluwarsanya.
func (m *Manager) IssueAccess(userID int, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	claims := jwtClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseAccess memvalidasi tanda tangan, algoritma, issuer, dan masa berlaku.
func (m *Manager) ParseAccess(raw string) (Claims, error) {
	var c jwtClaims
	tok, err := jwt.ParseWithClaims(raw, &c,
		func(*jwt.Token) (any, error) { return m.secret, nil },
		// hanya HS256: menolak token "alg: none" dan serangan pertukaran algoritma
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !tok.Valid {
		return Claims{}, ErrInvalid
	}
	id, err := strconv.Atoi(c.Subject)
	if err != nil || id <= 0 {
		return Claims{}, ErrInvalid
	}
	return Claims{UserID: id, Role: c.Role}, nil
}

// NewRefresh membuat refresh token acak (256 bit). raw diberikan ke klien,
// hash disimpan di database.
func NewRefresh() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefresh(raw), nil
}

// HashRefresh: SHA-256 cukup untuk token acak berentropi tinggi
// (berbeda dengan password manusia yang butuh bcrypt).
func HashRefresh(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
