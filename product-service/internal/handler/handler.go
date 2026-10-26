// Package handler mengurus HTTP: membaca request, memanggil service, menulis response.
// Tidak ada logika bisnis di sini.
package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tokoherbal/product-service/internal/auth"
	"tokoherbal/product-service/internal/domain"
	"tokoherbal/product-service/internal/reqid"
	"tokoherbal/product-service/internal/service"
	"tokoherbal/product-service/internal/upload"
)

// Verifier memeriksa header Authorization. Dipenuhi *auth.Client; di test diganti palsu.
type Verifier interface {
	Verify(ctx context.Context, authHeader string) (auth.Identity, error)
}

type Deps struct {
	Products *service.ProductService
	Orders   *service.OrderService
	Verifier Verifier
	Uploads  *upload.Store
	Log      *slog.Logger

	PingDB    func(context.Context) error // wajib: kegagalannya membuat /health = 503
	PingCache func(context.Context) error // opsional: kegagalannya hanya ditandai "down"

	RateLimiter     Counter // nil = tanpa rate limit
	RateLimitPerMin int
	TrustProxy      bool

	// InternalToken: rahasia bersama dengan server web (Next.js). Panggilan server-ke-server yang
	// membawanya lewat header X-Internal-Token dikecualikan dari rate limit, karena dari sisi
	// product-service semua pengunjung web terlihat berasal dari satu container. Kosong = tidak ada pengecualian.
	InternalToken string
}

type Handler struct {
	Deps
}

func New(d Deps) *Handler { return &Handler{Deps: d} }

// Router merangkai route dan middleware. Urutan dari luar ke dalam:
// request id -> recover -> logging -> CORS -> rate limit -> route.
func (h *Handler) Router(corsOrigin string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)

	// publik
	mux.HandleFunc("GET /products", h.listProducts)
	mux.HandleFunc("GET /products/{id}", h.getProduct)
	mux.HandleFunc("GET /categories", h.listCategories)
	mux.HandleFunc("GET /uploads/{name}", h.serveUpload)

	// admin
	mux.HandleFunc("POST /products", h.authed(true, h.createProduct))
	mux.HandleFunc("PUT /products/{id}", h.authed(true, h.updateProduct))
	mux.HandleFunc("DELETE /products/{id}", h.authed(true, h.deleteProduct))
	mux.HandleFunc("POST /products/{id}/image", h.authed(true, h.uploadImage))
	mux.HandleFunc("GET /admin/orders", h.authed(true, h.listAllOrders))

	// pelanggan saja: admin mengelola toko dan tidak berbelanja (403)
	mux.HandleFunc("GET /cart", h.shopper(h.getCart))
	mux.HandleFunc("PUT /cart/items/{product_id}", h.shopper(h.setCartItem))
	mux.HandleFunc("DELETE /cart/items/{product_id}", h.shopper(h.removeCartItem))
	mux.HandleFunc("POST /orders", h.shopper(h.createOrder))
	mux.HandleFunc("GET /orders", h.shopper(h.listMyOrders))

	// login (pelanggan untuk pesanannya sendiri, admin untuk pesanan siapa pun)
	mux.HandleFunc("GET /orders/{id}", h.authed(false, h.getOrder))
	mux.HandleFunc("PATCH /orders/{id}/status", h.authed(false, h.changeOrderStatus))

	var handler http.Handler = mux
	handler = rateLimit(h.RateLimiter, h.RateLimitPerMin, h.TrustProxy, h.InternalToken, h.Log)(handler)
	handler = cors(corsOrigin)(handler)
	handler = logging(h.Log)(handler)
	handler = recoverer(h.Log)(handler)
	handler = requestID(handler)
	return handler
}

// authed membungkus handler yang butuh login. Alurnya:
//
//	token kosong atau ditolak auth-service -> 401
//	auth-service tidak bisa dihubungi       -> 503 (bukan 500 dan bukan menggantung)
//	butuh admin tapi role bukan admin       -> 403
func (h *Handler) authed(adminOnly bool, next func(http.ResponseWriter, *http.Request, domain.Actor)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()

		id, err := h.Verifier.Verify(ctx, r.Header.Get("Authorization"))
		switch {
		case errors.Is(err, auth.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "perlu login")
			return
		case errors.Is(err, auth.ErrUnavailable):
			h.Log.Warn("auth-service tidak tersedia", "error", err, "request_id", reqid.From(r.Context()))
			w.Header().Set("Retry-After", "5")
			writeError(w, http.StatusServiceUnavailable, "layanan autentikasi sedang tidak tersedia, coba lagi sebentar")
			return
		case err != nil:
			h.serverError(w, r, "verify", err)
			return
		}

		if adminOnly && id.Role != domain.RoleAdmin {
			writeError(w, http.StatusForbidden, "hanya admin yang boleh melakukan ini")
			return
		}
		next(w, r, domain.Actor{UserID: id.UserID, Admin: id.Role == domain.RoleAdmin})
	}
}

// shopper membungkus handler belanja (keranjang dan pesanan). Syaratnya login, dan akun admin ditolak:
// admin mengelola produk dan status pesanan, tetapi tidak ikut membeli. Aturan ini ada di API,
// bukan hanya di tampilan, supaya tidak bisa dilewati dengan memanggil API langsung.
func (h *Handler) shopper(next func(http.ResponseWriter, *http.Request, domain.Actor)) http.HandlerFunc {
	return h.authed(false, func(w http.ResponseWriter, r *http.Request, who domain.Actor) {
		if who.Admin {
			writeError(w, http.StatusForbidden, "akun admin tidak dapat berbelanja; gunakan akun pelanggan")
			return
		}
		next(w, r, who)
	})
}

// fail menerjemahkan error service menjadi status HTTP. Detail error internal
// hanya masuk log; klien hanya menerima pesan yang aman.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	var invalid *domain.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Message)
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, domain.ErrOutOfStock):
		writeError(w, http.StatusConflict, "stok tidak mencukupi")
	case errors.Is(err, domain.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "perubahan status tidak diizinkan untuk status pesanan saat ini")
	case errors.Is(err, domain.ErrProductInUse):
		writeError(w, http.StatusConflict, "produk sudah pernah dipesan dan tidak bisa dihapus; ubah stoknya menjadi 0")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "tidak punya akses")
	default:
		h.serverError(w, r, op, err)
	}
}

func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	h.Log.Error(op, "error", err, "request_id", reqid.From(r.Context()))
	writeError(w, http.StatusInternalServerError, "terjadi kesalahan")
}

// health: database wajib sehat (kalau tidak, 503). Redis hanya optimasi, jadi
// bila Redis mati service tetap dianggap sehat tetapi ditandai "down".
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.PingDB != nil {
		if err := h.PingDB(ctx); err != nil {
			h.Log.Warn("health check: database gagal", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy", "database": "down"})
			return
		}
	}
	body := map[string]string{"status": "ok", "database": "ok"}
	if h.PingCache != nil {
		if err := h.PingCache(ctx); err != nil {
			body["redis"] = "down"
		} else {
			body["redis"] = "ok"
		}
	}
	writeJSON(w, http.StatusOK, body)
}

// serveUpload menyajikan gambar. Nama file dicocokkan dengan pola ketat sehingga
// tidak ada celah path traversal dan folder tidak bisa dilihat isinya.
func (h *Handler) serveUpload(w http.ResponseWriter, r *http.Request) {
	path, ok := h.Uploads.Path(r.PathValue("name"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

func isUploadURL(u string) (string, bool) {
	name, ok := strings.CutPrefix(u, "/uploads/")
	return name, ok && upload.ValidName(name)
}
