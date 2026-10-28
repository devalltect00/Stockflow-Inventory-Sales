/** Deterministic display helpers keep money and dates consistent across screens. */
const rupiahFormatter = new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
})
const dateFormatter = new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' })
const relativeFormatter = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })

/** Format integer IDR without introducing fractional money. */
export function rupiah(value: number | bigint): string {
  return rupiahFormatter.format(value)
}

/** Show a timestamp in the viewer's local timezone. */
export function dateTime(value: string): string {
  return dateFormatter.format(new Date(value))
}

/** Derive readable movement age; an injectable clock makes the rule testable. */
export function relativeTime(value: string | null, now = Date.now()): string {
  if (!value) return 'No movements'
  const seconds = Math.max(0, Math.floor((now - Date.parse(value)) / 1000))
  if (seconds < 60) return 'Just now'
  if (seconds < 3600) return relativeFormatter.format(-Math.floor(seconds / 60), 'minute')
  if (seconds < 86400) return relativeFormatter.format(-Math.floor(seconds / 3600), 'hour')
  return relativeFormatter.format(-Math.floor(seconds / 86400), 'day')
}

/** A consistent order label avoids implying that numeric IDs are secret credentials. */
export function orderNumber(id: number): string {
  return `SO-${String(id).padStart(4, '0')}`
}
