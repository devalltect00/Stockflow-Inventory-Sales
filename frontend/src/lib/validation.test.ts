import { describe, expect, it } from 'vitest'
import { orderSchema, productSchema, quantitySchema, stockSchema } from './validation'

describe('positive integer quantities', () => {
  it.each(['', '0', '-1', '1.5', '1e3', 'NaN', '2147483648'])('rejects %j', (input) => {
    expect(quantitySchema.safeParse(input).success).toBe(false)
  })
  it('accepts and normalizes an integer', () => {
    expect(quantitySchema.parse(' 12 ')).toBe(12)
  })
})

describe('catalog metadata', () => {
  it('allows a free product and trims labels', () => {
    expect(
      productSchema.parse({ sku: ' A ', name: ' Name ', price: '0', is_active: true }),
    ).toEqual({ sku: 'A', name: 'Name', price: 0, is_active: true })
  })
  it('rejects a missing name and fractional money', () => {
    expect(
      productSchema.safeParse({ sku: 'A', name: '', price: '1.5', is_active: true }).success,
    ).toBe(false)
  })
})

describe('stock adjustment', () => {
  it('rejects invalid directions', () => {
    expect(stockSchema.safeParse({ type: 'set', qty: '1', note: '' }).success).toBe(false)
  })
})

describe('order draft', () => {
  it('rejects duplicate SKUs on the specific item row', () => {
    const result = orderSchema.safeParse({
      customer: 'Customer',
      items: [
        { sku: 'A', qty: '1' },
        { sku: ' A ', qty: '2' },
      ],
    })
    expect(result.success).toBe(false)
    if (!result.success)
      expect(result.error.issues.some((issue) => issue.path.join('.') === 'items.1.sku')).toBe(true)
  })
  it('rejects empty orders and blank customers', () => {
    expect(orderSchema.safeParse({ customer: ' ', items: [] }).success).toBe(false)
  })
  it('emits quantities without client prices', () => {
    expect(orderSchema.parse({ customer: ' Customer ', items: [{ sku: 'A', qty: '2' }] })).toEqual({
      customer: 'Customer',
      items: [{ sku: 'A', qty: 2 }],
    })
  })
})
