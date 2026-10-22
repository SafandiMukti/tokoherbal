package handler

import (
	"errors"
	"net/http"

	"tokoherbal/product-service/internal/domain"
	"tokoherbal/product-service/internal/upload"
)

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	res, err := h.Products.List(r.Context(), domain.ListParams{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Sort:     q.Get("sort"),
		Page:     queryInt(r, "page", 1),
		Limit:    queryInt(r, "limit", 0),
	})
	if err != nil {
		h.fail(w, r, "list products", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.Products.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, "get product", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.Products.Categories(r.Context())
	if err != nil {
		h.fail(w, r, "list categories", err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request, _ domain.Actor) {
	var in domain.ProductInput
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := h.Products.Create(r.Context(), in)
	if err != nil {
		h.fail(w, r, "create product", err)
		return
	}
	w.Header().Set("Location", "/products/"+itoa(p.ID))
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request, _ domain.Actor) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in domain.ProductInput
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := h.Products.Update(r.Context(), id, in)
	if err != nil {
		h.fail(w, r, "update product", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request, _ domain.Actor) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.Products.Delete(r.Context(), id); err != nil {
		h.fail(w, r, "delete product", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// uploadImage menerima multipart/form-data dengan field "image".
func (h *Handler) uploadImage(w http.ResponseWriter, r *http.Request, _ domain.Actor) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}

	// batas body sedikit di atas batas gambar untuk ruang header multipart
	r.Body = http.MaxBytesReader(w, r.Body, upload.MaxBytes+(512<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran gambar maksimal 2 MB")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, `field "image" wajib diisi`)
		return
	}
	defer file.Close()

	name, err := h.Uploads.Save(file)
	switch {
	case errors.Is(err, upload.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
		return
	case errors.Is(err, upload.ErrBadType):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		h.serverError(w, r, "save upload", err)
		return
	}

	old, err := h.Products.SetImage(r.Context(), id, "/uploads/"+name)
	if err != nil {
		h.Uploads.Remove(name) // jangan tinggalkan file yatim
		h.fail(w, r, "set image", err)
		return
	}
	if oldName, ok := isUploadURL(old); ok {
		h.Uploads.Remove(oldName)
	}

	p, err := h.Products.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, "get product", err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
