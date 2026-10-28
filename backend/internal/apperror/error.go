package apperror

import "fmt"

type Error struct {
	Status    int               `json:"-"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	SKU       string            `json:"sku,omitempty"`
	Available *int              `json:"available,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// Validation constructs a 400 response with optional field-level explanations.
func Validation(message string, fields map[string]string) *Error {
	return &Error{Status: 400, Code: "VALIDATION_ERROR", Message: message, Fields: fields}
}

// Missing constructs a consistent 404 for a missing resource.
func Missing(resource string) *Error {
	return &Error{Status: 404, Code: "NOT_FOUND", Message: resource + " was not found."}
}

// Conflict constructs a named business conflict without exposing SQL details.
func Conflict(code, message string) *Error {
	return &Error{Status: 409, Code: code, Message: message}
}

// Insufficient preserves zero availability and the exact SKU for inline conflict feedback.
func Insufficient(sku string, available int) *Error {
	return &Error{Status: 409, Code: "INSUFFICIENT_STOCK", Message: "Insufficient stock for " + sku + ".", SKU: sku, Available: &available}
}
