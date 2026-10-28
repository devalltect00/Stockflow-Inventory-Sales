package httpapi

import (
	"net/http"
	"stockflow/backend/internal/orders"
)

// listOrders supports browsing seeded and previously created orders without local-only state.
func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	page, limit, err := pagination(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.orders.List(r.Context(), page, limit)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// createOrder returns only the committed order with authoritative item prices and total.
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var in orders.CreateInput
	if err := decode(w, r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.orders.Create(r.Context(), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, out)
}

// orderDetail returns historical prices, status, and cancellation timestamp.
func (s *Server) orderDetail(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.orders.Get(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

// cancelOrder returns the committed state and rejects repeated cancellation with 409.
func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.orders.Cancel(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}
