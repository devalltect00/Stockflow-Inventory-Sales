# StockFlow

Author: Rizky Purwanto Fernandes

An inventory and sales app built with React, TypeScript, Go, and PostgreSQL.
It handles products, stock adjustments, and sales orders. Creating an order
reduces stock; cancelling it restores stock and keeps the movement history.

The frontend uses Vite, React Router, TanStack Query, Zod, and Lucide icons.
The backend uses Go's `net/http`, pgx for PostgreSQL, and Goose for migrations.

<a href="docs/stockflow-inventory.png">
  <img src="docs/stockflow-inventory.png" alt="StockFlow inventory page" width="800">
</a>

The project diagrams are in [stockflow.drawio](docs/diagrams/stockflow.drawio).
Open the file with [draw.io](https://app.diagrams.net/) to view or edit them.

## Run with Docker

Docker Desktop must be running with Linux containers. Run this from the project
root:

```cmd
docker compose up -d --build
```

| Service          | Address                          |
| ---------------- | -------------------------------- |
| Frontend         | http://localhost:3000            |
| API health check | http://localhost:8080/api/health |
| PostgreSQL       | `localhost:5433`                 |

The backend applies migrations before starting the API. The seed adds 16
products, opening stock movements, and two sample orders. Goose tracks applied
migrations, so restarting the app does not insert the seed again.

Default database settings:

```text
Database: stockflow
Username: stockflow
Password: stockflow_local
```

Compose provides these defaults without a `.env` file. Optional database and port
overrides are listed in [.env.example](.env.example); Compose reads overrides from
the shell or a root `.env` file. Updating those values does not change the password
of an already initialized database.

Useful commands, also run from the project root:

```cmd
docker compose ps
docker compose logs --tail 50 backend frontend
docker compose down
```

Database files are stored in the named `stockflow_data` volume.
`docker compose down` stops and removes the containers but keeps that volume.

## Local development

For development outside Docker, use Go 1.27.1 or newer and Node.js 24 with npm.
PostgreSQL can stay in Docker.

If the full Docker app is running, stop its frontend and backend first:

```cmd
docker compose stop frontend backend
docker compose up -d db
```

In one terminal, starting from the project root:

```cmd
cd backend
go run ./cmd/api
```

In another terminal, also starting from the project root:

```cmd
cd frontend
npm ci
npm run dev
```

Open http://localhost:5173. The local backend defaults to PostgreSQL on port 5433.
If you change the database settings, set `DATABASE_URL` in the backend terminal.
The Go app reads environment variables directly; it does not load `.env` files.

API requests use relative `/api` URLs. Vite forwards them to
`http://127.0.0.1:8080` during development. In Docker, Nginx serves the built
frontend and forwards them to `http://backend:8080`.

## Features

- Search products by name or SKU, filter by status or low stock, and paginate.
- Add and edit products, adjust stock, and view movement history.
- Create sales orders using current product prices from the backend.
- View order details and cancel an order to restore its stock.
- See total products, active products, inventory value, and low-stock counts.

The stock form and order form display backend errors, including the failed SKU
and available quantity when stock is insufficient. To check the empty state,
search for a name that does not exist. To check error and retry handling, stop
the backend, reload the page, then start it again and retry.

## API

| Method | Path                           | Purpose                               |
| ------ | ------------------------------ | ------------------------------------- |
| GET    | `/api/health`                  | Check database connectivity           |
| GET    | `/api/products`                | Search, filter, and paginate products |
| GET    | `/api/products/summary`        | Inventory summary                     |
| POST   | `/api/products`                | Create a product                      |
| PUT    | `/api/products/{id}`           | Update product details                |
| DELETE | `/api/products/{id}`           | Delete a product without history      |
| POST   | `/api/products/{id}/stock`     | Add or remove stock                   |
| GET    | `/api/products/{id}/movements` | Movement history, newest first        |
| GET    | `/api/orders`                  | Paginated order list                  |
| POST   | `/api/orders`                  | Create an order                       |
| GET    | `/api/orders/{id}`             | Order details and items               |
| POST   | `/api/orders/{id}/cancel`      | Cancel an order                       |

Product listing accepts `search`, `page`, `limit`, and `filter`. Filter values are
`all`, `active`, `inactive`, and `low-stock`. Pagination defaults to page 1 and
12 rows, with a maximum limit of 100. Lists return `{data, page, limit, total}`;
movement history returns `{data}`.

For manual API checks, open [backend/api.http](backend/api.http) with the VS Code
REST Client extension. Send the named creation requests before the requests
that reuse their IDs. Successful writes remain in the database.

These curl examples use Windows CMD quoting:

```cmd
curl "http://localhost:8080/api/products?search=keyboard&page=1&limit=12"

curl -i -X POST http://localhost:8080/api/products ^
  -H "Content-Type: application/json" ^
  -d "{\"sku\":\"DEMO-001\",\"name\":\"Wireless keyboard\",\"price\":150000,\"is_active\":true}"
```

Replace `17` below with the product ID returned by the create request:

```cmd
curl -X POST http://localhost:8080/api/products/17/stock ^
  -H "Content-Type: application/json" ^
  -d "{\"type\":\"in\",\"qty\":10,\"note\":\"Supplier delivery\"}"

curl -i -X POST http://localhost:8080/api/orders ^
  -H "Content-Type: application/json" ^
  -d "{\"customer\":\"Demo customer\",\"items\":[{\"sku\":\"DEMO-001\",\"qty\":2}]}"
```

Validation errors return HTTP 400. Conflicts such as insufficient stock return
HTTP 409, with details inside the same error envelope:

```json
{
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "Insufficient stock for DEMO-001.",
    "sku": "DEMO-001",
    "available": 2
  }
}
```

## Tests

### Backend

Run the tests against PostgreSQL in Docker:

```cmd
docker compose --profile test run --build --rm backend-tests
```

This runs `go test -race ./... -count=1`. The test service is separate from normal
app startup. Each integration test creates its own database schema and removes
it afterward.

The current tests cover product price validation, order input validation, and
eight database integration cases:

- An insufficient multi-item order leaves stock, orders, items, and movements unchanged.
- A successful stock adjustment updates stock and writes a movement; a rejected
  overdraw changes neither.
- Orders use server prices, cancellations restore stock, and products with history
  cannot be deleted.
- Concurrent orders cannot sell more stock than is available.
- Concurrent cancellation restores stock only once.
- Invalid orders are rejected, and product search and pagination return the expected results.
- A database failure during order creation rolls back all writes.
- Large inventory totals keep their integer precision.

To run them locally in CMD, start the Docker database, then run from the project
root:

```cmd
cd backend
set "TEST_DATABASE_URL=postgres://stockflow:stockflow_local@localhost:5433/stockflow?sslmode=disable"
go test -v ./... -count=1
go vet ./...
```

Without `TEST_DATABASE_URL`, the unit tests run but database integration tests
are skipped. Running `go test -race` locally also requires a compatible C compiler.

### Frontend

From the project root:

```cmd
cd frontend
npm ci
npm run validate
```

`validate` runs Prettier checks, ESLint, the TypeScript and Vite build, and Vitest.
Use `npm test` to run just the tests, or `npm run test:watch` while developing.

Frontend tests cover validation, formatting, API error parsing, and the stock
dialog's insufficient-stock message. API responses in these tests are mocked;
browser end-to-end tests are not included yet.

[GitHub Actions](.github/workflows/ci.yml) runs backend formatting, `go vet`, and
tests with PostgreSQL and the race detector. It also runs the frontend validation
command on pushes and pull requests.

## Implementation notes

- Order creation uses one transaction for the order, items, stock updates, and
  movements. Product rows are locked in SKU order, and stock updates also check
  the available balance in SQL. Any failure rolls back the transaction.
- Cancellation locks the order before restoring stock. A second cancellation
  returns a conflict instead of restoring stock twice.
- Product prices come from the database. Prices and order totals are integer
  IDR. The inventory summary returns `total_stock_value` as a decimal string,
  which the frontend formats using `BigInt`.
- `last_movement_at` is calculated from the movements table. It is `null` when
  a product has no movements.
- New products start with zero stock. Stock changes go through the adjustment
  endpoint. SKUs cannot be changed, and products with history cannot be deleted.
- Low stock means 10 units or fewer. Inactive products are included in inventory
  counts and valuation but cannot be added to new sales orders.
- Successful frontend mutations refresh the affected TanStack Query data,
  including the product list and summary cards.

## Possible improvements

Add login and sign-up with administrator approval or invitations. Registering an
account should not automatically grant access. Permissions would control who can
view inventory, edit products, adjust stock, and create or cancel orders. These
checks would apply to backend API requests as well as the frontend.

## Time spent

Approximately 8 hours in total, including planning, learning, database setup,
backend and frontend development, testing, and bug fixes. This is an estimate;
time was not tracked precisely.
