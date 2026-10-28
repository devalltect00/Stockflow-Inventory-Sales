/** Central cache keys and invalidation preserve consistency after every inventory mutation. */

import { useQueryClient } from '@tanstack/react-query'

export const keys = {
  products: ['products'],
  summary: ['summary'],
  movements: ['movements'],
  orders: ['orders'],
} as const

export function useRefreshInventory() {
  const client = useQueryClient()
  return () =>
    Promise.all([
      client.invalidateQueries({ queryKey: keys.products }),
      client.invalidateQueries({ queryKey: keys.summary }),
      client.invalidateQueries({ queryKey: keys.movements }),
      client.invalidateQueries({ queryKey: keys.orders }),
    ])
}
