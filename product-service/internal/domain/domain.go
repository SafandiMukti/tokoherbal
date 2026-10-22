// Package domain berisi tipe data dan error yang dipakai di semua lapisan
// product-service. Hanya memakai pustaka standar, tanpa dependensi ke HTTP atau SQL.
package domain

import (
	"errors"
	"time"
)

const (
	RoleAdmin    = "admin"
	RoleCustomer = "customer"
)

// Status pesanan. Alur normal: pending -> paid -> shipped -> completed.
// Pesanan boleh dibatalkan selama masih pending atau paid.
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusShipped   = "shipped"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrOutOfStock        = errors.New("stok tidak mencukupi")
	ErrInvalidTransition = errors.New("perubahan status tidak diizinkan")
	ErrForbidden         = errors.New("tidak punya akses")
	ErrProductInUse      = errors.New("produk sudah pernah dipesan dan tidak bisa dihapus")
)

// InvalidInputError dipakai untuk kesalahan input yang pesannya aman ditampilkan ke klien.
type InvalidInputError struct{ Message string }

func (e *InvalidInputError) Error() string { return e.Message }

func Invalid(msg string) error { return &InvalidInputError{Message: msg} }

// Actor adalah pengguna yang sedang melakukan permintaan (hasil verifikasi token).
type Actor struct {
	UserID int
	Admin  bool
}

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Price       int64     `json:"price"` // rupiah, bilangan bulat (bukan float)
	Stock       int       `json:"stock"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProductInput adalah data yang boleh dikirim klien saat membuat atau mengubah produk.
type ProductInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
}

const (
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortName      = "name"
)

type ListParams struct {
	Query    string
	Category string
	Sort     string
	Page     int
	Limit    int
}

type ListResult struct {
	Items []Product `json:"items"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}

type CartItem struct {
	ProductID int    `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Qty       int    `json:"qty"`
	Stock     int    `json:"stock"`
	ImageURL  string `json:"image_url"`
	Subtotal  int64  `json:"subtotal"`
}

type Cart struct {
	Items []CartItem `json:"items"`
	Total int64      `json:"total"`
}

// OrderItemInput adalah satu baris yang dikirim klien: produk apa, berapa banyak.
type OrderItemInput struct {
	ProductID int `json:"product_id"`
	Qty       int `json:"qty"`
}

type OrderItem struct {
	ProductID int    `json:"product_id"`
	Name      string `json:"name"`
	Qty       int    `json:"qty"`
	Price     int64  `json:"price"` // harga saat dibeli
}

type Order struct {
	ID        int         `json:"id"`
	UserID    int         `json:"user_id"`
	Total     int64       `json:"total"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	Items     []OrderItem `json:"items"`
}
