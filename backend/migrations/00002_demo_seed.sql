-- Assessment-only demo data. Version tracking ensures restarts never reseed inventory.
-- +goose Up
INSERT INTO products (sku, name, price) VALUES
('KB-001', 'Logitech K380 Keyboard', 485000),
('MS-001', 'Logitech M331 Silent Mouse', 225000),
('CB-001', 'Anker USB-C Cable 1m', 95000),
('HB-001', 'UGREEN USB-C Hub 6-in-1', 425000),
('MN-001', 'Dell 24-inch IPS Monitor', 2150000),
('ST-001', 'Aluminium Laptop Stand', 275000),
('WC-001', 'Logitech C270 Webcam', 395000),
('HP-001', 'JBL Tune 500 Headphones', 345000),
('PD-001', 'Desk Mat — Slate', 125000),
('CH-001', 'Anker 30W USB-C Charger', 295000),
('FD-001', 'SanDisk Ultra 64GB USB Drive', 89000),
('NT-001', 'Hardcover A5 Notebook', 45000),
('PN-001', 'Gel Pen Set — Black', 35000),
('BG-001', 'Laptop Sleeve 14-inch', 185000),
('AD-001', 'HDMI Adapter — Legacy', 75000),
('LB-001', 'Thermal Shipping Labels', 65000);

-- Opening balances also have ledger entries, keeping displayed stock explainable.
INSERT INTO stock_movements (product_id, type, qty, note, created_at)
SELECT id, 'in', (ARRAY[32,48,80,18,12,22,15,9,35,24,40,60,75,16,6,10])[id],
       'Opening warehouse balance', now() - interval '5 days' FROM products;
UPDATE products p SET stock = m.qty FROM stock_movements m WHERE m.product_id = p.id;
UPDATE products SET is_active = false WHERE sku = 'AD-001';

INSERT INTO sales_orders (customer, total, created_at) VALUES
('Nusantara Creative Studio', 1160000, now() - interval '2 hours'),
('Kopi Ruang — Jakarta', 910000, now() - interval '1 day');
INSERT INTO sales_order_items (order_id, sku, qty, unit_price) VALUES
(1,'KB-001',2,485000), (1,'CB-001',2,95000),
(2,'MS-001',3,225000), (2,'PD-001',1,125000), (2,'NT-001',2,45000);

-- Correct the seeded total from the same historical item snapshots used by the API.
UPDATE sales_orders o SET total = (SELECT sum(qty::bigint * unit_price) FROM sales_order_items WHERE order_id = o.id);
INSERT INTO stock_movements (product_id,type,qty,note,order_id,created_at)
SELECT p.id,'out',i.qty,'Sample sales order #' || i.order_id,i.order_id,o.created_at
FROM sales_order_items i JOIN products p ON p.sku=i.sku JOIN sales_orders o ON o.id=i.order_id;
UPDATE products p SET stock = p.stock - s.qty
FROM (SELECT sku,sum(qty)::integer qty FROM sales_order_items GROUP BY sku) s WHERE p.sku=s.sku;

-- +goose Down
SELECT 1;
