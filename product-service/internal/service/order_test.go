package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tokoherbal/product-service/internal/domain"
	"tokoherbal/product-service/internal/fakes"
)

func newOrderSvc() (*OrderService, *fakes.OrderStore, *fakes.CartStore, *fakes.Cache) {
	o, c, ch := fakes.NewOrderStore(), fakes.NewCartStore(), fakes.NewCache()
	return NewOrderService(o, c, ch, quietLog()), o, c, ch
}

func TestNormalizeItems(t *testing.T) {
	got, err := normalizeItems([]domain.OrderItemInput{
		{ProductID: 5, Qty: 1}, {ProductID: 2, Qty: 3}, {ProductID: 5, Qty: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.OrderItemInput{{ProductID: 2, Qty: 3}, {ProductID: 5, Qty: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v (digabung dan terurut)", got, want)
	}

	bad := [][]domain.OrderItemInput{
		{{ProductID: 0, Qty: 1}},
		{{ProductID: 1, Qty: 0}},
		{{ProductID: 1, Qty: 100}},
		{{ProductID: 1, Qty: 60}, {ProductID: 1, Qty: 60}}, // total gabungan melewati batas
	}
	for i, in := range bad {
		if _, err := normalizeItems(in); err == nil {
			t.Errorf("kasus %d seharusnya ditolak", i)
		}
	}
}

func TestCreateOrderDirect(t *testing.T) {
	svc, os, _, ch := newOrderSvc()
	o, err := svc.CreateOrder(context.Background(), 7, []domain.OrderItemInput{{ProductID: 1, Qty: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if o.UserID != 7 || o.Status != domain.StatusPending {
		t.Fatalf("order = %+v", o)
	}
	if os.CreateCalls[0].ClearCart {
		t.Fatal("pesanan langsung tidak boleh mengosongkan keranjang")
	}
	if len(ch.Deletes) != 1 {
		t.Fatal("cache produk harus diinvalidasi karena stok berubah")
	}
}

func TestCreateOrderFromCart(t *testing.T) {
	svc, os, carts, _ := newOrderSvc()
	ctx := context.Background()

	if _, err := svc.CreateOrder(ctx, 7, nil); err == nil {
		t.Fatal("keranjang kosong harus ditolak")
	}

	carts.Set(ctx, 7, 3, 2)
	carts.Set(ctx, 7, 1, 1)
	if _, err := svc.CreateOrder(ctx, 7, nil); err != nil {
		t.Fatal(err)
	}
	call := os.CreateCalls[0]
	if !call.ClearCart {
		t.Fatal("pesanan dari keranjang harus mengosongkan keranjang")
	}
	if call.Items[0].ProductID != 1 || call.Items[1].ProductID != 3 {
		t.Fatalf("item harus terurut: %+v", call.Items)
	}
}

func TestCreateOrderOutOfStock(t *testing.T) {
	svc, os, _, ch := newOrderSvc()
	os.CreateErr = domain.ErrOutOfStock
	_, err := svc.CreateOrder(context.Background(), 1, []domain.OrderItemInput{{ProductID: 1, Qty: 1}})
	if !errors.Is(err, domain.ErrOutOfStock) {
		t.Fatalf("err = %v", err)
	}
	if len(ch.Deletes) != 0 {
		t.Fatal("pesanan gagal tidak mengubah stok, jadi cache tidak perlu dihapus")
	}
}

func TestGetOrderOwnership(t *testing.T) {
	svc, _, _, _ := newOrderSvc()
	ctx := context.Background()
	o, _ := svc.CreateOrder(ctx, 1, []domain.OrderItemInput{{ProductID: 1, Qty: 1}})

	if _, err := svc.GetOrder(ctx, domain.Actor{UserID: 1}, o.ID); err != nil {
		t.Fatalf("pemilik harus bisa: %v", err)
	}
	if _, err := svc.GetOrder(ctx, domain.Actor{UserID: 2}, o.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("pengguna lain harus dapat NotFound, dapat %v", err)
	}
	if _, err := svc.GetOrder(ctx, domain.Actor{UserID: 2, Admin: true}, o.ID); err != nil {
		t.Fatalf("admin harus bisa: %v", err)
	}
}

func TestChangeStatus(t *testing.T) {
	admin := domain.Actor{UserID: 99, Admin: true}
	owner := domain.Actor{UserID: 1}
	other := domain.Actor{UserID: 2}

	cases := []struct {
		name    string
		who     domain.Actor
		prepare []string // urutan status admin sebelum kasus diuji
		to      string
		wantErr error
	}{
		{"admin: pending -> paid", admin, nil, domain.StatusPaid, nil},
		{"admin: pending -> shipped (loncat)", admin, nil, domain.StatusShipped, domain.ErrInvalidTransition},
		{"admin: paid -> shipped", admin, []string{domain.StatusPaid}, domain.StatusShipped, nil},
		{"admin: shipped -> completed", admin, []string{domain.StatusPaid, domain.StatusShipped}, domain.StatusCompleted, nil},
		{"admin: shipped -> cancelled (sudah dikirim)", admin, []string{domain.StatusPaid, domain.StatusShipped}, domain.StatusCancelled, domain.ErrInvalidTransition},
		{"admin: status ngawur", admin, nil, "refunded", &domain.InvalidInputError{}},
		{"pemilik membatalkan pending", owner, nil, domain.StatusCancelled, nil},
		{"pemilik mencoba paid", owner, nil, domain.StatusPaid, domain.ErrForbidden},
		{"orang lain membatalkan", other, nil, domain.StatusCancelled, domain.ErrNotFound},
		{"pemilik membatalkan setelah dikirim", owner, []string{domain.StatusPaid, domain.StatusShipped}, domain.StatusCancelled, domain.ErrInvalidTransition},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _, _, ch := newOrderSvc()
			ctx := context.Background()
			o, _ := svc.CreateOrder(ctx, owner.UserID, []domain.OrderItemInput{{ProductID: 1, Qty: 1}})
			for _, st := range c.prepare {
				if _, err := svc.ChangeStatus(ctx, admin, o.ID, st); err != nil {
					t.Fatalf("persiapan %s: %v", st, err)
				}
			}
			deletesBefore := len(ch.Deletes)

			got, err := svc.ChangeStatus(ctx, c.who, o.ID, c.to)
			switch want := c.wantErr.(type) {
			case nil:
				if err != nil {
					t.Fatalf("err = %v", err)
				}
				if got.Status != c.to {
					t.Fatalf("status = %s, want %s", got.Status, c.to)
				}
				// pembatalan mengembalikan stok, jadi cache harus dibersihkan
				if c.to == domain.StatusCancelled && len(ch.Deletes) != deletesBefore+1 {
					t.Fatal("pembatalan harus menginvalidasi cache")
				}
			case *domain.InvalidInputError:
				var invalid *domain.InvalidInputError
				if !errors.As(err, &invalid) {
					t.Fatalf("err = %v, want InvalidInputError", err)
				}
			default:
				if !errors.Is(err, want) {
					t.Fatalf("err = %v, want %v", err, want)
				}
			}
		})
	}
}

func TestSetCartItemValidation(t *testing.T) {
	svc, _, _, _ := newOrderSvc()
	ctx := context.Background()
	if _, err := svc.SetCartItem(ctx, 1, 1, 0); err == nil {
		t.Error("qty 0 harus ditolak")
	}
	if _, err := svc.SetCartItem(ctx, 1, 0, 1); err == nil {
		t.Error("product_id 0 harus ditolak")
	}
	cart, err := svc.SetCartItem(ctx, 1, 5, 2)
	if err != nil || len(cart.Items) != 1 || cart.Items[0].Qty != 2 {
		t.Fatalf("cart=%+v err=%v", cart, err)
	}
	cart, _ = svc.RemoveCartItem(ctx, 1, 5)
	if len(cart.Items) != 0 {
		t.Fatalf("cart harus kosong: %+v", cart)
	}
}

func TestListAllStatusFilter(t *testing.T) {
	svc, _, _, _ := newOrderSvc()
	if _, err := svc.ListAll(context.Background(), "ngawur", 1, 10); err == nil {
		t.Fatal("status tidak dikenal harus ditolak")
	}
	for _, st := range []string{"", "pending", "paid", "shipped", "completed", "cancelled"} {
		if _, err := svc.ListAll(context.Background(), st, 1, 10); err != nil {
			t.Errorf("status %q: %v", st, err)
		}
	}
}
