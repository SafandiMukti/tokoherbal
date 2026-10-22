package store

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tokoherbal/product-service/internal/domain"
)

type OrderStore struct{ db *pgxpool.Pool }

func NewOrderStore(db *pgxpool.Pool) *OrderStore { return &OrderStore{db: db} }

// Create membuat pesanan secara atomik. Pemanggil (service) sudah mengurutkan
// items berdasarkan product_id supaya urutan penguncian baris selalu sama.
//
// Inti pencegahan "dua pembeli rebutan stok terakhir":
//
//	UPDATE products SET stock = stock - $qty WHERE id = $id AND stock >= $qty
//
// Postgres mengunci baris itu sampai transaksi selesai. Pembeli kedua menunggu,
// lalu membaca stok yang sudah berkurang, kondisi stock >= qty gagal, dan
// 0 baris terubah. Tidak ada celah "cek dulu, kurangi kemudian".
func (s *OrderStore) Create(ctx context.Context, userID int, items []domain.OrderItemInput, clearCart bool) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) // aman dipanggil setelah Commit (tidak berefek)

	type line struct {
		productID int
		qty       int
		price     int64
	}
	lines := make([]line, 0, len(items))
	var total int64

	for _, it := range items {
		var price int64
		err := tx.QueryRow(ctx,
			`UPDATE products SET stock = stock - $1, updated_at = now()
			 WHERE id = $2 AND stock >= $1
			 RETURNING price`, it.Qty, it.ProductID).Scan(&price)
		if errors.Is(err, pgx.ErrNoRows) {
			// bedakan "produk tidak ada" dari "stok kurang"
			var exists bool
			if e := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, it.ProductID).Scan(&exists); e != nil {
				return 0, e
			}
			if !exists {
				return 0, domain.ErrNotFound
			}
			return 0, fmt.Errorf("produk %d: %w", it.ProductID, domain.ErrOutOfStock)
		}
		if err != nil {
			return 0, err
		}
		lines = append(lines, line{it.ProductID, it.Qty, price})
		total += price * int64(it.Qty)
	}

	var orderID int
	if err := tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, total) VALUES ($1, $2) RETURNING id`,
		userID, total).Scan(&orderID); err != nil {
		return 0, err
	}

	for _, l := range lines {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, qty, price) VALUES ($1, $2, $3, $4)`,
			orderID, l.productID, l.qty, l.price); err != nil {
			return 0, err
		}
	}

	if clearCart {
		if _, err := tx.Exec(ctx, `DELETE FROM cart_items WHERE user_id = $1`, userID); err != nil {
			return 0, err
		}
	}
	return orderID, tx.Commit(ctx)
}

func (s *OrderStore) Get(ctx context.Context, id int) (domain.Order, error) {
	var o domain.Order
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, total, status, created_at FROM orders WHERE id = $1`, id).
		Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Order{}, err
	}
	orders := []domain.Order{o}
	if err := s.attachItems(ctx, orders); err != nil {
		return domain.Order{}, err
	}
	return orders[0], nil
}

func (s *OrderStore) ListByUser(ctx context.Context, userID, limit, offset int) ([]domain.Order, error) {
	return s.list(ctx,
		`SELECT id, user_id, total, status, created_at FROM orders
		 WHERE user_id = $1 ORDER BY id DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
}

func (s *OrderStore) ListAll(ctx context.Context, status string, limit, offset int) ([]domain.Order, error) {
	return s.list(ctx,
		`SELECT id, user_id, total, status, created_at FROM orders
		 WHERE ($1 = '' OR status = $1) ORDER BY id DESC LIMIT $2 OFFSET $3`,
		status, limit, offset)
}

func (s *OrderStore) list(ctx context.Context, query string, args ...any) ([]domain.Order, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	return out, s.attachItems(ctx, out)
}

// attachItems mengisi Items untuk SEMUA order dengan satu query (order_id = ANY($1)),
// bukan satu query per order (N+1).
func (s *OrderStore) attachItems(ctx context.Context, orders []domain.Order) error {
	if len(orders) == 0 {
		return nil
	}
	ids := make([]int32, len(orders))
	byID := make(map[int]*domain.Order, len(orders))
	for i := range orders {
		ids[i] = int32(orders[i].ID)
		orders[i].Items = []domain.OrderItem{}
		byID[orders[i].ID] = &orders[i]
	}

	rows, err := s.db.Query(ctx,
		`SELECT oi.order_id, oi.product_id, p.name, oi.qty, oi.price
		 FROM order_items oi
		 JOIN products p ON p.id = oi.product_id
		 WHERE oi.order_id = ANY($1)
		 ORDER BY oi.id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID int
		var it domain.OrderItem
		if err := rows.Scan(&orderID, &it.ProductID, &it.Name, &it.Qty, &it.Price); err != nil {
			return err
		}
		if o := byID[orderID]; o != nil {
			o.Items = append(o.Items, it)
		}
	}
	return rows.Err()
}

func (s *OrderStore) SetStatus(ctx context.Context, id int, allowedFrom []string, to string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE: kunci baris pesanan supaya dua perubahan status serentak
	// (misal admin menandai "paid" tepat saat pelanggan membatalkan) berjalan bergantian.
	var current string
	err = tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 FOR UPDATE`, id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}
	if !slices.Contains(allowedFrom, current) {
		return domain.ErrInvalidTransition
	}

	if _, err := tx.Exec(ctx, `UPDATE orders SET status = $2, updated_at = now() WHERE id = $1`, id, to); err != nil {
		return err
	}
	if to == domain.StatusCancelled {
		if _, err := tx.Exec(ctx,
			`UPDATE products p SET stock = p.stock + oi.qty, updated_at = now()
			 FROM order_items oi
			 WHERE oi.order_id = $1 AND oi.product_id = p.id`, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
