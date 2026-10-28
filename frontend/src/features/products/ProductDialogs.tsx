import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ArrowDownLeft, ArrowUpRight, LoaderCircle } from 'lucide-react'
import { Link } from 'react-router-dom'
import { api } from '../../api/client'
import { keys, useRefreshInventory } from '../../api/queries'
import type { Product, ProductInput, StockInput } from '../../api/types'
import { Dialog } from '../../components/Dialog'
import { EmptyState, ErrorNotice, FieldError, TableSkeleton } from '../../components/Feedback'
import { dateTime, orderNumber } from '../../lib/format'
import { fieldErrors, productSchema, stockSchema } from '../../lib/validation'

/** Shared completion callbacks report a successful mutation to the parent products page. */
interface DialogProps {
  onClose: () => void
  onSuccess: (message: string) => void
}

/** Create or edit catalog metadata; opening stock is intentionally a separate adjustment. */
export function ProductDialog({
  product,
  onClose,
  onSuccess,
}: DialogProps & { product?: Product }) {
  const [form, setForm] = useState({
    sku: product?.sku ?? '',
    name: product?.name ?? '',
    price: String(product?.price ?? ''),
    is_active: product?.is_active ?? true,
  })
  const [fields, setFields] = useState<Record<string, string>>({})
  const refresh = useRefreshInventory()
  const mutation = useMutation({
    mutationFn: (input: ProductInput) =>
      product ? api.updateProduct(product.id, input) : api.createProduct(input),
    onSuccess: async () => {
      await refresh()
      onSuccess(
        product ? 'Product updated.' : 'Product created. Add its opening stock with Adjust stock.',
      )
    },
  })

  /** Validate all inputs locally; the API repeats the same checks before persistence. */
  function submit(event: FormEvent) {
    event.preventDefault()
    const result = productSchema.safeParse(form)
    if (!result.success) {
      setFields(fieldErrors(result.error))
      return
    }
    setFields({})
    mutation.mutate(result.data)
  }

  return (
    <Dialog
      title={product ? 'Edit product' : 'Add a product'}
      description={
        product
          ? 'Update product details. SKU and inventory history stay unchanged.'
          : 'Create a catalog entry, then receive stock when it arrives.'
      }
      busy={mutation.isPending}
      onClose={onClose}
    >
      <form onSubmit={submit} noValidate className="dialog-body form-stack">
        {mutation.isError && <ErrorNotice error={mutation.error} />}
        <label>
          SKU
          <input
            autoFocus={!product}
            value={form.sku}
            onChange={(e) => setForm({ ...form, sku: e.target.value })}
            disabled={!!product || mutation.isPending}
            placeholder="e.g. KB-002"
            maxLength={64}
            aria-invalid={!!fields.sku}
          />
          <FieldError message={fields.sku} />
        </label>
        <label>
          Product name
          <input
            autoFocus={!!product}
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            disabled={mutation.isPending}
            placeholder="e.g. Wireless keyboard"
            maxLength={160}
            aria-invalid={!!fields.name}
          />
          <FieldError message={fields.name} />
        </label>
        <label>
          Unit price <span className="label-hint">IDR</span>
          <input
            value={form.price}
            onChange={(e) => setForm({ ...form, price: e.target.value })}
            disabled={mutation.isPending}
            inputMode="numeric"
            placeholder="0"
            aria-invalid={!!fields.price}
          />
          <FieldError message={fields.price} />
        </label>
        <label className="checkbox-field">
          <input
            type="checkbox"
            checked={form.is_active}
            onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
            disabled={mutation.isPending}
          />
          <span>
            Active product<small>Available for new sales orders</small>
          </span>
        </label>
        <footer className="dialog-actions">
          <button
            type="button"
            className="button secondary"
            disabled={mutation.isPending}
            onClick={onClose}
          >
            Cancel
          </button>
          <button className="button primary" disabled={mutation.isPending}>
            {mutation.isPending && <LoaderCircle className="spin" size={16} />}
            {mutation.isPending ? 'Saving…' : product ? 'Save changes' : 'Create product'}
          </button>
        </footer>
      </form>
    </Dialog>
  )
}

