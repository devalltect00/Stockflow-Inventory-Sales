package orders

import "context"

// Service provides order operations to HTTP handlers without coupling rules to transport.
type Service struct{ *Store }

// NewService explicitly wires the order persistence dependency.
func NewService(store *Store) *Service { return &Service{Store: store} }

// Create validates the whole request before Store acquires locks or writes rows.
func (s *Service) Create(ctx context.Context, in CreateInput) (Order, error) {
	if err := in.Validate(); err != nil {
		return Order{}, err
	}
	return s.Store.Create(ctx, in)
}
