package inventory

import (
	"strings"
	"time"
	"unicode/utf8"

	"stockflow/backend/internal/apperror"
)

// MaxInteger is the PostgreSQL INTEGER ceiling; arithmetic uses int64 before persistence.
const MaxInteger = 2147483647

// Product is the public catalog projection, including a ledger-derived latest movement.
type Product struct {
	ID             int        `json:"id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Price          int        `json:"price"`
	Stock          int        `json:"stock"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

// ProductInput excludes stock: only the dedicated stock endpoint can change balances.
type ProductInput struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    *int   `json:"price"`
	IsActive *bool  `json:"is_active,omitempty"`
}

// Validate trims labels and returns all catalog field errors together.
func (in *ProductInput) Validate() error {
	in.SKU = strings.TrimSpace(in.SKU)
	in.Name = strings.TrimSpace(in.Name)
	fields := map[string]string{}
	if utf8.RuneCountInString(in.SKU) < 1 || utf8.RuneCountInString(in.SKU) > 64 {
		fields["sku"] = "SKU must contain 1–64 characters."
	}
	if utf8.RuneCountInString(in.Name) < 1 || utf8.RuneCountInString(in.Name) > 160 {
		fields["name"] = "Name must contain 1–160 characters."
	}
	if in.Price == nil || *in.Price < 0 || *in.Price > MaxInteger {
		fields["price"] = "Price must be an integer between 0 and 2147483647 IDR."
	}
	if len(fields) > 0 {
		return apperror.Validation("Please check the product fields.", fields)
	}
	return nil
}

// StockInput represents one manual ledger entry, never an absolute balance overwrite.
type StockInput struct {
	Type string `json:"type"`
	Qty  int    `json:"qty"`
	Note string `json:"note"`
}

// Validate checks the direction and prevents fractional, zero, oversized, or unbounded input.
func (in *StockInput) Validate() error {
	in.Note = strings.TrimSpace(in.Note)
	fields := map[string]string{}
	if in.Type != "in" && in.Type != "out" {
		fields["type"] = "Direction must be in or out."
	}
	if in.Qty <= 0 || in.Qty > MaxInteger {
		fields["qty"] = "Quantity must be a positive integer up to 2147483647."
	}
	if utf8.RuneCountInString(in.Note) > 500 {
		fields["note"] = "Note must contain at most 500 characters."
	}
	if len(fields) > 0 {
		return apperror.Validation("Please check the stock adjustment.", fields)
	}
	return nil
}

// Movement records a permanent inventory delta and an optional related sales order.
type Movement struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Type      string    `json:"type"`
	Qty       int       `json:"qty"`
	Note      *string   `json:"note"`
	OrderID   *int      `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ListParams describes server-side filtering and pagination with validated defaults.
type ListParams struct {
	Search string
	Filter string
	Page   int
	Limit  int
}

// ProductPage exactly preserves the assignment's paginated response shape.
type ProductPage struct {
	Data  []Product `json:"data"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
	Total int       `json:"total"`
}

// Summary aggregates the entire inventory, independent of current filters and pages.
type Summary struct {
	TotalProducts  int `json:"total_products"`
	ActiveProducts int `json:"active_products"`
	// Decimal text preserves exact whole-IDR valuation beyond JavaScript's safe integer range.
	TotalStockValue string `json:"total_stock_value"`
	LowStockCount   int    `json:"low_stock_count"`
}