/** Capture one inventory movement and show server-side overdraw details directly in the modal. */
export function StockDialog({ product, onClose, onSuccess }: DialogProps & { product: Product }) {
  const [form, setForm] = useState<{ type: 'in' | 'out'; qty: string; note: string }>({
    type: 'in',
    qty: '',
    note: '',
  })
  const [fields, setFields] = useState<Record<string, string>>({})
  const refresh = useRefreshInventory()
  const mutation = useMutation({
    mutationFn: (input: StockInput) => api.adjustStock(product.id, input),
    onSuccess: async () => {
      await refresh()
      onSuccess(`Stock updated for ${product.sku}.`)
    },
  })

  /** Reject fractional or nonpositive input while keeping the backend authoritative about availability. */
  function submit(event: FormEvent) {
    event.preventDefault()
    const result = stockSchema.safeParse(form)
    if (!result.success) {
      setFields(fieldErrors(result.error))
      return
    }
    setFields({})
    mutation.mutate(result.data)
  }

  return (
    <Dialog
      title="Adjust stock"
      description={`${product.sku} · ${product.name}`}
      busy={mutation.isPending}
      onClose={onClose}
    >
      <form onSubmit={submit} noValidate className="dialog-body form-stack">
        <div className="balance-preview">
          <span>Current stock</span>
          <strong>
            {product.stock}
            <small> units</small>
          </strong>
        </div>
        {mutation.isError && <ErrorNotice error={mutation.error} />}
        <fieldset className="direction-picker" disabled={mutation.isPending}>
          <legend>Direction</legend>
          <label className={form.type === 'in' ? 'selected' : ''}>
            <input
              type="radio"
              name="direction"
              value="in"
              checked={form.type === 'in'}
              onChange={() => setForm({ ...form, type: 'in' })}
            />
            <ArrowDownLeft size={20} />
            <span>
              Stock in<small>Receive inventory</small>
            </span>
          </label>
          <label className={form.type === 'out' ? 'selected' : ''}>
            <input
              type="radio"
              name="direction"
              value="out"
              checked={form.type === 'out'}
              onChange={() => setForm({ ...form, type: 'out' })}
            />
            <ArrowUpRight size={20} />
            <span>
              Stock out<small>Remove inventory</small>
            </span>
          </label>
        </fieldset>
        <label>
          Quantity
          <input
            autoFocus
            inputMode="numeric"
            value={form.qty}
            onChange={(e) => setForm({ ...form, qty: e.target.value })}
            disabled={mutation.isPending}
            placeholder="Enter units"
            aria-invalid={!!fields.qty}
          />
          <FieldError message={fields.qty} />
        </label>
        <label>
          Note <span className="label-hint">Optional</span>
          <textarea
            value={form.note}
            onChange={(e) => setForm({ ...form, note: e.target.value })}
            disabled={mutation.isPending}
            maxLength={500}
            rows={3}
            placeholder="e.g. Received from supplier"
          />
          <FieldError message={fields.note} />
        </label>
        <p className="form-hint">
          This adjustment is saved permanently in the product’s movement history.
        </p>
        <footer className="dialog-actions">
          <button
            type="button"
            className="button secondary"
            disabled={mutation.isPending}
            onClick={onClose}
          >
            Cancel
          </button>
          <button className="button primary" disabled={mutation.isPending}>
            {mutation.isPending && <LoaderCircle className="spin" size={16} />}
            {mutation.isPending ? 'Saving…' : 'Save adjustment'}
          </button>
        </footer>
      </form>
    </Dialog>
  )
}

/** Read ledger history from the backend, including order and cancellation provenance. */
export function HistoryDialog({ product, onClose }: { product: Product; onClose: () => void }) {
  const query = useQuery({
    queryKey: [...keys.movements, product.id],
    queryFn: ({ signal }) => api.movements(product.id, signal),
  })
  return (
    <Dialog
      wide
      title="Movement history"
      description={`${product.sku} · ${product.name}`}
      onClose={onClose}
    >
      <div className="dialog-body">
        {query.isPending ? (
          <TableSkeleton columns={4} />
        ) : query.isError ? (
          <ErrorNotice error={query.error} retry={() => void query.refetch()} />
        ) : !query.data.data.length ? (
          <EmptyState
            title="No movements yet"
            description="The first stock adjustment will appear here."
          />
        ) : (
          <div className="movement-list">
            {query.data.data.map((m) => (
              <article className="movement-row" key={m.id}>
                <span className={`movement-icon ${m.type}`}>
                  {m.type === 'in' ? <ArrowDownLeft size={19} /> : <ArrowUpRight size={19} />}
                </span>
                <div>
                  <strong>{m.type === 'in' ? 'Stock received' : 'Stock removed'}</strong>
                  <p>
                    {m.note || 'Manual adjustment'}
                    {m.order_id && (
                      <>
                        {' '}
                        ·{' '}
                        <Link to={`/orders/${m.order_id}`} onClick={onClose}>
                          {orderNumber(m.order_id)}
                        </Link>
                      </>
                    )}
                  </p>
                  <time dateTime={m.created_at}>{dateTime(m.created_at)}</time>
                </div>
                <b className={m.type === 'in' ? 'positive' : ''}>
                  {m.type === 'in' ? '+' : '−'}
                  {m.qty}
                </b>
              </article>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}

/** Delete only an unused catalog entry; the API's history safeguard remains visible on failure. */
export function DeleteProductDialog({
  product,
  onClose,
  onSuccess,
}: DialogProps & { product: Product }) {
  const refresh = useRefreshInventory()
  const mutation = useMutation({
    mutationFn: () => api.deleteProduct(product.id),
    onSuccess: async () => {
      await refresh()
      onSuccess(`${product.sku} deleted.`)
    },
  })
  return (
    <Dialog
      title="Delete product?"
      description={`${product.sku} · ${product.name}`}
      busy={mutation.isPending}
      onClose={onClose}
    >
      <div className="dialog-body form-stack">
        <p>
          A product with stock movements or sales history cannot be deleted. You can deactivate it
          instead.
        </p>
        {mutation.isError && <ErrorNotice error={mutation.error} />}
        <footer className="dialog-actions">
          <button className="button secondary" disabled={mutation.isPending} onClick={onClose}>
            Keep product
          </button>
          <button
            className="button danger"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? 'Deleting…' : 'Delete product'}
          </button>
        </footer>
      </div>
    </Dialog>
  )
}
