package service

import (
	"context"
	"log/slog"
	"sort"

	"tokoherbal/product-service/internal/domain"
)

const (
	maxQtyPerItem   = 99
	maxItemsPerCart = 50
)

// transitions: status tujuan -> status asal yang boleh. Satu-satunya tempat
// aturan alur pesanan ditulis.
var transitions = map[string][]string{
	domain.StatusPaid:      {domain.StatusPending},
	domain.StatusShipped:   {domain.StatusPaid},
	domain.StatusCompleted: {domain.StatusShipped},
	domain.StatusCancelled: {domain.StatusPending, domain.StatusPaid},
}

type OrderService struct {
	orders OrderStore
	carts  CartStore
	cache  Cache
	log    *slog.Logger
}

func NewOrderService(o OrderStore, c CartStore, cache Cache, log *slog.Logger) *OrderService {
	return &OrderService{orders: o, carts: c, cache: cache, log: log}
}

// ---- keranjang ----

func (s *OrderService) Cart(ctx context.Context, userID int) (domain.Cart, error) {
	return s.carts.Get(ctx, userID)
}

func (s *OrderService) SetCartItem(ctx context.Context, userID, productID, qty int) (domain.Cart, error) {
	if productID <= 0 {
		return domain.Cart{}, domain.Invalid("product_id tidak valid")
	}
	if qty < 1 || qty > maxQtyPerItem {
		return domain.Cart{}, domain.Invalid("qty harus antara 1 dan 99")
	}
	if err := s.carts.Set(ctx, userID, productID, qty); err != nil {
		return domain.Cart{}, err
	}
	return s.carts.Get(ctx, userID)
}

func (s *OrderService) RemoveCartItem(ctx context.Context, userID, productID int) (domain.Cart, error) {
	if err := s.carts.Remove(ctx, userID, productID); err != nil {
		return domain.Cart{}, err
	}
	return s.carts.Get(ctx, userID)
}

// ---- pesanan ----

// CreateOrder membuat pesanan. Bila items kosong, isi keranjang pengguna yang
// dipesan lalu keranjang dikosongkan.
func (s *OrderService) CreateOrder(ctx context.Context, userID int, items []domain.OrderItemInput) (domain.Order, error) {
	clearCart := false
	if len(items) == 0 {
		cart, err := s.carts.Get(ctx, userID)
		if err != nil {
			return domain.Order{}, err
		}
		if len(cart.Items) == 0 {
			return domain.Order{}, domain.Invalid("keranjang kosong")
		}
		for _, it := range cart.Items {
			items = append(items, domain.OrderItemInput{ProductID: it.ProductID, Qty: it.Qty})
		}
		clearCart = true
	}

	items, err := normalizeItems(items)
	if err != nil {
		return domain.Order{}, err
	}

	id, err := s.orders.Create(ctx, userID, items, clearCart)
	if err != nil {
		return domain.Order{}, err
	}
	invalidate(ctx, s.cache, s.log) // stok berubah, cache daftar produk basi
	s.log.Info("order created", "order_id", id, "user_id", userID, "items", len(items))
	return s.orders.Get(ctx, id)
}

// normalizeItems memvalidasi, menggabungkan produk yang sama, dan mengurutkan
// berdasarkan product_id. Urutan tetap penting: semua transaksi mengunci baris
// produk dengan urutan yang sama, sehingga dua pesanan serentak tidak saling
// menunggu melingkar (deadlock).
func normalizeItems(in []domain.OrderItemInput) ([]domain.OrderItemInput, error) {
	merged := make(map[int]int, len(in))
	for _, it := range in {
		if it.ProductID <= 0 {
			return nil, domain.Invalid("product_id tidak valid")
		}
		if it.Qty < 1 || it.Qty > maxQtyPerItem {
			return nil, domain.Invalid("qty harus antara 1 dan 99")
		}
		merged[it.ProductID] += it.Qty
	}
	if len(merged) > maxItemsPerCart {
		return nil, domain.Invalid("maksimal 50 jenis produk per pesanan")
	}
	out := make([]domain.OrderItemInput, 0, len(merged))
	for pid, qty := range merged {
		if qty > maxQtyPerItem {
			return nil, domain.Invalid("qty harus antara 1 dan 99")
		}
		out = append(out, domain.OrderItemInput{ProductID: pid, Qty: qty})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out, nil
}

// GetOrder: pelanggan hanya boleh melihat pesanannya sendiri. Pesanan orang lain
// dijawab "tidak ditemukan" (bukan "dilarang") agar id pesanan tidak bisa ditebak.
func (s *OrderService) GetOrder(ctx context.Context, who domain.Actor, id int) (domain.Order, error) {
	o, err := s.orders.Get(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !who.Admin && o.UserID != who.UserID {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

func (s *OrderService) ListMine(ctx context.Context, userID, page, limit int) ([]domain.Order, error) {
	limit, offset := paging(page, limit)
	out, err := s.orders.ListByUser(ctx, userID, limit, offset)
	if out == nil {
		out = []domain.Order{}
	}
	return out, err
}

func (s *OrderService) ListAll(ctx context.Context, status string, page, limit int) ([]domain.Order, error) {
	if status != "" {
		if _, ok := transitions[status]; !ok && status != domain.StatusPending {
			return nil, domain.Invalid("status tidak dikenal")
		}
	}
	limit, offset := paging(page, limit)
	out, err := s.orders.ListAll(ctx, status, limit, offset)
	if out == nil {
		out = []domain.Order{}
	}
	return out, err
}

func paging(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return limit, (page - 1) * limit
}

// ChangeStatus menerapkan aturan alur status.
//   - Admin boleh semua perubahan yang sah menurut tabel transitions.
//   - Pelanggan hanya boleh membatalkan pesanannya sendiri.
//
// Pembayaran disimulasikan: admin yang menandai "paid". Integrasi payment
// gateway sengaja di luar cakupan proyek ini.
func (s *OrderService) ChangeStatus(ctx context.Context, who domain.Actor, id int, to string) (domain.Order, error) {
	allowedFrom, known := transitions[to]
	if !known {
		return domain.Order{}, domain.Invalid("status tujuan tidak dikenal")
	}

	o, err := s.orders.Get(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if !who.Admin {
		if o.UserID != who.UserID {
			return domain.Order{}, domain.ErrNotFound
		}
		if to != domain.StatusCancelled {
			return domain.Order{}, domain.ErrForbidden
		}
	}

	// pengecekan status sebenarnya terjadi di dalam transaksi (dengan row lock),
	// bukan dari objek `o` di atas yang bisa sudah basi
	if err := s.orders.SetStatus(ctx, id, allowedFrom, to); err != nil {
		return domain.Order{}, err
	}
	if to == domain.StatusCancelled {
		invalidate(ctx, s.cache, s.log) // stok dikembalikan
	}
	s.log.Info("order status changed", "order_id", id, "to", to, "by_user", who.UserID)
	return s.orders.Get(ctx, id)
}
