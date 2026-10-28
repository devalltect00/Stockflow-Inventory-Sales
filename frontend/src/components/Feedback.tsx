/** Shared accessible feedback for loading, empty results, and recoverable backend failures. */
import { AlertCircle, Inbox, RotateCcw } from 'lucide-react'
import { ApiError } from '../api/client'

/** Render server details, including a zero available quantity, in the visible UI. */
export function ErrorNotice({ error, retry }: { error: unknown; retry?: () => void }) {
  const detail = error instanceof ApiError ? error.detail : undefined
  const message = error instanceof Error ? error.message : 'Something went wrong. Please try again.'
  return (
    <div className="error-notice" role="alert">
      <AlertCircle size={19} aria-hidden="true" />
      <div>
        <strong>
          {detail?.code === 'INSUFFICIENT_STOCK'
            ? `Not enough stock${detail.sku ? ` for ${detail.sku}` : ''}`
            : 'Unable to complete the request'}
        </strong>
        <p>
          {message}
          {detail?.available !== undefined && (
            <>
              {' '}
              Available quantity: <b>{detail.available}</b>.
            </>
          )}
        </p>
        {detail?.fields && (
          <ul>
            {Object.entries(detail.fields).map(([field, value]) => (
              <li key={field}>{value}</li>
            ))}
          </ul>
        )}
      </div>
      {retry && (
        <button type="button" className="button secondary small" onClick={retry}>
          <RotateCcw size={14} />
          Retry
        </button>
      )}
    </div>
  )
}

/** Empty states distinguish a valid empty result from loading and network failure. */
export function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="empty-state">
      <span className="empty-icon">
        <Inbox size={28} />
      </span>
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  )
}

/** Skeleton rows retain table proportions and expose an accessible loading status. */
export function TableSkeleton({ columns = 6 }: { columns?: number }) {
  return (
    <div role="status" aria-label="Loading records" className="table-skeleton">
      {Array.from({ length: 6 }, (_, row) => (
        <div className="skeleton-row" key={row}>
          {Array.from({ length: columns }, (_, col) => (
            <span className="skeleton" key={col} />
          ))}
        </div>
      ))}
    </div>
  )
}

/** Keep field errors adjacent to the exact input that needs attention. */
export function FieldError({ message, id }: { message?: string; id?: string }) {
  return message ? (
    <span id={id} className="field-error">
      {message}
    </span>
  ) : null
}
