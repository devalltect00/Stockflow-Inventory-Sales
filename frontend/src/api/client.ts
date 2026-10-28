/** The sole network boundary: endpoint construction, JSON parsing, cancellation, and errors. */

import type {
  ErrorDetail,
  InventorySummary,
  Movement,
  Order,
  OrderInput,
  Page,
  Product,
  ProductFilter,
  ProductInput,
  StockInput,
} from './types'

export class ApiError extends Error {
  readonly status: number
  readonly detail: ErrorDetail

  constructor(status: number, detail: ErrorDetail) {
    super(detail.message)
    this.name = 'ApiError'
    this.status = status
    this.detail = detail
  }
}

function isErrorEnvelope(value: unknown): value is { error: ErrorDetail } {
  if (!value || typeof value !== 'object' || !('error' in value)) return false
  const detail = value.error
  return (
    !!detail &&
    typeof detail === 'object' &&
    'code' in detail &&
    typeof detail.code === 'string' &&
    'message' in detail &&
    typeof detail.message === 'string'
  )
}

/** Execute one relative API call; Vite/Nginx own proxying so components know no hosts. */
async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  let response: Response
  try {
    response = await fetch(`/api${path}`, {
      ...options,
      headers: {
        ...(options.body ? { 'Content-Type': 'application/json' } : {}),
        ...options.headers,
      },
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') throw error
    throw new ApiError(0, {
      code: 'NETWORK_ERROR',
      message: 'Cannot reach StockFlow. Check the connection and try again.',
    })
  }
  if (response.status === 204) return undefined as T
  let body: unknown
  try {
    body = await response.json()
  } catch {
    throw new ApiError(response.status, {
      code: 'INVALID_RESPONSE',
      message: 'The server returned an unreadable response. Please retry.',
    })
  }
  if (!response.ok) {
    if (isErrorEnvelope(body)) throw new ApiError(response.status, body.error)
    throw new ApiError(response.status, {
      code: 'HTTP_ERROR',
      message: 'The request failed. Please retry.',
    })
  }
  return body as T
}

/** Encode pagination in one place; feature components never assemble endpoint URLs. */
function pageQuery(page: number, limit: number): URLSearchParams {
  return new URLSearchParams({ page: String(page), limit: String(limit) })
}

/** Explicit endpoint methods are the only public network surface used by feature hooks. */
export const api = {
  /** Load every active catalog page so the order picker never silently omits products after page 100. */
  async productOptions(signal?: AbortSignal): Promise<Product[]> {
    const products: Product[] = []
    let page = 1
    let result: Page<Product>
    do {
      result = await api.products('', 'active', page++, 100, signal)
      products.push(...result.data)
    } while (result.page * result.limit < result.total)
    return products
  },

  products(search = '', filter: ProductFilter = 'all', page = 1, limit = 12, signal?: AbortSignal) {
    const query = pageQuery(page, limit)
    query.set('search', search)
    query.set('filter', filter)
    return request<Page<Product>>(`/products?${query}`, { signal })
  },

  summary: (signal?: AbortSignal) => request<InventorySummary>('/products/summary', { signal }),

  createProduct: (input: ProductInput) =>
    request<Product>('/products', { method: 'POST', body: JSON.stringify(input) }),

  updateProduct: (id: number, input: ProductInput) =>
    request<Product>(`/products/${id}`, { method: 'PUT', body: JSON.stringify(input) }),

  deleteProduct: (id: number) => request<void>(`/products/${id}`, { method: 'DELETE' }),

  adjustStock: (id: number, input: StockInput) =>
    request<Product>(`/products/${id}/stock`, { method: 'POST', body: JSON.stringify(input) }),

  movements: (id: number, signal?: AbortSignal) =>
    request<{ data: Movement[] }>(`/products/${id}/movements`, { signal }),

  orders: (page = 1, signal?: AbortSignal) =>
    request<Page<Order>>(`/orders?${pageQuery(page, 12)}`, { signal }),

  createOrder: (input: OrderInput) =>
    request<Order>('/orders', { method: 'POST', body: JSON.stringify(input) }),

  order: (id: number, signal?: AbortSignal) => request<Order>(`/orders/${id}`, { signal }),

  cancelOrder: (id: number) => request<Order>(`/orders/${id}/cancel`, { method: 'POST' }),
}
