package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tokoherbal/product-service/internal/auth"
	"tokoherbal/product-service/internal/domain"
	"tokoherbal/product-service/internal/fakes"
	"tokoherbal/product-service/internal/service"
	"tokoherbal/product-service/internal/upload"
)

// fakeVerifier: token "admin" -> admin, "user" -> pelanggan id 1, "down" -> auth mati.
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, header string) (auth.Identity, error) {
	switch header {
	case "Bearer admin":
		return auth.Identity{UserID: 99, Role: domain.RoleAdmin}, nil
	case "Bearer user":
		return auth.Identity{UserID: 1, Role: domain.RoleCustomer}, nil
	case "Bearer user2":
		return auth.Identity{UserID: 2, Role: domain.RoleCustomer}, nil
	case "Bearer down":
		return auth.Identity{}, fmt.Errorf("%w: connection refused", auth.ErrUnavailable)
	}
	return auth.Identity{}, auth.ErrUnauthorized
}

type env struct {
	srv      http.Handler
	products *fakes.ProductStore
	counter  *fakes.Counter
	uploads  *upload.Store
}

func newEnv(t *testing.T, perMin int) *env {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ps, cache := fakes.NewProductStore(), fakes.NewCache()
	up, err := upload.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	counter := fakes.NewCounter()
	h := New(Deps{
		Products:        service.NewProductService(ps, cache, log),
		Orders:          service.NewOrderService(fakes.NewOrderStore(), fakes.NewCartStore(), cache, log),
		Verifier:        fakeVerifier{},
		Uploads:         up,
		Log:             log,
		PingDB:          func(context.Context) error { return nil },
		RateLimiter:     counter,
		RateLimitPerMin: perMin,
	})
	return &env{srv: h.Router("http://localhost:3000"), products: ps, counter: counter, uploads: up}
}

func (e *env) do(method, path, token, body string) *httptest.ResponseRecorder {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rd)
	req.RemoteAddr = "203.0.113.5:4444"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec
}

const validProduct = `{"name":"Jamu Kunyit","category":"jamu","price":15000,"stock":10}`

func TestHealth(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.do("GET", "/health", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestHealthDatabaseDown(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(Deps{Log: log, PingDB: func(context.Context) error { return errors.New("down") }})
	rec := httptest.NewRecorder()
	h.Router("*").ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rec.Code)
	}
}

