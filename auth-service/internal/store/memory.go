package store

import (
	"context"
	"sync"
	"time"
)

// Memory adalah Store di memori. Dipakai untuk test supaya tidak butuh database
// sungguhan. Perilakunya sengaja dibuat sama dengan Postgres (termasuk atomik).
// memastikan saat kompilasi bahwa Memory memenuhi interface Store
var _ Store = (*Memory)(nil)

type Memory struct {
	mu         sync.Mutex
	users      map[int]User
	byEmail    map[string]int
	tokens     map[string]*RefreshToken // kunci: hash token
	nextUserID int
	nextTokID  int
}

func NewMemory() *Memory {
	return &Memory{
		users:      map[int]User{},
		byEmail:    map[string]int{},
		tokens:     map[string]*RefreshToken{},
		nextUserID: 1,
		nextTokID:  1,
	}
}

func (m *Memory) Ping(context.Context) error { return nil }

// SetRole hanya untuk test (di produksi role diubah lewat SQL).
func (m *Memory) SetRole(email, role string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id, ok := m.byEmail[email]; ok {
		u := m.users[id]
		u.Role = role
		m.users[id] = u
	}
}

func (m *Memory) CreateUser(_ context.Context, email, passwordHash string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, taken := m.byEmail[email]; taken {
		return User{}, ErrEmailTaken
	}
	u := User{ID: m.nextUserID, Email: email, PasswordHash: passwordHash, Role: "customer", CreatedAt: time.Now()}
	m.nextUserID++
	m.users[u.ID] = u
	m.byEmail[email] = u.ID
	return u, nil
}

func (m *Memory) GetUserByEmail(_ context.Context, email string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return m.users[id], nil
}

func (m *Memory) GetUserByID(_ context.Context, id int) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (m *Memory) SaveRefreshToken(_ context.Context, userID int, tokenHash string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[tokenHash] = &RefreshToken{ID: m.nextTokID, UserID: userID, ExpiresAt: expiresAt}
	m.nextTokID++
	return nil
}

func (m *Memory) ConsumeRefreshToken(_ context.Context, tokenHash string) (RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[tokenHash]
	if !ok || t.RevokedAt != nil || !t.ExpiresAt.After(time.Now()) {
		return RefreshToken{}, ErrNotFound
	}
	now := time.Now()
	t.RevokedAt = &now
	return *t, nil
}

func (m *Memory) GetRefreshToken(_ context.Context, tokenHash string) (RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tokens[tokenHash]
	if !ok {
		return RefreshToken{}, ErrNotFound
	}
	return *t, nil
}

func (m *Memory) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tokens[tokenHash]; ok && t.RevokedAt == nil {
		now := time.Now()
		t.RevokedAt = &now
	}
	return nil
}

func (m *Memory) RevokeAllRefreshTokens(_ context.Context, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, t := range m.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}
