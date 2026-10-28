/** Public API contracts. Domain data uses these shared types throughout the frontend. */

export interface Product {
  id: number
  sku: string
  name: string
  price: number
  stock: number
  is_active: boolean
  created_at: string
  last_movement_at: string | null
}

export interface Page<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export interface InventorySummary {
  total_products: number
  active_products: number
  /** Decimal integer text avoids rounding large aggregate money in JavaScript. */
  total_stock_value: string
  low_stock_count: number
}

export interface ProductInput {
  sku: string
  name: string
  price: number
  is_active: boolean
}

export interface StockInput {
  type: 'in' | 'out'
  qty: number
  note: string
}

export interface Movement {
  id: number
  product_id: number
  type: 'in' | 'out'
  qty: number
  note: string | null
  order_id: number | null
  created_at: string
}

export interface OrderInput {
  customer: string
  items: { sku: string; qty: number }[]
}

export interface OrderItem {
  id: number
  sku: string
  qty: number
  unit_price: number
}

export interface Order {
  id: number
  customer: string
  status: 'CREATED' | 'CANCELLED'
  total: number
  created_at: string
  cancelled_at: string | null
  items: OrderItem[]
}

export type ProductFilter = 'all' | 'active' | 'inactive' | 'low-stock'

export interface ErrorDetail {
  code: string
  message: string
  fields?: Record<string, string>
  sku?: string
  available?: number
}
