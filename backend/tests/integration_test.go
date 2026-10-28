// Each test uses its own PostgreSQL schema.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"stockflow/backend/internal/database"
	"stockflow/backend/internal/httpapi"
	"stockflow/backend/internal/inventory"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testSequence atomic.Int64

type fixture struct {
	pool   *pgxpool.Pool
	server *httptest.Server
}

func setup(t *testing.T) *fixture {
	t.Helper()
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("set TEST_DATABASE_URL to run real PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("sf_test_%d_%d", time.Now().UnixNano(), testSequence.Add(1))
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	parsed, err := url.Parse(connection)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	pool, err := database.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err = database.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.New(pool, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(server.Close)
	return &fixture{pool: pool, server: server}
}

func (f *fixture) request(t *testing.T, method, path, body string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, f.server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, data
}

// Opening stock goes through the API so it also creates a movement.
func (f *fixture) product(t *testing.T, sku string, stock int) inventory.Product {
	t.Helper()
	status, body := f.request(t, "POST", "/api/products", fmt.Sprintf(`{"sku":%q,"name":"Test product","price":15000}`, sku))
	if status != 201 {
		t.Fatalf("create: %d %s", status, body)
	}
	var product inventory.Product
	if err := json.Unmarshal(body, &product); err != nil {
		t.Fatal(err)
	}
	if stock > 0 {
		status, body = f.request(t, "POST", fmt.Sprintf("/api/products/%d/stock", product.ID), fmt.Sprintf(`{"type":"in","qty":%d,"note":"Test opening stock"}`, stock))
		if status != 200 {
			t.Fatalf("opening stock: %d %s", status, body)
		}
	}
	return product
}

// Counts orders, order items, and stock movements, in that order.
func (f *fixture) counts(t *testing.T) [3]int {
	t.Helper()
	var counts [3]int
	err := f.pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM sales_orders),(SELECT count(*) FROM sales_order_items),(SELECT count(*) FROM stock_movements)`).Scan(&counts[0], &counts[1], &counts[2])
	if err != nil {
		t.Fatal(err)
	}
	return counts
}

func (f *fixture) balance(t *testing.T, sku string) int {
	t.Helper()
	var stock int
	if err := f.pool.QueryRow(context.Background(), `SELECT stock FROM products WHERE sku=$1`, sku).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	return stock
}

func TestInsufficientOrderWritesNothing(t *testing.T) {
	f := setup(t)
	f.product(t, "TEST-A", 10)
	f.product(t, "TEST-B", 2)
	before := f.counts(t)
	status, body := f.request(t, "POST", "/api/orders", `{"customer":"Rollback customer","items":[{"sku":"TEST-A","qty":3},{"sku":"TEST-B","qty":4}]}`)
	if status != 409 || !bytes.Contains(body, []byte(`"code":"INSUFFICIENT_STOCK"`)) || !bytes.Contains(body, []byte(`"sku":"TEST-B"`)) || !bytes.Contains(body, []byte(`"available":2`)) {
		t.Fatalf("conflict envelope: %d %s", status, body)
	}
	if f.counts(t) != before || f.balance(t, "TEST-A") != 10 || f.balance(t, "TEST-B") != 2 {
		t.Fatal("failed order changed persisted state")
	}
}

func TestStockAdjustmentLedger(t *testing.T) {
	f := setup(t)
	p := f.product(t, "TEST-STOCK", 5)
	before := f.counts(t)
	status, body := f.request(t, "POST", fmt.Sprintf("/api/products/%d/stock", p.ID), `{"type":"out","qty":3,"note":"Dispatch"}`)
	if status != 200 {
		t.Fatalf("adjustment: %d %s", status, body)
	}
	if f.balance(t, p.SKU) != 2 || f.counts(t)[2] != before[2]+1 {
		t.Fatal("successful adjustment did not update both ledger and balance")
	}
	after := f.counts(t)
	status, body = f.request(t, "POST", fmt.Sprintf("/api/products/%d/stock", p.ID), `{"type":"out","qty":3}`)
	if status != 409 || !bytes.Contains(body, []byte(`"available":2`)) {
		t.Fatalf("overdraw: %d %s", status, body)
	}
	if f.balance(t, p.SKU) != 2 || f.counts(t) != after {
		t.Fatal("rejected overdraw changed state")
	}
}
