package orders

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"stockflow/backend/internal/apperror"
	"stockflow/backend/internal/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store owns transaction boundaries for creation and cancellation, including movement writes.
type Store struct{ pool *pgxpool.Pool }

// NewStore connects order persistence to the shared database pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// lockedProduct contains only the authoritative values required to validate a sale.
type lockedProduct struct {
	ID     int
	Price  int
	Stock  int
	Active bool
}

// Create validates all locked products before writing any order, balance, or movement.
func (s *Store) Create(ctx context.Context, in CreateInput) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	// Consistent SKU ordering prevents cross-item deadlocks between creation and cancellation.
	sorted := append([]ItemInput(nil), in.Items...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SKU < sorted[j].SKU })
	products := map[string]lockedProduct{}
	fields := map[string]string{}
	for _, item := range sorted {
		var p lockedProduct
		err = tx.QueryRow(ctx, `SELECT id,price,stock,is_active FROM products WHERE sku=$1 FOR UPDATE`, item.SKU).Scan(&p.ID, &p.Price, &p.Stock, &p.Active)
		if errors.Is(err, pgx.ErrNoRows) {
			fields[item.SKU] = "Unknown SKU: " + item.SKU
			continue
		}
		if err != nil {
			return Order{}, err
		}
		if !p.Active {
			fields[item.SKU] = "Product " + item.SKU + " is inactive."
		}
		products[item.SKU] = p
	}
	if len(fields) > 0 {
		return Order{}, apperror.Validation("One or more products cannot be ordered.", fields)
	}
	var total int64
	for _, item := range sorted {
		p := products[item.SKU]
		if p.Stock < item.Qty {
			return Order{}, apperror.Insufficient(item.SKU, p.Stock)
		}
		total += int64(p.Price) * int64(item.Qty)
		// Check each addition before the next item so large requests cannot overflow int64.
		if total > inventory.MaxInteger {
			return Order{}, apperror.Validation("Order total exceeds 2147483647 IDR. Split this order.", nil)
		}
	}
	var id int
	if err = tx.QueryRow(ctx, `INSERT INTO sales_orders(customer,total) VALUES($1,$2) RETURNING id`, in.Customer, total).Scan(&id); err != nil {
		return Order{}, err
	}
	for _, item := range sorted {
		p := products[item.SKU]
		result, updateErr := tx.Exec(ctx, `UPDATE products SET stock=stock-$2 WHERE id=$1 AND stock>=$2`, p.ID, item.Qty)
		if updateErr != nil {
			return Order{}, updateErr
		}
		if result.RowsAffected() != 1 {
			return Order{}, apperror.Insufficient(item.SKU, p.Stock)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO sales_order_items(order_id,sku,qty,unit_price) VALUES($1,$2,$3,$4)`, id, item.SKU, item.Qty, p.Price); err != nil {
			return Order{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stock_movements(product_id,type,qty,note,order_id) VALUES($1,'out',$2,$3,$4)`, p.ID, item.Qty, fmt.Sprintf("Sales order #%d", id), id); err != nil {
			return Order{}, err
		}
	}
	out, err := readOrder(ctx, tx, id)
	if err != nil {
		return Order{}, err
	}
	return out, tx.Commit(ctx)
}

// Cancel serializes on the order and restores each item once, preserving the original order.
func (s *Store) Cancel(ctx context.Context, id int) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM sales_orders WHERE id=$1 FOR UPDATE`, id).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return Order{}, apperror.Missing("Order")
	} else if err != nil {
		return Order{}, err
	}
	if status == "CANCELLED" {
		return Order{}, apperror.Conflict("ORDER_ALREADY_CANCELLED", "This order has already been cancelled.")
	}
	order, err := readOrder(ctx, tx, id)
	if err != nil {
		return Order{}, err
	}
	sort.Slice(order.Items, func(i, j int) bool { return order.Items[i].SKU < order.Items[j].SKU })
	// Validate every restoration before writes, including the INTEGER storage ceiling.
	for _, item := range order.Items {
		var stock int
		if err = tx.QueryRow(ctx, `SELECT stock FROM products WHERE sku=$1 FOR UPDATE`, item.SKU).Scan(&stock); err != nil {
			return Order{}, err
		}
		if int64(stock)+int64(item.Qty) > inventory.MaxInteger {
			return Order{}, apperror.Conflict("STOCK_LIMIT_EXCEEDED", "Restoring "+item.SKU+" would exceed the stock limit.")
		}
	}
	for _, item := range order.Items {
		var productID int
		if err = tx.QueryRow(ctx, `UPDATE products SET stock=stock+$2 WHERE sku=$1 AND stock::bigint+$2<=2147483647 RETURNING id`, item.SKU, item.Qty).Scan(&productID); err != nil {
			return Order{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO stock_movements(product_id,type,qty,note,order_id) VALUES($1,'in',$2,$3,$4)`, productID, item.Qty, fmt.Sprintf("Cancellation of order #%d", id), id); err != nil {
			return Order{}, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE sales_orders SET status='CANCELLED',cancelled_at=now() WHERE id=$1`, id); err != nil {
		return Order{}, err
	}
	out, err := readOrder(ctx, tx, id)
	if err != nil {
		return Order{}, err
	}
	return out, tx.Commit(ctx)
}

// querier allows detail reads to share either a pool or an already-open transaction.
type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// readOrder reads the header and immutable item snapshots through the supplied transaction boundary.
func readOrder(ctx context.Context, q querier, id int) (Order, error) {
	var out Order
	out.Items = []Item{}
	err := q.QueryRow(ctx, `SELECT id,customer,status,total,created_at,cancelled_at FROM sales_orders WHERE id=$1`, id).Scan(&out.ID, &out.Customer, &out.Status, &out.Total, &out.CreatedAt, &out.CancelledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, apperror.Missing("Order")
	}
	if err != nil {
		return out, err
	}
	rows, err := q.Query(ctx, `SELECT id,sku,qty,unit_price FROM sales_order_items WHERE order_id=$1 ORDER BY id`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		if err = rows.Scan(&item.ID, &item.SKU, &item.Qty, &item.UnitPrice); err != nil {
			return out, err
		}
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}

// Get reads a consistent order snapshot even while another request cancels it.
func (s *Store) Get(ctx context.Context, id int) (Order, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)
	out, err := readOrder(ctx, tx, id)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

// List returns recent order headers; detail includes the full item collection.
func (s *Store) List(ctx context.Context, page, limit int) (Page, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Page{}, err
	}
	defer tx.Rollback(ctx)
	out := Page{Data: []Order{}, Page: page, Limit: limit}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM sales_orders`).Scan(&out.Total); err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `SELECT id,customer,status,total,created_at,cancelled_at FROM sales_orders ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2`, limit, (page-1)*limit)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var o Order
		o.Items = []Item{}
		if err = rows.Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Data = append(out.Data, o)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
