/** A small reusable debounce delays server search while cancelling obsolete timers. */
import { useEffect, useState } from 'react'

/** Return the latest settled value after the user pauses typing. */
export function useDebounce<T>(value: T, delay = 300): T {
  const [settled, setSettled] = useState(value)
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(value), delay)
    return () => window.clearTimeout(timer)
  }, [value, delay])
  return settled
}
