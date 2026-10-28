package httpapi

import (
	"net/http"
	"strings"

	"stockflow/backend/internal/apperror"
	"stockflow/backend/internal/inventory"
)

// listProducts validates pagination and performs literal substring search on the server.
func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	page, limit, err := pagination(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	filter := r.URL.Query().Get("filter")
	if filter == "" {
		filter = "all"
	}
	if filter != "all" && filter != "active" && filter != "inactive" && filter != "low-stock" {
		s.fail(w, apperror.Validation("Filter must be all, active, inactive, or low-stock.", nil))
		return
	}
	out, err := s.inventory.List(r.Context(), inventory.ListParams{Search: strings.TrimSpace(r.URL.Query().Get("search")), Filter: filter, Page: page, Limit: limit})
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// summary returns whole-inventory metrics independent of the products table's current page.
func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	out, err := s.inventory.Summary(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// createProduct accepts catalog metadata and returns the newly created zero-stock product.
func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var in inventory.ProductInput
	if err := decode(w, r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.inventory.Create(r.Context(), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, out)
}

// updateProduct edits metadata; immutable SKU and stock rules remain enforced below HTTP.
func (s *Server) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	var in inventory.ProductInput
	if err = decode(w, r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.inventory.Update(r.Context(), id, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// deleteProduct removes only products whose entire history is empty.
func (s *Server) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err = s.inventory.Delete(r.Context(), id); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 204, nil)
}

// adjustStock exposes conflicts with availability intact for inline frontend feedback.
func (s *Server) adjustStock(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	var in inventory.StockInput
	if err = decode(w, r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.inventory.AdjustStock(r.Context(), id, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// movements returns an ordered, immutable history collection for one existing product.
func (s *Server) movements(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.inventory.Movements(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": out})
}
