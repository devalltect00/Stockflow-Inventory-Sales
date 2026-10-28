/** Network-boundary tests prove that structured server conflicts survive fetch parsing. */
import { expect, it, vi } from 'vitest'
import { api, ApiError } from './client'

it('preserves a SKU conflict and zero availability', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'INSUFFICIENT_STOCK',
            message: 'Insufficient stock.',
            sku: 'A',
            available: 0,
          },
        }),
        { status: 409 },
      ),
    ),
  )
  await expect(
    api.createOrder({ customer: 'Customer', items: [{ sku: 'A', qty: 1 }] }),
  ).rejects.toMatchObject({ status: 409, detail: { sku: 'A', available: 0 } })
})
it('turns an unreachable server into a useful typed failure', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
  await expect(api.summary()).rejects.toBeInstanceOf(ApiError)
})
it('accepts successful 204 deletion without parsing nonexistent JSON', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
  await expect(api.deleteProduct(1)).resolves.toBeUndefined()
})
