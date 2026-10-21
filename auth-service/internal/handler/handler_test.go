package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tokoherbal/auth-service/internal/service"
	"tokoherbal/auth-service/internal/store"
	"tokoherbal/auth-service/internal/token"
)

const testSecret = "rahasia-untuk-test-minimal-32-karakter!!"

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	svc, err := service.New(store.NewMemory(),
		token.NewManager(testSecret, 15*time.Minute),
		service.Config{BcryptCost: bcrypt.MinCost})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(svc, log).Router("http://localhost:3000")
}

func do(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body bukan JSON yang diharapkan: %v\n%s", err, rec.Body.String())
	}
	return v
}

func TestHealth(t *testing.T) {
	rec := do(t, newRouter(t), http.MethodGet, "/health", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestFullFlow(t *testing.T) {
	h := newRouter(t)

	// register
	rec := do(t, h, http.MethodPost, "/register", `{"email":"andi@example.com","password":"rahasia123"}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d: %s", rec.Code, rec.Body)
	}
	if got := decode[map[string]any](t, rec); got["role"] != "customer" {
		t.Fatalf("role = %v", got["role"])
	}

	// login
	rec = do(t, h, http.MethodPost, "/login", `{"email":"andi@example.com","password":"rahasia123"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("respons token harus Cache-Control: no-store")
	}
	tokens := decode[tokenResponse](t, rec)
	if tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.TokenType != "Bearer" || tokens.ExpiresIn <= 0 {
		t.Fatalf("tokens = %+v", tokens)
	}

	// verify
	rec = do(t, h, http.MethodGet, "/verify", "", map[string]string{"Authorization": "Bearer " + tokens.AccessToken})
	if rec.Code != http.StatusOK {
		t.Fatalf("verify: want 200, got %d", rec.Code)
	}
	if got := decode[map[string]any](t, rec); got["role"] != "customer" || got["user_id"] != float64(1) {
		t.Fatalf("verify = %v", got)
	}

	// refresh -> token lama tidak berlaku lagi
	rec = do(t, h, http.MethodPost, "/refresh", `{"refresh_token":"`+tokens.RefreshToken+`"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh: want 200, got %d", rec.Code)
	}
	newTokens := decode[tokenResponse](t, rec)
	rec = do(t, h, http.MethodPost, "/refresh", `{"refresh_token":"`+tokens.RefreshToken+`"}`, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token lama: want 401, got %d", rec.Code)
	}

	// logout (token baru sudah ikut dicabut oleh reuse detection di atas, tetap 204)
	rec = do(t, h, http.MethodPost, "/logout", `{"refresh_token":"`+newTokens.RefreshToken+`"}`, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: want 204, got %d", rec.Code)
	}
}

func TestErrorResponses(t *testing.T) {
	h := newRouter(t)
	do(t, h, http.MethodPost, "/register", `{"email":"andi@example.com","password":"rahasia123"}`, nil)

	cases := []struct {
		name    string
		method  string
		path    string
		body    string
		headers map[string]string
		want    int
	}{
		{"register: JSON rusak", "POST", "/register", `{bukan json`, nil, 400},
		{"register: tidak boleh memilih role", "POST", "/register", `{"email":"x@example.com","password":"rahasia123","role":"admin"}`, nil, 400},
		{"register: dua objek JSON", "POST", "/register", `{"email":"x@example.com","password":"rahasia123"}{}`, nil, 400},
		{"register: password pendek", "POST", "/register", `{"email":"x@example.com","password":"pendek"}`, nil, 400},
		{"register: email duplikat", "POST", "/register", `{"email":"andi@example.com","password":"rahasia123"}`, nil, 409},
		{"login: password salah", "POST", "/login", `{"email":"andi@example.com","password":"salah-banget"}`, nil, 401},
		{"login: email tidak ada", "POST", "/login", `{"email":"nope@example.com","password":"rahasia123"}`, nil, 401},
		{"verify: tanpa header", "GET", "/verify", "", nil, 401},
		{"verify: token ngawur", "GET", "/verify", "", map[string]string{"Authorization": "Bearer ngawur"}, 401},
		{"verify: bukan skema Bearer", "GET", "/verify", "", map[string]string{"Authorization": "Basic abc"}, 401},
		{"refresh: token ngawur", "POST", "/refresh", `{"refresh_token":"ngawur"}`, nil, 401},
		{"method salah", "GET", "/login", "", nil, 405},
		{"route tidak ada", "GET", "/tidak-ada", "", nil, 404},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, h, c.method, c.path, c.body, c.headers)
			if rec.Code != c.want {
				t.Fatalf("want %d, got %d: %s", c.want, rec.Code, rec.Body)
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	rec := do(t, newRouter(t), http.MethodOptions, "/login", "", map[string]string{
		"Origin":                        "http://localhost:3000",
		"Access-Control-Request-Method": "POST",
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Allow-Origin = %q", got)
	}
}

func TestRequestIDPropagation(t *testing.T) {
	h := newRouter(t)

	rec := do(t, h, http.MethodGet, "/health", "", map[string]string{"X-Request-ID": "abc-123"})
	if got := rec.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("request id dari pemanggil harus dipertahankan, got %q", got)
	}

	rec = do(t, h, http.MethodGet, "/health", "", nil)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id harus dibuat bila tidak ada")
	}
}
