package inventory

import "context"

// Service keeps request-independent validation out of HTTP handlers and SQL code.
type Service struct{ *Store }

// NewService assembles the small inventory application layer without a DI framework.
func NewService(store *Store) *Service { return &Service{Store: store} }

// Create validates and normalizes a catalog entry before inserting it.
func (s *Service) Create(ctx context.Context, in ProductInput) (Product, error) {
	if err := in.Validate(); err != nil {
		return Product{}, err
	}
	return s.Store.Create(ctx, in)
}

// Update validates metadata while the store protects immutable identity and stock.
func (s *Service) Update(ctx context.Context, id int, in ProductInput) (Product, error) {
	if err := in.Validate(); err != nil {
		return Product{}, err
	}
	return s.Store.Update(ctx, id, in)
}

// AdjustStock rejects invalid quantities before entering the balance-and-ledger transaction.
func (s *Service) AdjustStock(ctx context.Context, id int, in StockInput) (Product, error) {
	if err := in.Validate(); err != nil {
		return Product{}, err
	}
	return s.Store.AdjustStock(ctx, id, in)
}
