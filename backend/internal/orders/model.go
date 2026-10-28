package orders

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"stockflow/backend/internal/apperror"
	"stockflow/backend/internal/inventory"
)

// ItemInput accepts only identity and quantity; client-supplied prices are rejected by the decoder.
type ItemInput struct {
	SKU string `json:"sku"`
	Qty int    `json:"qty"`
}

// CreateInput represents an entire atomic order, limited to 100 distinct products.
type CreateInput struct {
	Customer string      `json:"customer"`
	Items    []ItemInput `json:"items"`
}

// Validate reports every malformed item before a transaction can write anything.
func (in *CreateInput) Validate() error {
	in.Customer = strings.TrimSpace(in.Customer)
	fields := map[string]string{}
	if utf8.RuneCountInString(in.Customer) < 1 || utf8.RuneCountInString(in.Customer) > 160 {
		fields["customer"] = "Customer must contain 1–160 characters."
	}
	if len(in.Items) < 1 || len(in.Items) > 100 {
		fields["items"] = "An order must contain 1–100 distinct products."
	}
	seen := map[string]bool{}
	for i := range in.Items {
		item := &in.Items[i]
		item.SKU = strings.TrimSpace(item.SKU)
		key := fmt.Sprintf("items.%d", i)
		if item.SKU == "" || utf8.RuneCountInString(item.SKU) > 64 {
			fields[key+".sku"] = "Choose a valid SKU."
		} else if seen[item.SKU] {
			fields[key+".sku"] = "Duplicate SKU: " + item.SKU + "."
		}
		seen[item.SKU] = true
		if item.Qty <= 0 || item.Qty > inventory.MaxInteger {
			fields[key+".qty"] = "Quantity must be a positive integer up to 2147483647."
		}
	}
	if len(fields) > 0 {
		return apperror.Validation("Please check the order fields.", fields)
	}
	return nil
}

// Item stores a historical price snapshot instead of consulting the current catalog price.
type Item struct {
	ID        int    `json:"id"`
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
}

// Order is the authoritative response for both creation, detail, and cancellation.
type Order struct {
	ID          int        `json:"id"`
	Customer    string     `json:"customer"`
	Status      string     `json:"status"`
	Total       int        `json:"total"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	Items       []Item     `json:"items"`
}

// Page supports the extra order-list endpoint without changing any required API shape.
type Page struct {
	Data  []Order `json:"data"`
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int     `json:"total"`
}
