// Package service berisi logika bisnis product-service: validasi, cache-aside,
// keranjang, dan aturan status pesanan. Tidak tahu apa-apa tentang HTTP atau SQL.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"tokoherbal/product-service/internal/domain"
)

const (
	cachePrefix   = "products:" // semua key cache produk berawalan ini
	cacheTTL      = 60 * time.Second
	defaultLimit  = 12
	maxLimit      = 50
	maxQueryChars = 100
)

type ProductService struct {
	store ProductStore
	cache Cache
	log   *slog.Logger
}

func NewProductService(s ProductStore, c Cache, log *slog.Logger) *ProductService {
	return &ProductService{store: s, cache: c, log: log}
}

func normalizeList(p domain.ListParams) (domain.ListParams, error) {
	p.Query = strings.TrimSpace(p.Query)
	p.Category = strings.TrimSpace(p.Category)
	if utf8.RuneCountInString(p.Query) > maxQueryChars {
		return p, domain.Invalid("kata kunci terlalu panjang")
	}
	if utf8.RuneCountInString(p.Category) > 50 {
		return p, domain.Invalid("kategori terlalu panjang")
	}
	switch p.Sort {
	case "":
		p.Sort = domain.SortNewest
	case domain.SortNewest, domain.SortPriceAsc, domain.SortPriceDesc, domain.SortName:
	default:
		return p, domain.Invalid("sort tidak dikenal (newest, price_asc, price_desc, name)")
	}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
	return p, nil
}

// listKey membuat key cache dari parameter. Teks pencarian di-hash supaya key
// pendek dan aman walau berisi karakter aneh.
func listKey(p domain.ListParams) string {
	raw := fmt.Sprintf("%s|%s|%s|%d|%d", p.Query, p.Category, p.Sort, p.Page, p.Limit)
	sum := sha256.Sum256([]byte(raw))
	return cachePrefix + "list:" + hex.EncodeToString(sum[:8])
}

// List memakai pola cache-aside: cek cache, kalau tidak ada ambil dari database
// lalu simpan ke cache dengan TTL.
func (s *ProductService) List(ctx context.Context, p domain.ListParams) (domain.ListResult, error) {
	p, err := normalizeList(p)
	if err != nil {
		return domain.ListResult{}, err
	}
	key := listKey(p)

	var res domain.ListResult
	if s.cacheGet(ctx, key, &res) {
		s.log.Info("cache HIT", "key", key)
		return res, nil
	}
	s.log.Info("cache MISS", "key", key)

	items, total, err := s.store.List(ctx, p)
	if err != nil {
		return domain.ListResult{}, err
	}
	if items == nil {
		items = []domain.Product{} // JSON "[]", bukan "null"
	}
	res = domain.ListResult{Items: items, Total: total, Page: p.Page, Limit: p.Limit}
	s.cacheSet(ctx, key, res)
	return res, nil
}

func (s *ProductService) Get(ctx context.Context, id int) (domain.Product, error) {
	key := cachePrefix + "detail:" + strconv.Itoa(id)
	var p domain.Product
	if s.cacheGet(ctx, key, &p) {
		return p, nil
	}
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Product{}, err // not found tidak di-cache
	}
	s.cacheSet(ctx, key, p)
	return p, nil
}

func (s *ProductService) Categories(ctx context.Context) ([]string, error) {
	key := cachePrefix + "categories"
	var cats []string
	if s.cacheGet(ctx, key, &cats) {
		return cats, nil
	}
	cats, err := s.store.Categories(ctx)
	if err != nil {
		return nil, err
	}
	if cats == nil {
		cats = []string{}
	}
	s.cacheSet(ctx, key, cats)
	return cats, nil
}

func (s *ProductService) Create(ctx context.Context, in domain.ProductInput) (domain.Product, error) {
	in, err := validateInput(in)
	if err != nil {
		return domain.Product{}, err
	}
	p, err := s.store.Create(ctx, in)
	if err != nil {
		return domain.Product{}, err
	}
	s.invalidate(ctx)
	return p, nil
}

func (s *ProductService) Update(ctx context.Context, id int, in domain.ProductInput) (domain.Product, error) {
	in, err := validateInput(in)
	if err != nil {
		return domain.Product{}, err
	}
	p, err := s.store.Update(ctx, id, in)
	if err != nil {
		return domain.Product{}, err
	}
	s.invalidate(ctx)
	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, id int) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	s.invalidate(ctx)
	return nil
}

// SetImage menyimpan URL gambar baru dan mengembalikan URL yang lama (untuk dihapus filenya).
func (s *ProductService) SetImage(ctx context.Context, id int, url string) (string, error) {
	old, err := s.store.SetImage(ctx, id, url)
	if err != nil {
		return "", err
	}
	s.invalidate(ctx)
	return old, nil
}

func validateInput(in domain.ProductInput) (domain.ProductInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Category = strings.ToLower(strings.TrimSpace(in.Category))
	in.Description = strings.TrimSpace(in.Description)

	switch {
	case in.Name == "" || utf8.RuneCountInString(in.Name) > 120:
		return in, domain.Invalid("nama wajib diisi, maksimal 120 karakter")
	case in.Category == "" || utf8.RuneCountInString(in.Category) > 50:
		return in, domain.Invalid("kategori wajib diisi, maksimal 50 karakter")
	case utf8.RuneCountInString(in.Description) > 2000:
		return in, domain.Invalid("deskripsi maksimal 2000 karakter")
	case in.Price < 0 || in.Price > 1_000_000_000:
		return in, domain.Invalid("harga harus antara 0 dan 1.000.000.000")
	case in.Stock < 0 || in.Stock > 1_000_000:
		return in, domain.Invalid("stok harus antara 0 dan 1.000.000")
	}
	return in, nil
}

// --- helper cache: semua error cache hanya dicatat, tidak pernah dikembalikan ---

func (s *ProductService) cacheGet(ctx context.Context, key string, dst any) bool {
	raw, found, err := s.cache.Get(ctx, key)
	if err != nil {
		s.log.Warn("cache get gagal, lanjut ke database", "key", key, "error", err)
		return false
	}
	if !found {
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		s.log.Warn("isi cache rusak, diabaikan", "key", key, "error", err)
		return false
	}
	return true
}

func (s *ProductService) cacheSet(ctx context.Context, key string, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, key, raw, cacheTTL); err != nil {
		s.log.Warn("cache set gagal", "key", key, "error", err)
	}
}

// invalidate dipanggil setiap data produk atau stok berubah.
func (s *ProductService) invalidate(ctx context.Context) {
	invalidate(ctx, s.cache, s.log)
}

func invalidate(ctx context.Context, c Cache, log *slog.Logger) {
	if err := c.DeletePrefix(ctx, cachePrefix); err != nil {
		// TTL pendek menjadi jaring pengaman: data basi paling lama sebatas TTL
		log.Warn("invalidasi cache gagal", "error", err)
	}
}
