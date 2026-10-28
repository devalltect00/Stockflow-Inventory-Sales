import { useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import { X } from 'lucide-react'

/** Mount one modal; pending writes prevent closing it before the result is known. */
export function Dialog({
  title,
  description,
  children,
  onClose,
  busy = false,
  wide = false,
}: {
  title: string
  description?: string
  children: ReactNode
  onClose: () => void
  busy?: boolean
  wide?: boolean
}) {
  const ref = useRef<HTMLDialogElement>(null)
  const titleID = useId()
  const descriptionID = useId()
  useEffect(() => {
    const node = ref.current
    node?.showModal()
    return () => node?.close()
  }, [])
  return (
    <dialog
      ref={ref}
      className={`dialog ${wide ? 'wide' : ''}`}
      aria-labelledby={titleID}
      aria-describedby={description ? descriptionID : undefined}
      onCancel={(event) => {
        event.preventDefault()
        if (!busy) onClose()
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget && !busy) {
          const rect = event.currentTarget.getBoundingClientRect()
          if (
            event.clientX < rect.left ||
            event.clientX > rect.right ||
            event.clientY < rect.top ||
            event.clientY > rect.bottom
          )
            onClose()
        }
      }}
    >
      <header className="dialog-header">
        <div>
          <h2 id={titleID}>{title}</h2>
          {description && <p id={descriptionID}>{description}</p>}
        </div>
        <button className="icon-button" aria-label="Close dialog" disabled={busy} onClick={onClose}>
          <X size={20} />
        </button>
      </header>
      {children}
    </dialog>
  )
}
