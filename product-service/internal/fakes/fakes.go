// Package fakes berisi implementasi palsu (di memori) dari interface service.
// Dipakai test service dan handler supaya cepat dan tidak butuh PostgreSQL/Redis.
package fakes

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"tokoherbal/product-service/internal/domain"
)

// ---- ProductStore ----

type ProductStore struct {
	mu       sync.Mutex
	Products map[int]domain.Product
	nextID   int

	ListCalls int // berapa kali List dipanggil (untuk memastikan cache bekerja)
	GetCalls  int
}

func NewProductStore() *ProductStore {
	return &ProductStore{Products: map[int]domain.Product{}, nextID: 1}
}

func (s *ProductStore) List(_ context.Context, p domain.ListParams) ([]domain.Product, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ListCalls++

	var all []domain.Product
	for _, pr := range s.Products {
		if p.Query != "" && !strings.Contains(strings.ToLower(pr.Name), strings.ToLower(p.Query)) {
			continue
		}
		if p.Category != "" && pr.Category != p.Category {
			continue
		}
		all = append(all, pr)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	total := len(all)

	start := (p.Page - 1) * p.Limit
	if start > total {
		start = total
	}
	end := start + p.Limit
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (s *ProductStore) Get(_ context.Context, id int) (domain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GetCalls++
	p, ok := s.Products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	return p, nil
}

func (s *ProductStore) Create(_ context.Context, in domain.ProductInput) (domain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := domain.Product{
		ID: s.nextID, Name: in.Name, Description: in.Description, Category: in.Category,
		Price: in.Price, Stock: in.Stock, CreatedAt: time.Now(),
	}
	s.nextID++
	s.Products[p.ID] = p
	return p, nil
}

func (s *ProductStore) Update(_ context.Context, id int, in domain.ProductInput) (domain.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.Products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}
	p.Name, p.Description, p.Category, p.Price, p.Stock = in.Name, in.Description, in.Category, in.Price, in.Stock
	s.Products[id] = p
	return p, nil
}

func (s *ProductStore) Delete(_ context.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Products[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.Products, id)
	return nil
}

func (s *ProductStore) SetImage(_ context.Context, id int, url string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.Products[id]
	if !ok {
		return "", domain.ErrNotFound
	}
	old := p.ImageURL
	p.ImageURL = url
	s.Products[id] = p
	return old, nil
}

func (s *ProductStore) Categories(context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, p := range s.Products {
		if !seen[p.Category] {
			seen[p.Category] = true
			out = append(out, p.Category)
		}
	}
	sort.Strings(out)
	return out, nil
}

// ---- CartStore ----

type CartStore struct {
	mu    sync.Mutex
	Items map[int]map[int]int // userID -> productID -> qty
}

func NewCartStore() *CartStore { return &CartStore{Items: map[int]map[int]int{}} }

func (s *CartStore) Get(_ context.Context, userID int) (domain.Cart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cart := domain.Cart{Items: []domain.CartItem{}}
	var ids []int
	for pid := range s.Items[userID] {
		ids = append(ids, pid)
	}
	sort.Ints(ids)
	for _, pid := range ids {
		qty := s.Items[userID][pid]
		cart.Items = append(cart.Items, domain.CartItem{ProductID: pid, Name: "produk", Price: 1000, Qty: qty, Stock: 100, Subtotal: int64(1000 * qty)})
		cart.Total += int64(1000 * qty)
	}
	return cart, nil
}

func (s *CartStore) Set(_ context.Context, userID, productID, qty int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Items[userID] == nil {
		s.Items[userID] = map[int]int{}
	}
	s.Items[userID][productID] = qty
	return nil
}

func (s *CartStore) Remove(_ context.Context, userID, productID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Items[userID], productID)
	return nil
}

// ---- OrderStore ----

type CreateCall struct {
	UserID    int
	Items     []domain.OrderItemInput
	ClearCart bool
}

type OrderStore struct {
	mu          sync.Mutex
	Orders      map[int]domain.Order
	nextID      int
	CreateCalls []CreateCall
	CreateErr   error
}

func NewOrderStore() *OrderStore { return &OrderStore{Orders: map[int]domain.Order{}, nextID: 1} }

func (s *OrderStore) Create(_ context.Context, userID int, items []domain.OrderItemInput, clearCart bool) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CreateCalls = append(s.CreateCalls, CreateCall{userID, items, clearCart})
	if s.CreateErr != nil {
		return 0, s.CreateErr
	}
	o := domain.Order{ID: s.nextID, UserID: userID, Status: domain.StatusPending, CreatedAt: time.Now(), Items: []domain.OrderItem{}}
	for _, it := range items {
		o.Items = append(o.Items, domain.OrderItem{ProductID: it.ProductID, Name: "produk", Qty: it.Qty, Price: 1000})
		o.Total += int64(1000 * it.Qty)
	}
	s.nextID++
	s.Orders[o.ID] = o
	return o.ID, nil
}

func (s *OrderStore) Get(_ context.Context, id int) (domain.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.Orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

func (s *OrderStore) ListByUser(_ context.Context, userID, limit, offset int) ([]domain.Order, error) {
	return s.filter(func(o domain.Order) bool { return o.UserID == userID }, limit, offset), nil
}

func (s *OrderStore) ListAll(_ context.Context, status string, limit, offset int) ([]domain.Order, error) {
	return s.filter(func(o domain.Order) bool { return status == "" || o.Status == status }, limit, offset), nil
}

func (s *OrderStore) filter(keep func(domain.Order) bool, limit, offset int) []domain.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []domain.Order
	for _, o := range s.Orders {
		if keep(o) {
			all = append(all, o)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID > all[j].ID })
	if offset > len(all) {
		offset = len(all)
	}
	all = all[offset:]
	if limit < len(all) {
		all = all[:limit]
	}
	return all
}

func (s *OrderStore) SetStatus(_ context.Context, id int, allowedFrom []string, to string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.Orders[id]
	if !ok {
		return domain.ErrNotFound
	}
	allowed := false
	for _, a := range allowedFrom {
		if a == o.Status {
			allowed = true
		}
	}
	if !allowed {
		return domain.ErrInvalidTransition
	}
	o.Status = to
	s.Orders[id] = o
	return nil
}

// ---- Cache ----

type Cache struct {
	mu        sync.Mutex
	data      map[string][]byte
	Sets      int
	Deletes   []string // prefix yang pernah dihapus
	GetErr    error    // bila diisi, Get selalu gagal (simulasi Redis mati)
	DeleteErr error
}

func NewCache() *Cache { return &Cache{data: map[string][]byte{}} }

func (c *Cache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.GetErr != nil {
		return nil, false, c.GetErr
	}
	v, ok := c.data[key]
	return v, ok, nil
}

func (c *Cache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sets++
	c.data[key] = val
	return nil
}

func (c *Cache) DeletePrefix(_ context.Context, prefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Deletes = append(c.Deletes, prefix)
	if c.DeleteErr != nil {
		return c.DeleteErr
	}
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
		}
	}
	return nil
}

// Counter: penghitung rate limit palsu.
type Counter struct {
	mu  sync.Mutex
	N   map[string]int64
	Err error
}

func NewCounter() *Counter { return &Counter{N: map[string]int64{}} }

func (c *Counter) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Err != nil {
		return 0, c.Err
	}
	c.N[key]++
	return c.N[key], nil
}
