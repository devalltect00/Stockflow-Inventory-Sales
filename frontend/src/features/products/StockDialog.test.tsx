import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'
import type { Product } from '../../api/types'
import { StockDialog } from './ProductDialogs'

/** The fixture is typed so contract changes also break this critical interaction test. */
const product: Product = {
  id: 1,
  sku: 'KB-001',
  name: 'Keyboard',
  price: 100,
  stock: 5,
  is_active: true,
  created_at: '2026-10-28T00:00:00Z',
  last_movement_at: null,
}

it('displays available quantity when the backend rejects stock out', async () => {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: 'INSUFFICIENT_STOCK',
            message: 'Insufficient stock for KB-001.',
            sku: 'KB-001',
            available: 2,
          },
        }),
        { status: 409 },
      ),
    ),
  )
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  const user = userEvent.setup()
  render(
    <QueryClientProvider client={client}>
      <StockDialog product={product} onClose={vi.fn()} onSuccess={vi.fn()} />
    </QueryClientProvider>,
  )
  await user.click(screen.getByRole('radio', { name: /Stock out/ }))
  await user.type(screen.getByLabelText('Quantity'), '10')
  await user.click(screen.getByRole('button', { name: 'Save adjustment' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Available quantity: 2')
  expect(screen.getByRole('dialog')).toBeVisible()
})
