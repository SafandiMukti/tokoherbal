package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tokoherbal/product-service/internal/domain"
)

const productCols = `id, name, description, category, price, stock, image_url, created_at`

// orderByClause adalah DAFTAR PUTIH: nilai sort dari klien dipetakan ke potongan
// SQL yang kita tulis sendiri. Klien tidak pernah menyumbang teks ke ORDER BY
// (placeholder $1 tidak bisa dipakai untuk nama kolom, jadi ini satu-satunya cara aman).
var orderByClause = map[string]string{
	domain.SortNewest:    "id DESC",
	domain.SortPriceAsc:  "price ASC, id DESC",
	domain.SortPriceDesc: "price DESC, id DESC",
	domain.SortName:      "lower(name) ASC, id DESC",
}

type ProductStore struct{ db *pgxpool.Pool }

func NewProductStore(db *pgxpool.Pool) *ProductStore { return &ProductStore{db: db} }

func scanProduct(row scanner) (domain.Product, error) {
	var p domain.Product
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock, &p.ImageURL, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, err
}

// escapeLike menetralkan karakter khusus LIKE agar "50%" dicari apa adanya.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func (s *ProductStore) List(ctx context.Context, p domain.ListParams) ([]domain.Product, int, error) {
	var (
		conds []string
		args  []any
	)
	if p.Query != "" {
		args = append(args, "%"+escapeLike(p.Query)+"%")
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", len(args)))
	}
	if p.Category != "" {
		args = append(args, p.Category)
		conds = append(conds, fmt.Sprintf("category = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM products "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order, ok := orderByClause[p.Sort]
	if !ok {
		order = orderByClause[domain.SortNewest]
	}
	args = append(args, p.Limit, (p.Page-1)*p.Limit)
	// Sprintf di sini HANYA menyusun nomor placeholder dan potongan dari daftar
	// putih; nilai dari pengguna tetap lewat $1, $2, ...
	q := fmt.Sprintf(`SELECT %s FROM products %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		productCols, where, order, len(args)-1, len(args))

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.Product
	for rows.Next() {
		prod, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, prod)
	}
	return out, total, rows.Err()
}

func (s *ProductStore) Get(ctx context.Context, id int) (domain.Product, error) {
	return scanProduct(s.db.QueryRow(ctx, `SELECT `+productCols+` FROM products WHERE id = $1`, id))
}

func (s *ProductStore) Create(ctx context.Context, in domain.ProductInput) (domain.Product, error) {
	return scanProduct(s.db.QueryRow(ctx,
		`INSERT INTO products (name, description, category, price, stock)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+productCols,
		in.Name, in.Description, in.Category, in.Price, in.Stock))
}

func (s *ProductStore) Update(ctx context.Context, id int, in domain.ProductInput) (domain.Product, error) {
	return scanProduct(s.db.QueryRow(ctx,
		`UPDATE products
		 SET name = $2, description = $3, category = $4, price = $5, stock = $6, updated_at = now()
		 WHERE id = $1 RETURNING `+productCols,
		id, in.Name, in.Description, in.Category, in.Price, in.Stock))
}

func (s *ProductStore) Delete(ctx context.Context, id int) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		// order_items.product_id mereferensikan produk ini: riwayat pesanan harus tetap utuh
		return domain.ErrProductInUse
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *ProductStore) SetImage(ctx context.Context, id int, url string) (string, error) {
	var old string
	// CTE membaca nilai LAMA; UPDATE menulis nilai baru. Keduanya dalam satu perintah.
	err := s.db.QueryRow(ctx,
		`WITH old AS (SELECT image_url FROM products WHERE id = $1)
		 UPDATE products SET image_url = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING (SELECT image_url FROM old)`,
		id, url).Scan(&old)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return old, err
}

func (s *ProductStore) Categories(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT DISTINCT category FROM products ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
