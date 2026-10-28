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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"stockflow/backend/internal/database"
	"stockflow/backend/internal/httpapi"
	"stockflow/backend/internal/inventory"
	"stockflow/backend/internal/orders"

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
func TestOrderPricesCancellationAndHistory(t *testing.T) {
	f := setup(t)
	p := f.product(t, "TEST-SALE", 8)
	before := f.counts(t)
	status, body := f.request(t, "POST", "/api/orders", `{"customer":"Happy customer","items":[{"sku":"TEST-SALE","qty":3}]}`)
	if status != 201 {
		t.Fatalf("order: %d %s", status, body)
	}
	var order orders.Order
	if err := json.Unmarshal(body, &order); err != nil {
		t.Fatal(err)
	}
	if order.Total != 45000 || len(order.Items) != 1 || order.Items[0].UnitPrice != 15000 || f.balance(t, p.SKU) != 5 || f.counts(t)[2] != before[2]+1 {
		t.Fatal("order did not use server prices and write stock movement")
	}
	status, body = f.request(t, "POST", fmt.Sprintf("/api/orders/%d/cancel", order.ID), "")
	if status != 200 || f.balance(t, p.SKU) != 8 {
		t.Fatalf("cancel: %d %s", status, body)
	}
	after := f.counts(t)
	status, body = f.request(t, "POST", fmt.Sprintf("/api/orders/%d/cancel", order.ID), "")
	if status != 409 || !bytes.Contains(body, []byte("ORDER_ALREADY_CANCELLED")) || f.balance(t, p.SKU) != 8 || f.counts(t) != after {
		t.Fatalf("repeat cancel: %d %s", status, body)
	}
	status, body = f.request(t, "DELETE", fmt.Sprintf("/api/products/%d", p.ID), "")
	if status != 409 || !bytes.Contains(body, []byte("PRODUCT_HAS_HISTORY")) {
		t.Fatalf("delete history: %d %s", status, body)
	}
}

