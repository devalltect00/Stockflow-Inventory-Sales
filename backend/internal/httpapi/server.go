package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"stockflow/backend/internal/apperror"
	"stockflow/backend/internal/inventory"
	"stockflow/backend/internal/orders"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Server explicitly holds dependencies shared by the thin request handlers.
type Server struct {
	inventory *inventory.Service
	orders    *orders.Service
	pool      *pgxpool.Pool
	logger    *slog.Logger
}

// New registers the public contract and wraps it in recovery, logging, and a bounded deadline.
func New(pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	s := &Server{inventory: inventory.NewService(inventory.NewStore(pool)), orders: orders.NewService(orders.NewStore(pool)), pool: pool, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/products", s.listProducts)
	mux.HandleFunc("GET /api/products/summary", s.summary)
	mux.HandleFunc("POST /api/products", s.createProduct)
	mux.HandleFunc("PUT /api/products/{id}", s.updateProduct)
	mux.HandleFunc("DELETE /api/products/{id}", s.deleteProduct)
	mux.HandleFunc("POST /api/products/{id}/stock", s.adjustStock)
	mux.HandleFunc("GET /api/products/{id}/movements", s.movements)
	mux.HandleFunc("GET /api/orders", s.listOrders)
	mux.HandleFunc("POST /api/orders", s.createOrder)
	mux.HandleFunc("GET /api/orders/{id}", s.orderDetail)
	mux.HandleFunc("POST /api/orders/{id}/cancel", s.cancelOrder)
	mux.HandleFunc("/", s.notFound)
	return s.middleware(http.TimeoutHandler(mux, 15*time.Second, `{"error":{"code":"TIMEOUT","message":"Request timed out. Please retry."}}`))
}

// writeJSON owns response encoding and consistent content-type/cache headers.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}

// fail translates known errors while logging unexpected failures without returning SQL internals.
func (s *Server) fail(w http.ResponseWriter, err error) {
	var public *apperror.Error
	if !errors.As(err, &public) {
		s.logger.Error("request failed", "error", err)
		public = &apperror.Error{Status: 500, Code: "INTERNAL_ERROR", Message: "Something went wrong. Please retry."}
	}
	writeJSON(w, public.Status, map[string]any{"error": public})
}

// decode rejects extra properties, oversized requests, fractional integers, and trailing JSON.
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return apperror.Validation("Content-Type must be application/json.", nil)
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperror.Validation("Invalid JSON body: "+err.Error(), nil)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return apperror.Validation("Body must contain exactly one JSON object.", nil)
	}
	return nil
}

// pathID validates IDs before they reach SQL and preserves predictable 400 responses.
func pathID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 || id > inventory.MaxInteger {
		return 0, apperror.Validation("ID must be a positive integer.", nil)
	}
	return id, nil
}

// pagination supplies defaults and rejects unbounded or malformed page requests.
func pagination(r *http.Request) (int, int, error) {
	page, limit := 1, 12
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page < 1 || page > 1000000 {
			return 0, 0, apperror.Validation("Page must be between 1 and 1000000.", nil)
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, apperror.Validation("Limit must be between 1 and 100.", nil)
		}
	}
	return page, limit, nil
}

// middleware provides a final panic boundary and lightweight structured request timing.
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("request panic", "panic", recovered)
				s.fail(w, errors.New("unexpected panic"))
			}
			s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
		}()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// health checks database readiness rather than reporting a process-only success.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.pool.Ping(r.Context()); err != nil {
		writeJSON(w, 503, map[string]any{"error": &apperror.Error{Code: "NOT_READY", Message: "Database is unavailable."}})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

// notFound preserves the JSON error envelope for unknown API paths.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.fail(w, apperror.Missing("Endpoint"))
}
