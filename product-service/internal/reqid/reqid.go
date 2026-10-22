// Package reqid menyimpan request id di context. Dipisah dari handler agar
// pemanggil keluar (internal/auth) bisa meneruskannya tanpa import siklik.
package reqid

import "context"

type key struct{}

func With(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key{}, id)
}

// From mengembalikan request id, atau string kosong bila tidak ada.
func From(ctx context.Context) string {
	id, _ := ctx.Value(key{}).(string)
	return id
}
