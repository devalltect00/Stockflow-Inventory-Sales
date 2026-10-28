import { expect, it } from 'vitest'
import { orderNumber, relativeTime, rupiah } from './format'

it('formats IDR as whole rupiah', () => {
  expect(rupiah(1234567).replace(/\s/g, ' ')).toBe('Rp 1.234.567')
})
it('handles null and fixed-clock movement timestamps', () => {
  const now = Date.parse('2026-10-28T03:00:00Z')
  expect(relativeTime(null, now)).toBe('No movements')
  expect(relativeTime('2026-10-28T01:00:00Z', now)).toBe('2 hours ago')
  expect(relativeTime('2026-10-28T02:59:50Z', now)).toBe('Just now')
})
it('formats a stable order reference', () => {
  expect(orderNumber(12)).toBe('SO-0012')
})