func TestHealthRedisDownStillOK(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(Deps{
		Log:       log,
		PingDB:    func(context.Context) error { return nil },
		PingCache: func(context.Context) error { return errors.New("redis down") },
	})
	rec := httptest.NewRecorder()
	h.Router("*").ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"redis":"down"`) {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body)
	}
}

func TestPublicListNeedsNoToken(t *testing.T) {
	e := newEnv(t, 0)
	e.do("POST", "/products", "admin", validProduct)

	rec := e.do("GET", "/products", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var res domain.ListResult
	json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Total != 1 || len(res.Items) != 1 || res.Page != 1 {
		t.Fatalf("res = %+v", res)
	}

	if rec := e.do("GET", "/products?sort=bogus", "", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("sort tidak valid: code = %d", rec.Code)
	}
}

func TestEmptyListIsArrayNotNull(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.do("GET", "/products", "", "")
	if !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Fatalf("body = %s", rec.Body)
	}
}

func TestAdminEndpointsAuthorization(t *testing.T) {
	e := newEnv(t, 0)
	cases := []struct {
		name, token string
		want        int
	}{
		{"tanpa token", "", http.StatusUnauthorized},
		{"token salah", "ngawur", http.StatusUnauthorized},
		{"pelanggan biasa", "user", http.StatusForbidden},
		{"auth-service mati", "down", http.StatusServiceUnavailable},
		{"admin", "admin", http.StatusCreated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := e.do("POST", "/products", c.token, validProduct)
			if rec.Code != c.want {
				t.Fatalf("code = %d, want %d, body = %s", rec.Code, c.want, rec.Body)
			}
			if c.want == http.StatusServiceUnavailable && rec.Header().Get("Retry-After") == "" {
				t.Error("503 sebaiknya menyertakan Retry-After")
			}
		})
	}
}

func TestCreateProductRejectsBadBody(t *testing.T) {
	e := newEnv(t, 0)
	bodies := []string{
		`{"name":"x","category":"jamu","price":1,"stock":1,"role":"admin"}`, // field asing
		`{"name":"","category":"jamu","price":1,"stock":1}`,
		`{"name":"x","category":"jamu","price":-5,"stock":1}`,
		`bukan json`,
		validProduct + validProduct,
	}
	for _, b := range bodies {
		if rec := e.do("POST", "/products", "admin", b); rec.Code != http.StatusBadRequest {
			t.Errorf("body %q: code = %d, want 400", b, rec.Code)
		}
	}
}

func TestProductCRUD(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.do("POST", "/products", "admin", validProduct)
	var p domain.Product
	json.Unmarshal(rec.Body.Bytes(), &p)
	if p.ID == 0 || rec.Header().Get("Location") != fmt.Sprintf("/products/%d", p.ID) {
		t.Fatalf("p=%+v location=%q", p, rec.Header().Get("Location"))
	}

	path := fmt.Sprintf("/products/%d", p.ID)
	if rec := e.do("GET", path, "", ""); rec.Code != 200 {
		t.Fatalf("get: %d", rec.Code)
	}
	if rec := e.do("PUT", path, "admin", `{"name":"Baru","category":"jamu","price":1,"stock":2}`); rec.Code != 200 {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("DELETE", path, "user", ""); rec.Code != 403 {
		t.Fatalf("delete oleh pelanggan: %d", rec.Code)
	}
	if rec := e.do("DELETE", path, "admin", ""); rec.Code != 204 {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec := e.do("GET", path, "", ""); rec.Code != 404 {
		t.Fatalf("get setelah delete: %d", rec.Code)
	}
	if rec := e.do("GET", "/products/abc", "", ""); rec.Code != 400 {
		t.Fatalf("id bukan angka: %d", rec.Code)
	}
}

func TestOrderFlowAndOwnership(t *testing.T) {
	e := newEnv(t, 0)

	if rec := e.do("PUT", "/cart/items/1", "user", `{"qty":2}`); rec.Code != 200 {
		t.Fatalf("cart: %d %s", rec.Code, rec.Body)
	}
	rec := e.do("POST", "/orders", "user", `{}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("order dari keranjang: %d %s", rec.Code, rec.Body)
	}
	var o domain.Order
	json.Unmarshal(rec.Body.Bytes(), &o)
	path := fmt.Sprintf("/orders/%d", o.ID)

	if rec := e.do("GET", path, "user2", ""); rec.Code != 404 {
		t.Fatalf("pesanan orang lain harus 404, dapat %d", rec.Code)
	}
	if rec := e.do("PATCH", path+"/status", "user", `{"status":"paid"}`); rec.Code != 403 {
		t.Fatalf("pelanggan tidak boleh set paid: %d", rec.Code)
	}
	if rec := e.do("PATCH", path+"/status", "admin", `{"status":"paid"}`); rec.Code != 200 {
		t.Fatalf("admin set paid: %d", rec.Code)
	}
	if rec := e.do("PATCH", path+"/status", "admin", `{"status":"paid"}`); rec.Code != 409 {
		t.Fatalf("paid dua kali harus 409: %d", rec.Code)
	}
	if rec := e.do("GET", "/orders", "user", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"status":"paid"`) {
		t.Fatalf("riwayat: %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("GET", "/admin/orders", "user", ""); rec.Code != 403 {
		t.Fatalf("/admin/orders oleh pelanggan: %d", rec.Code)
	}
	if rec := e.do("GET", "/admin/orders?status=paid", "admin", ""); rec.Code != 200 {
		t.Fatalf("/admin/orders: %d", rec.Code)
	}
}

func TestRateLimit(t *testing.T) {
	e := newEnv(t, 3)
	for i := 1; i <= 3; i++ {
		if rec := e.do("GET", "/products", "", ""); rec.Code != 200 {
			t.Fatalf("request %d: %d", i, rec.Code)
		}
	}
	rec := e.do("GET", "/products", "", "")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("code = %d retry-after = %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	if rec := e.do("GET", "/health", "", ""); rec.Code != 200 {
		t.Fatalf("/health tidak boleh kena rate limit: %d", rec.Code)
	}
}

// newEnvToken seperti newEnv, tetapi dengan InternalToken diisi.
func newEnvToken(t *testing.T, perMin int, token string) *env {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ps, cache := fakes.NewProductStore(), fakes.NewCache()
	up, err := upload.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	counter := fakes.NewCounter()
	h := New(Deps{
		Products:        service.NewProductService(ps, cache, log),
		Orders:          service.NewOrderService(fakes.NewOrderStore(), fakes.NewCartStore(), cache, log),
		Verifier:        fakeVerifier{},
		Uploads:         up,
		Log:             log,
		PingDB:          func(context.Context) error { return nil },
		RateLimiter:     counter,
		RateLimitPerMin: perMin,
		InternalToken:   token,
	})
	return &env{srv: h.Router("http://localhost:3000"), products: ps, counter: counter, uploads: up}
}

func (e *env) getWithInternal(header string) int {
	req := httptest.NewRequest("GET", "/products", nil)
	req.RemoteAddr = "203.0.113.5:4444"
	if header != "" {
		req.Header.Set("X-Internal-Token", header)
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec.Code
}

// Admin mengelola toko dan tidak berbelanja: keranjang dan pesanan baru ditolak (403),
// tetapi admin tetap boleh melihat dan mengubah status pesanan pelanggan.
func TestAdminCannotShop(t *testing.T) {
	e := newEnv(t, 0)
	// fake store kosong; produk dibuat lewat API seperti di tes lain (id pertama = 1)
	if rec := e.do("POST", "/products", "admin", validProduct); rec.Code != 201 {
		t.Fatalf("membuat produk = %d: %s", rec.Code, rec.Body.String())
	}
	before, err := e.products.Get(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct{ method, path, body string }{
		{"GET", "/cart", ""},
		{"PUT", "/cart/items/1", `{"qty":1}`},
		{"DELETE", "/cart/items/1", ""},
		{"POST", "/orders", `{"items":[{"product_id":1,"qty":1}]}`},
		{"GET", "/orders", ""},
	} {
		if rec := e.do(c.method, c.path, "admin", c.body); rec.Code != 403 {
			t.Errorf("admin %s %s = %d, want 403", c.method, c.path, rec.Code)
		}
	}
	// stok tidak berubah: penolakan terjadi sebelum menyentuh data
	if p, _ := e.products.Get(context.Background(), 1); p.Stock != before.Stock {
		t.Errorf("stok = %d, want %d", p.Stock, before.Stock)
	}

	// pelanggan tetap bisa; admin tetap bisa mengelola pesanan pelanggan itu
	if rec := e.do("POST", "/orders", "user", `{"items":[{"product_id":1,"qty":1}]}`); rec.Code != 201 {
		t.Fatalf("pelanggan memesan = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := e.do("PATCH", "/orders/1/status", "admin", `{"status":"paid"}`); rec.Code != 200 {
		t.Errorf("admin menandai dibayar = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := e.do("GET", "/admin/orders", "admin", ""); rec.Code != 200 {
		t.Errorf("admin melihat semua pesanan = %d", rec.Code)
	}
	if rec := e.do("GET", "/orders/1", "admin", ""); rec.Code != 200 {
		t.Errorf("admin melihat detail pesanan = %d", rec.Code)
	}
}

func TestRateLimitInternalTokenBypass(t *testing.T) {
	const secret = "rahasia-internal-1234567890"
	e := newEnvToken(t, 2, secret)

	// token benar: tidak pernah kena batas, jauh melebihi 2 per menit
	for i := 0; i < 10; i++ {
		if code := e.getWithInternal(secret); code != 200 {
			t.Fatalf("token benar, request %d: %d", i, code)
		}
	}
	// tanpa token dan token salah tetap dibatasi (dan token benar tadi tidak memakai jatah IP yang sama)
	for _, h := range []string{"", "salah", secret + "x", secret[:len(secret)-1]} {
		e2 := newEnvToken(t, 2, secret)
		e2.getWithInternal(h)
		e2.getWithInternal(h)
		if code := e2.getWithInternal(h); code != http.StatusTooManyRequests {
			t.Fatalf("header %q seharusnya tetap dibatasi, dapat %d", h, code)
		}
	}
}

func TestRateLimitNoInternalTokenConfigured(t *testing.T) {
	// fitur mati: header apa pun, termasuk kosong, tidak boleh mengecualikan
	e := newEnvToken(t, 2, "")
	for _, h := range []string{"", "apa-saja"} {
		e2 := newEnvToken(t, 2, "")
		e2.getWithInternal(h)
		e2.getWithInternal(h)
		if code := e2.getWithInternal(h); code != http.StatusTooManyRequests {
			t.Fatalf("header %q, fitur mati, dapat %d", h, code)
		}
	}
	_ = e
}

func TestRateLimitRedisDownAllowsRequests(t *testing.T) {
	e := newEnv(t, 1)
	e.counter.Err = errors.New("redis down")
	for i := 0; i < 5; i++ {
		if rec := e.do("GET", "/products", "", ""); rec.Code != 200 {
			t.Fatalf("request %d: %d (Redis mati tidak boleh memblokir pengguna)", i, rec.Code)
		}
	}
}

func TestCORSPreflight(t *testing.T) {
	e := newEnv(t, 0)
	req := httptest.NewRequest("OPTIONS", "/orders/1/status", nil)
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	h := rec.Header()
	if rec.Code != 204 || h.Get("Access-Control-Allow-Origin") != "http://localhost:3000" ||
		!strings.Contains(h.Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Fatalf("code=%d headers=%v", rec.Code, h)
	}
}

func TestRequestIDPropagation(t *testing.T) {
	e := newEnv(t, 0)
	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Request-ID", "abc-123")
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("request id = %q", got)
	}

	req = httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Request-ID", "jahat\nbaris-palsu")
	rec = httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got == "" || strings.ContainsAny(got, "\n ") {
		t.Fatalf("request id berbahaya harus diganti, dapat %q", got)
	}
}

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(content)
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func TestUploadImage(t *testing.T) {
	e := newEnv(t, 0)
	rec := e.do("POST", "/products", "admin", validProduct)
	var p domain.Product
	json.Unmarshal(rec.Body.Bytes(), &p)
	path := fmt.Sprintf("/products/%d/image", p.ID)

	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

	post := func(token, field, filename string, content []byte) *httptest.ResponseRecorder {
		body, ctype := multipartBody(t, field, filename, content)
		req := httptest.NewRequest("POST", path, body)
		req.Header.Set("Content-Type", ctype)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		e.srv.ServeHTTP(rec, req)
		return rec
	}

	if rec := post("user", "image", "a.png", png); rec.Code != 403 {
		t.Fatalf("pelanggan: %d", rec.Code)
	}
	if rec := post("admin", "image", "evil.png", []byte("<html>bukan gambar</html>")); rec.Code != 400 {
		t.Fatalf("bukan gambar: %d %s", rec.Code, rec.Body)
	}
	if rec := post("admin", "file", "a.png", png); rec.Code != 400 {
		t.Fatalf("field salah: %d", rec.Code)
	}

	rec = post("admin", "image", "../../evil.png", png) // nama dari klien diabaikan
	if rec.Code != 200 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	json.Unmarshal(rec.Body.Bytes(), &p)
	if !strings.HasPrefix(p.ImageURL, "/uploads/") {
		t.Fatalf("image_url = %q", p.ImageURL)
	}

	// file bisa diambil lewat /uploads/{name}
	got := e.do("GET", p.ImageURL, "", "")
	if got.Code != 200 || got.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("serve: %d %v", got.Code, got.Header())
	}

	// unggah lagi: file lama harus terhapus
	oldName := strings.TrimPrefix(p.ImageURL, "/uploads/")
	post("admin", "image", "b.png", png)
	if _, ok := e.uploads.Path(oldName); !ok {
		t.Fatal("nama lama seharusnya valid")
	}
	if rec := e.do("GET", "/uploads/"+oldName, "", ""); rec.Code != 404 {
		t.Fatalf("file lama harus terhapus, code = %d", rec.Code)
	}
}

func TestServeUploadRejectsTraversal(t *testing.T) {
	e := newEnv(t, 0)
	for _, p := range []string{"/uploads/..%2F..%2Fetc%2Fpasswd", "/uploads/passwd", "/uploads/"} {
		if rec := e.do("GET", p, "", ""); rec.Code != 404 {
			t.Errorf("%s: code = %d, want 404", p, rec.Code)
		}
	}
}
