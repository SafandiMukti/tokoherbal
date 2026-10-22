package service

import (
	"context"
	"time"

	"tokoherbal/product-service/internal/domain"
)

// Interface di bawah didefinisikan DI SISI PEMAKAI (package service), bukan di
// package store. Ini idiom Go: "terima interface, kembalikan struct". Akibatnya
// service dan handler bisa diuji dengan implementasi palsu (internal/fakes)
// tanpa perlu PostgreSQL maupun Redis.

type ProductStore interface {
	List(ctx context.Context, p domain.ListParams) ([]domain.Product, int, error)
	Get(ctx context.Context, id int) (domain.Product, error)
	Create(ctx context.Context, in domain.ProductInput) (domain.Product, error)
	Update(ctx context.Context, id int, in domain.ProductInput) (domain.Product, error)
	Delete(ctx context.Context, id int) error
	// SetImage mengganti gambar dan mengembalikan URL gambar yang lama.
	SetImage(ctx context.Context, id int, url string) (oldURL string, err error)
	Categories(ctx context.Context) ([]string, error)
}

type CartStore interface {
	Get(ctx context.Context, userID int) (domain.Cart, error)
	// Set mengisi atau mengganti jumlah. Mengembalikan domain.ErrNotFound bila produk tidak ada.
	Set(ctx context.Context, userID, productID, qty int) error
	Remove(ctx context.Context, userID, productID int) error
}

type OrderStore interface {
	// Create membuat pesanan dalam SATU transaksi: kurangi stok, buat order,
	// buat item, dan (bila clearCart) kosongkan keranjang. Gagal satu, batal semua.
	Create(ctx context.Context, userID int, items []domain.OrderItemInput, clearCart bool) (int, error)
	Get(ctx context.Context, id int) (domain.Order, error)
	ListByUser(ctx context.Context, userID, limit, offset int) ([]domain.Order, error)
	ListAll(ctx context.Context, status string, limit, offset int) ([]domain.Order, error)
	// SetStatus mengubah status hanya jika status saat ini ada di allowedFrom.
	// Bila to == cancelled, stok dikembalikan dalam transaksi yang sama.
	SetStatus(ctx context.Context, id int, allowedFrom []string, to string) error
}

// Cache dipakai untuk cache-aside. Error cache TIDAK boleh menggagalkan request:
// pemanggil mencatat peringatan lalu lanjut ke database.
type Cache interface {
	Get(ctx context.Context, key string) (val []byte, found bool, err error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	// DeletePrefix menghapus semua key berawalan prefix (invalidasi).
	DeletePrefix(ctx context.Context, prefix string) error
}
