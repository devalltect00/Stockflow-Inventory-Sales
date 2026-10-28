/** Pagination renders server metadata consistently for products and orders. */
import { ChevronLeft, ChevronRight } from 'lucide-react'

/** Navigation is bounded by the complete server result and remains usable on an empty page. */
export function Pagination({
  page,
  limit,
  total,
  onPage,
}: {
  page: number
  limit: number
  total: number
  onPage: (page: number) => void
}) {
  const pages = Math.max(1, Math.ceil(total / limit))
  return (
    <footer className="pagination">
      <span>
        {total
          ? `${Math.min((page - 1) * limit + 1, total)}–${Math.min(page * limit, total)} of ${total}`
          : '0 results'}
      </span>
      <div>
        <button
          className="icon-button"
          aria-label="Previous page"
          disabled={page <= 1}
          onClick={() => onPage(page - 1)}
        >
          <ChevronLeft size={17} />
        </button>
        <span>
          Page {page} of {pages}
        </span>
        <button
          className="icon-button"
          aria-label="Next page"
          disabled={page >= pages}
          onClick={() => onPage(page + 1)}
        >
          <ChevronRight size={17} />
        </button>
      </div>
    </footer>
  )
}
