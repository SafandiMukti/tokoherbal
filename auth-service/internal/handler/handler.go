// Package handler mengurus HTTP: membaca request, memanggil service, menulis response.
// Tidak ada logika bisnis di sini.
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"tokoherbal/auth-service/internal/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func New(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Router merangkai route dan middleware. Urutan dari luar ke dalam:
// request id -> recover -> logging -> CORS -> route.
func (h *Handler) Router(corsOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /register", h.register)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /refresh", h.refresh)
	mux.HandleFunc("POST /logout", h.logout)
	mux.HandleFunc("GET /verify", h.verify)

	var handler http.Handler = mux
	handler = cors(corsOrigin)(handler)
	handler = logging(h.log)(handler)
	handler = recoverer(h.log)(handler)
	handler = requestID(handler)
	return handler
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik
}

func toTokenResponse(p service.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  p.AccessToken,
		RefreshToken: p.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(p.AccessExpiresAt).Seconds()),
	}
}

// serverError mencatat detail error di log dan hanya mengirim pesan umum ke klien.
func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.log.Error(op, "error", err, "request_id", RequestIDFrom(r.Context()))
	writeError(w, http.StatusInternalServerError, "terjadi kesalahan")
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.svc.Ping(ctx); err != nil {
		h.log.Warn("health check gagal", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decodeJSON(w, r, &in) {
		return
	}
	u, err := h.svc.Register(r.Context(), in.Email, in.Password)

	var invalid *service.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Message)
	case errors.Is(err, service.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email sudah terdaftar")
	case err != nil:
		h.serverError(w, r, "register", err)
	default:
		writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "email": u.Email, "role": u.Role})
	}
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decodeJSON(w, r, &in) {
		return
	}
	pair, err := h.svc.Login(r.Context(), in.Email, in.Password)
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "email atau password salah")
	case err != nil:
		h.serverError(w, r, "login", err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, toTokenResponse(pair))
	}
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	pair, err := h.svc.Refresh(r.Context(), in.RefreshToken)
	switch {
	case errors.Is(err, service.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, "refresh token tidak valid")
	case err != nil:
		h.serverError(w, r, "refresh", err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, toTokenResponse(pair))
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var in refreshRequest
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.svc.Logout(r.Context(), in.RefreshToken); err != nil {
		h.serverError(w, r, "logout", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// verify dipanggil service lain untuk memeriksa access token.
func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	tok, ok := bearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "token tidak ada")
		return
	}
	id, err := h.svc.Verify(tok)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "token tidak valid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": id.UserID, "role": id.Role})
}