func TestConcurrentOrdersNeverOversell(t *testing.T) {
	f := setup(t)
	f.product(t, "TEST-RACE", 10)
	before := f.counts(t)
	var success, conflict atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	failures := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			response, err := http.Post(f.server.URL+"/api/orders", "application/json", strings.NewReader(`{"customer":"Concurrent buyer","items":[{"sku":"TEST-RACE","qty":1}]}`))
			if err != nil {
				failures <- err.Error()
				return
			}
			defer response.Body.Close()
			if response.StatusCode == 201 {
				success.Add(1)
			} else if response.StatusCode == 409 {
				conflict.Add(1)
			} else {
				body, _ := io.ReadAll(response.Body)
				failures <- fmt.Sprintf("status %d: %s", response.StatusCode, body)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if success.Load() != 10 || conflict.Load() != 10 || f.balance(t, "TEST-RACE") != 0 || f.counts(t) != [3]int{before[0] + 10, before[1] + 10, before[2] + 10} {
		t.Fatalf("oversell or ledger mismatch: successes=%d conflicts=%d stock=%d counts=%v", success.Load(), conflict.Load(), f.balance(t, "TEST-RACE"), f.counts(t))
	}
}

func TestConcurrentCancellationRestoresOnce(t *testing.T) {
	f := setup(t)
	f.product(t, "TEST-CANCEL", 5)
	status, body := f.request(t, "POST", "/api/orders", `{"customer":"Concurrent cancellation","items":[{"sku":"TEST-CANCEL","qty":2}]}`)
	if status != 201 {
		t.Fatalf("order: %d %s", status, body)
	}
	var order orders.Order
	if err := json.Unmarshal(body, &order); err != nil {
		t.Fatal(err)
	}
	before := f.counts(t)
	var success, conflict atomic.Int32
	var wg sync.WaitGroup
	failures := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, err := http.Post(fmt.Sprintf("%s/api/orders/%d/cancel", f.server.URL, order.ID), "application/json", nil)
			if err != nil {
				failures <- err.Error()
				return
			}
			defer response.Body.Close()
			if response.StatusCode == 200 {
				success.Add(1)
			} else if response.StatusCode == 409 {
				conflict.Add(1)
			} else {
				failures <- fmt.Sprintf("unexpected cancellation status %d", response.StatusCode)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	if success.Load() != 1 || conflict.Load() != 7 || f.balance(t, "TEST-CANCEL") != 5 || f.counts(t)[2] != before[2]+1 {
		t.Fatal("cancellation restored more than once")
	}
}

func TestValidationAndCatalogPages(t *testing.T) {
	f := setup(t)
	before := f.counts(t)
	for _, body := range []string{`{"customer":"X","items":[]}`, `{"customer":"X","items":[{"sku":"MISSING","qty":1}]}`, `{"customer":"X","items":[{"sku":"KB-001","qty":0}]}`, `{"customer":"X","items":[{"sku":"KB-001","qty":1},{"sku":"KB-001","qty":1}]}`, `{"customer":"X","items":[{"sku":"KB-001","qty":1,"unit_price":1}]}`} {
		status, response := f.request(t, "POST", "/api/orders", body)
		if status != 400 {
			t.Errorf("expected validation: %d %s", status, response)
		}
	}
	if f.counts(t) != before {
		t.Fatal("invalid request wrote rows")
	}
	status, body := f.request(t, "GET", "/api/products?search=not-a-real-product", "")
	if status != 200 || !bytes.Contains(body, []byte(`"data":[]`)) || !bytes.Contains(body, []byte(`"total":0`)) {
		t.Fatalf("empty page: %d %s", status, body)
	}
	status, body = f.request(t, "GET", "/api/products?search=logitech&limit=1&page=2", "")
	var page inventory.ProductPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatal(err)
	}
	if status != 200 || page.Total != 3 || len(page.Data) != 1 {
		t.Fatalf("search pagination: %d %s", status, body)
	}
	status, body = f.request(t, "GET", "/api/products?limit=0", "")
	if status != 400 {
		t.Fatalf("unbounded page: %d %s", status, body)
	}
}

func TestDatabaseFailureRollsBack(t *testing.T) {
	f := setup(t)
	f.product(t, "TEST-FAIL", 5)
	before := f.counts(t)
	// A fixture-only trigger simulates a persistence failure after the product and item writes.
	_, err := f.pool.Exec(context.Background(), `CREATE FUNCTION reject_test_movement() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.note LIKE 'Sales order%' THEN RAISE EXCEPTION 'simulated movement failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_test_movement BEFORE INSERT ON stock_movements FOR EACH ROW EXECUTE FUNCTION reject_test_movement()`)
	if err != nil {
		t.Fatal(err)
	}
	status, body := f.request(t, "POST", "/api/orders", `{"customer":"Failure customer","items":[{"sku":"TEST-FAIL","qty":2}]}`)
	if status != 500 || bytes.Contains(body, []byte("simulated movement failure")) {
		t.Fatalf("failure boundary: %d %s", status, body)
	}
	if f.balance(t, "TEST-FAIL") != 5 || f.counts(t) != before {
		t.Fatal("database error partially committed the order")
	}
}

func TestLargeInventoryValuePreservesIntegerPrecision(t *testing.T) {
	f := setup(t)
	// Direct fixture setup reaches the numeric boundary without millions of stock operations.
	if _, err := f.pool.Exec(context.Background(), `INSERT INTO products(sku,name,price,stock) VALUES ('HUGE-A','Large valuation fixture',2147483647,2147483647),('HUGE-B','Large valuation fixture',2147483647,2147483647),('HUGE-C','Large valuation fixture',2147483647,2147483647)`); err != nil {
		t.Fatal(err)
	}
	status, body := f.request(t, "GET", "/api/products/summary", "")
	var summary inventory.Summary
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatal(err)
	}
	var expected string
	if err := f.pool.QueryRow(context.Background(), `SELECT sum(price::bigint*stock)::text FROM products`).Scan(&expected); err != nil {
		t.Fatal(err)
	}
	if status != 200 || summary.TotalStockValue != expected {
		t.Fatalf("inexact valuation: %d %s", status, body)
	}
}
