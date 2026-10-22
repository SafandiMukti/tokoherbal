package handler

import (
	"net/http"
	"strconv"

	"tokoherbal/product-service/internal/domain"
)

func itoa(n int) string { return strconv.Itoa(n) }

func (h *Handler) getCart(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	cart, err := h.Orders.Cart(r.Context(), who.UserID)
	if err != nil {
		h.fail(w, r, "get cart", err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) setCartItem(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	productID, ok := pathID(w, r, "product_id")
	if !ok {
		return
	}
	var in struct {
		Qty int `json:"qty"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	cart, err := h.Orders.SetCartItem(r.Context(), who.UserID, productID, in.Qty)
	if err != nil {
		h.fail(w, r, "set cart item", err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) removeCartItem(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	productID, ok := pathID(w, r, "product_id")
	if !ok {
		return
	}
	cart, err := h.Orders.RemoveCartItem(r.Context(), who.UserID, productID)
	if err != nil {
		h.fail(w, r, "remove cart item", err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

// createOrder: body {"items":[{"product_id":1,"qty":2}]} memesan langsung.
// Body {} (atau items kosong) memesan seluruh isi keranjang.
func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	var in struct {
		Items []domain.OrderItemInput `json:"items"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := h.Orders.CreateOrder(r.Context(), who.UserID, in.Items)
	if err != nil {
		h.fail(w, r, "create order", err)
		return
	}
	w.Header().Set("Location", "/orders/"+itoa(o.ID))
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) listMyOrders(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	out, err := h.Orders.ListMine(r.Context(), who.UserID, queryInt(r, "page", 1), queryInt(r, "limit", 20))
	if err != nil {
		h.fail(w, r, "list orders", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) listAllOrders(w http.ResponseWriter, r *http.Request, _ domain.Actor) {
	out, err := h.Orders.ListAll(r.Context(), r.URL.Query().Get("status"), queryInt(r, "page", 1), queryInt(r, "limit", 20))
	if err != nil {
		h.fail(w, r, "list all orders", err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	o, err := h.Orders.GetOrder(r.Context(), who, id)
	if err != nil {
		h.fail(w, r, "get order", err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *Handler) changeOrderStatus(w http.ResponseWriter, r *http.Request, who domain.Actor) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := h.Orders.ChangeStatus(r.Context(), who, id, in.Status)
	if err != nil {
		h.fail(w, r, "change order status", err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}
