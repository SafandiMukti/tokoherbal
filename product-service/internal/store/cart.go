package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tokoherbal/product-service/internal/domain"
)

type CartStore struct{ db *pgxpool.Pool }

func NewCartStore(db *pgxpool.Pool) *CartStore { return &CartStore{db: db} }

// Get mengambil isi keranjang beserta data produk terbaru lewat satu JOIN
// (bukan satu query per baris: itu masalah N+1).
func (s *CartStore) Get(ctx context.Context, userID int) (domain.Cart, error) {
	rows, err := s.db.Query(ctx,
		`SELECT c.product_id, p.name, p.price, c.qty, p.stock, p.image_url
		 FROM cart_items c
		 JOIN products p ON p.id = c.product_id
		 WHERE c.user_id = $1
		 ORDER BY c.id`, userID)
	if err != nil {
		return domain.Cart{}, err
	}
	defer rows.Close()

	cart := domain.Cart{Items: []domain.CartItem{}}
	for rows.Next() {
		var it domain.CartItem
		if err := rows.Scan(&it.ProductID, &it.Name, &it.Price, &it.Qty, &it.Stock, &it.ImageURL); err != nil {
			return domain.Cart{}, err
		}
		it.Subtotal = it.Price * int64(it.Qty)
		cart.Total += it.Subtotal
		cart.Items = append(cart.Items, it)
	}
	return cart, rows.Err()
}

func (s *CartStore) Set(ctx context.Context, userID, productID, qty int) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO cart_items (user_id, product_id, qty) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, product_id) DO UPDATE SET qty = EXCLUDED.qty`,
		userID, productID, qty)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return domain.ErrNotFound // product_id tidak ada
	}
	return err
}

func (s *CartStore) Remove(ctx context.Context, userID, productID int) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}
