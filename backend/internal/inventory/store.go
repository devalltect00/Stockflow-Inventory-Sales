package inventory

import (
	"context"
	"errors"
	"strings"
	"time"

	"stockflow/backend/internal/apperror"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store owns catalog SQL; callers cannot bypass transactions for stock changes.
type Store struct{ pool *pgxpool.Pool }

// NewStore binds catalog persistence to the application's shared connection pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// productColumns derives the latest movement through an indexed correlated lookup.
const productColumns = `p.id,p.sku,p.name,p.price,p.stock,p.is_active,p.created_at,
(SELECT max(m.created_at) FROM stock_movements m WHERE m.product_id=p.id)`

// scanProduct keeps the SQL projection and public product representation aligned.
func scanProduct(row pgx.Row) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt)
	return p, err
}

// List returns one filtered page and its count from a single consistent statement snapshot.
func (s *Store) List(ctx context.Context, in ListParams) (ProductPage, error) {
	// Escape SQL LIKE wildcards so search is a literal, case-insensitive substring.
	search := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(in.Search) + "%"
	rows, err := s.pool.Query(ctx, `WITH filtered AS (
	 SELECT `+productColumns+` AS last_movement_at FROM products p
	 WHERE (p.sku ILIKE $1 OR p.name ILIKE $1)
	 AND ($2='all' OR ($2='active' AND p.is_active) OR ($2='inactive' AND NOT p.is_active) OR ($2='low-stock' AND p.stock<=10))
	), page AS (SELECT * FROM filtered ORDER BY id DESC LIMIT $3 OFFSET $4)
	SELECT (SELECT count(*) FROM filtered),page.* FROM (SELECT 1) anchor LEFT JOIN page ON true`, search, in.Filter, in.Limit, (in.Page-1)*in.Limit)
	if err != nil {
		return ProductPage{}, err
	}
	defer rows.Close()
	result := ProductPage{Data: []Product{}, Page: in.Page, Limit: in.Limit}
	for rows.Next() {
		var p Product
		var id *int
		var sku, name *string
		var price, stock *int
		var active *bool
		var createdAt *time.Time
		if err = rows.Scan(&result.Total, &id, &sku, &name, &price, &stock, &active, &createdAt, &p.LastMovementAt); err != nil {
			return ProductPage{}, err
		}
		if id != nil {
			p.ID = *id
			p.SKU = *sku
			p.Name = *name
			p.Price = *price
			p.Stock = *stock
			p.IsActive = *active
			p.CreatedAt = *createdAt
			result.Data = append(result.Data, p)
		}
	}
	return result, rows.Err()
}

// Get returns one product or a stable public 404 error.
func (s *Store) Get(ctx context.Context, id int) (Product, error) {
	p, err := scanProduct(s.pool.QueryRow(ctx, `SELECT `+productColumns+` FROM products p WHERE p.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, apperror.Missing("Product")
	}
	return p, err
}

// Create inserts a zero-stock product; any opening stock must use AdjustStock.
func (s *Store) Create(ctx context.Context, in ProductInput) (Product, error) {
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	p, err := scanProduct(s.pool.QueryRow(ctx, `INSERT INTO products AS p(sku,name,price,is_active) VALUES($1,$2,$3,$4) RETURNING p.id,p.sku,p.name,p.price,p.stock,p.is_active,p.created_at,NULL::timestamptz`, in.SKU, in.Name, in.Price, active))
	return p, catalogError(err)
}

// Update preserves stable SKUs and the stock ledger while editing product metadata.
func (s *Store) Update(ctx context.Context, id int, in ProductInput) (Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback(ctx)
	var sku string
	if err = tx.QueryRow(ctx, `SELECT sku FROM products WHERE id=$1 FOR UPDATE`, id).Scan(&sku); errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apperror.Missing("Product")
	} else if err != nil {
		return Product{}, err
	}
	if sku != in.SKU {
		return Product{}, apperror.Conflict("SKU_IMMUTABLE", "SKU cannot be changed after creation.")
	}
	_, err = tx.Exec(ctx, `UPDATE products SET name=$2,price=$3,is_active=COALESCE($4,is_active) WHERE id=$1`, id, in.Name, in.Price, in.IsActive)
	if err != nil {
		return Product{}, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Product{}, err
	}
	return s.Get(ctx, id)
}

// Delete locks the product so concurrent ledger writes cannot bypass the history guard.
func (s *Store) Delete(ctx context.Context, id int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var exists int
	if err = tx.QueryRow(ctx, `SELECT id FROM products WHERE id=$1 FOR UPDATE`, id).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return apperror.Missing("Product")
	} else if err != nil {
		return err
	}
	var history bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stock_movements WHERE product_id=$1) OR EXISTS(SELECT 1 FROM sales_order_items WHERE sku=(SELECT sku FROM products WHERE id=$1))`, id).Scan(&history); err != nil {
		return err
	}
	if history {
		return apperror.Conflict("PRODUCT_HAS_HISTORY", "This product has inventory history. Deactivate it instead.")
	}
	if _, err = tx.Exec(ctx, `DELETE FROM products WHERE id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AdjustStock atomically changes a locked balance and inserts its matching movement.
func (s *Store) AdjustStock(ctx context.Context, id int, in StockInput) (Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback(ctx)
	var sku string
	var stock int
	if err = tx.QueryRow(ctx, `SELECT sku,stock FROM products WHERE id=$1 FOR UPDATE`, id).Scan(&sku, &stock); errors.Is(err, pgx.ErrNoRows) {
		return Product{}, apperror.Missing("Product")
	} else if err != nil {
		return Product{}, err
	}
	delta := in.Qty
	if in.Type == "out" {
		if stock < in.Qty {
			return Product{}, apperror.Insufficient(sku, stock)
		}
		delta = -in.Qty
	}
	if int64(stock)+int64(delta) > MaxInteger {
		return Product{}, apperror.Validation("Stock would exceed the supported integer limit.", nil)
	}
	result, err := tx.Exec(ctx, `UPDATE products SET stock=stock+$2 WHERE id=$1 AND stock::bigint+$2 BETWEEN 0 AND 2147483647`, id, delta)
	if err != nil {
		return Product{}, err
	}
	if result.RowsAffected() != 1 {
		return Product{}, apperror.Insufficient(sku, stock)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO stock_movements(product_id,type,qty,note) VALUES($1,$2,$3,NULLIF($4,''))`, id, in.Type, in.Qty, in.Note); err != nil {
		return Product{}, err
	}
	p, err := scanProduct(tx.QueryRow(ctx, `SELECT `+productColumns+` FROM products p WHERE p.id=$1`, id))
	if err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}

// Movements returns permanent history newest first, distinguishing an empty ledger from a missing product.
func (s *Store) Movements(ctx context.Context, id int) ([]Movement, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id,product_id,type,qty,note,order_id,created_at FROM stock_movements WHERE product_id=$1 ORDER BY created_at DESC,id DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Movement{}
	for rows.Next() {
		var m Movement
		if err = rows.Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.OrderID, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

// Summary computes global counts and valuation in PostgreSQL, never from a paginated UI list.
func (s *Store) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	err := s.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE is_active),COALESCE(sum(price::bigint*stock),0),count(*) FILTER(WHERE stock<=10) FROM products`).Scan(&out.TotalProducts, &out.ActiveProducts, &out.TotalStockValue, &out.LowStockCount)
	return out, err
}

// catalogError maps the unique-SKU constraint to an actionable domain conflict.
func catalogError(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return apperror.Conflict("SKU_EXISTS", "A product with this SKU already exists.")
	}
	return err
}
