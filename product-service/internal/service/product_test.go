package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"tokoherbal/product-service/internal/domain"
	"tokoherbal/product-service/internal/fakes"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newProductSvc() (*ProductService, *fakes.ProductStore, *fakes.Cache) {
	st, c := fakes.NewProductStore(), fakes.NewCache()
	return NewProductService(st, c, quietLog()), st, c
}

func TestValidateInput(t *testing.T) {
	ok := domain.ProductInput{Name: "Jamu", Category: "Jamu", Price: 15000, Stock: 3}
	cases := []struct {
		name    string
		mutate  func(*domain.ProductInput)
		wantErr bool
	}{
		{"valid", func(*domain.ProductInput) {}, false},
		{"nama kosong", func(p *domain.ProductInput) { p.Name = "   " }, true},
		{"nama terlalu panjang", func(p *domain.ProductInput) { p.Name = string(make([]rune, 121)) + "x" }, true},
		{"kategori kosong", func(p *domain.ProductInput) { p.Category = "" }, true},
		{"harga negatif", func(p *domain.ProductInput) { p.Price = -1 }, true},
		{"harga nol boleh", func(p *domain.ProductInput) { p.Price = 0 }, false},
		{"stok negatif", func(p *domain.ProductInput) { p.Stock = -5 }, true},
		{"stok kebesaran", func(p *domain.ProductInput) { p.Stock = 2_000_000 }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := ok
			c.mutate(&in)
			_, err := validateInput(in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			var invalid *domain.InvalidInputError
			if err != nil && !errors.As(err, &invalid) {
				t.Fatalf("error harus bertipe *InvalidInputError, dapat %T", err)
			}
		})
	}
}

func TestValidateInputNormalizes(t *testing.T) {
	got, err := validateInput(domain.ProductInput{Name: "  Jamu  ", Category: " JAMU "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Jamu" || got.Category != "jamu" {
		t.Fatalf("hasil = %+v", got)
	}
}

func TestNormalizeList(t *testing.T) {
	p, err := normalizeList(domain.ListParams{Page: -3, Limit: 9999})
	if err != nil {
		t.Fatal(err)
	}
	if p.Page != 1 || p.Limit != maxLimit || p.Sort != domain.SortNewest {
		t.Fatalf("hasil = %+v", p)
	}
	if _, err := normalizeList(domain.ListParams{Sort: "id; DROP TABLE products"}); err == nil {
		t.Fatal("sort tidak dikenal harus ditolak")
	}
}

func TestListCacheAside(t *testing.T) {
	svc, st, _ := newProductSvc()
	ctx := context.Background()
	if _, err := svc.Create(ctx, domain.ProductInput{Name: "Jamu", Category: "jamu", Price: 1000, Stock: 1}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		res, err := svc.List(ctx, domain.ListParams{})
		if err != nil || len(res.Items) != 1 {
			t.Fatalf("List #%d: %v %+v", i, err, res)
		}
	}
	if st.ListCalls != 1 {
		t.Fatalf("database dipanggil %d kali, seharusnya 1 (sisanya dari cache)", st.ListCalls)
	}

	// parameter berbeda = key berbeda = miss baru
	if _, err := svc.List(ctx, domain.ListParams{Query: "jam"}); err != nil {
		t.Fatal(err)
	}
	if st.ListCalls != 2 {
		t.Fatalf("ListCalls = %d, want 2", st.ListCalls)
	}
}

func TestMutationInvalidatesCache(t *testing.T) {
	svc, st, c := newProductSvc()
	ctx := context.Background()
	p, _ := svc.Create(ctx, domain.ProductInput{Name: "Jamu", Category: "jamu", Price: 1000, Stock: 1})
	svc.List(ctx, domain.ListParams{}) // isi cache
	before := st.ListCalls

	if _, err := svc.Update(ctx, p.ID, domain.ProductInput{Name: "Jamu Baru", Category: "jamu", Price: 2000, Stock: 1}); err != nil {
		t.Fatal(err)
	}
	res, _ := svc.List(ctx, domain.ListParams{})
	if st.ListCalls != before+1 {
		t.Fatal("setelah update, List harus mengambil ulang dari database")
	}
	if res.Items[0].Name != "Jamu Baru" {
		t.Fatalf("data basi: %+v", res.Items[0])
	}
	if len(c.Deletes) == 0 || c.Deletes[len(c.Deletes)-1] != cachePrefix {
		t.Fatalf("DeletePrefix tidak dipanggil dengan %q: %v", cachePrefix, c.Deletes)
	}
}

func TestCacheDownFallsBackToDatabase(t *testing.T) {
	svc, st, c := newProductSvc()
	ctx := context.Background()
	svc.Create(ctx, domain.ProductInput{Name: "Jamu", Category: "jamu", Price: 1000, Stock: 1})
	c.GetErr = errors.New("redis: connection refused")
	c.DeleteErr = errors.New("redis: connection refused")

	res, err := svc.List(ctx, domain.ListParams{})
	if err != nil {
		t.Fatalf("Redis mati tidak boleh menggagalkan request: %v", err)
	}
	if len(res.Items) != 1 || st.ListCalls != 1 {
		t.Fatalf("res=%+v listCalls=%d", res, st.ListCalls)
	}
	// invalidasi yang gagal juga tidak boleh menggagalkan penulisan
	if _, err := svc.Create(ctx, domain.ProductInput{Name: "Madu", Category: "madu", Price: 1, Stock: 1}); err != nil {
		t.Fatalf("Create gagal karena cache: %v", err)
	}
}

func TestGetNotFoundIsNotCached(t *testing.T) {
	svc, st, _ := newProductSvc()
	for i := 0; i < 2; i++ {
		if _, err := svc.Get(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	}
	if st.GetCalls != 2 {
		t.Fatalf("GetCalls = %d, want 2", st.GetCalls)
	}
}
