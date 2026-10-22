package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"tokoherbal/product-service/internal/reqid"
)

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// validRequestID hanya menerima huruf, angka, '-' dan '_' (maks 64 karakter),
// supaya header dari luar tidak bisa menyusupkan baris palsu ke log.
func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			return false
		}
	}
	return true
}

// requestID memakai X-Request-ID dari pemanggil bila ada, atau membuat yang baru.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(reqid.With(r.Context(), id)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			if r.URL.Path == "/health" {
				level = slog.LevelDebug
			}
			log.Log(r.Context(), level, "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", reqid.From(r.Context()),
			)
		})
	}
}

func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if v == http.ErrAbortHandler {
						panic(v)
					}
					log.Error("panic",
						"error", v,
						"request_id", reqid.From(r.Context()),
						"stack", string(debug.Stack()))
					writeError(w, http.StatusInternalServerError, "terjadi kesalahan")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// cors hanya mengizinkan SATU origin (alamat web Next.js), bukan tanda bintang.
// Berbeda dengan auth-service, di sini PATCH ikut diizinkan (ubah status pesanan).
func cors(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Max-Age", "600")
			h.Add("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Counter menghitung permintaan dalam satu jendela waktu (diimplementasikan Redis).
type Counter interface {
	Incr(ctx context.Context, key string, window time.Duration) (int64, error)
}

// rateLimit membatasi jumlah permintaan per IP per menit (fixed window).
// Bila Redis bermasalah, permintaan DIIZINKAN: rate limit adalah pengaman
// tambahan, bukan alasan untuk menolak pengguna yang sah.
func rateLimit(c Counter, perMinute int, trustProxy bool, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if c == nil || perMinute <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			n, err := c.Incr(r.Context(), "rl:"+clientIP(r, trustProxy), time.Minute)
			if err != nil {
				log.Warn("rate limit dilewati, Redis bermasalah", "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if n > int64(perMinute) {
				w.Header().Set("Retry-After", strconv.Itoa(60))
				writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi sebentar")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP mengambil alamat klien. X-Forwarded-For hanya dipercaya bila service
// memang berada di belakang reverse proxy kita sendiri (TRUST_PROXY=true);
// kalau tidak, klien bisa memalsukannya untuk menghindari rate limit.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
