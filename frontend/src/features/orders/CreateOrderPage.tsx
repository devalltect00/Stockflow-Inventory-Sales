/** The order form submits identities and quantities, then navigates to the committed server result. */
import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ArrowLeft, ArrowRight, Info, LoaderCircle, Plus, ShoppingBag, Trash2 } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { api, ApiError } from '../../api/client'
import { keys, useRefreshInventory } from '../../api/queries'
import type { OrderInput } from '../../api/types'
import { ErrorNotice, FieldError } from '../../components/Feedback'
import { rupiah } from '../../lib/format'
import { fieldErrors, orderSchema, quantitySchema } from '../../lib/validation'

/** Draft item keys remain stable when an earlier row is removed. */
interface DraftItem {
  key: number
  sku: string
  qty: string
}

/** CreateOrderPage handles validation and SKU-specific conflicts without trusting local availability. */
export function CreateOrderPage() {
  const navigate = useNavigate()
  const refresh = useRefreshInventory()
  const [customer, setCustomer] = useState('')
  const [items, setItems] = useState<DraftItem[]>([{ key: 0, sku: '', qty: '1' }])
  const [nextKey, setNextKey] = useState(1)
  const [fields, setFields] = useState<Record<string, string>>({})
  const products = useQuery({
    queryKey: [...keys.products, 'order-options'],
    queryFn: ({ signal }) => api.productOptions(signal),
  })
  const mutation = useMutation({
    mutationFn: (input: OrderInput) => api.createOrder(input),
    onSuccess: async (order) => {
      await refresh()
      navigate(`/orders/${order.id}`, { state: { created: true } })
    },
  })
  const failed = mutation.error instanceof ApiError ? mutation.error.detail : undefined
  const estimated = items.reduce((sum, item) => {
    const quantity = quantitySchema.safeParse(item.qty)
    const product = products.data?.find((p) => p.sku === item.sku)
    return sum + (quantity.success && product ? quantity.data * product.price : 0)
  }, 0)

  /** Updating a row clears stale network feedback without discarding other user input. */
  function updateItem(key: number, field: 'sku' | 'qty', value: string) {
    setItems((current) =>
      current.map((item) => (item.key === key ? { ...item, [field]: value } : item)),
    )
    mutation.reset()
  }

  /** Validate the entire draft once; authoritative product checks happen inside the backend transaction. */
  function submit(event: FormEvent) {
    event.preventDefault()
    const result = orderSchema.safeParse({
      customer,
      items: items.map(({ sku, qty }) => ({ sku, qty })),
    })
    if (!result.success) {
      setFields(fieldErrors(result.error))
      return
    }
    setFields({})
    mutation.mutate(result.data)
  }

  return (
    <>
      <Link to="/orders" className="back-link">
        <ArrowLeft size={16} />
        All sales orders
      </Link>
      <header className="page-heading">
        <div>
          <div className="eyebrow">A NEW SALE</div>
          <h1>Create order</h1>
          <p>Choose your products. We’ll take care of the stock.</p>
        </div>
      </header>
      <form onSubmit={submit} noValidate className="order-layout">
        <div className="form-stack">
          {mutation.isError && <ErrorNotice error={mutation.error} />}
          <section className="panel">
            <header className="panel-heading">
              <div>
                <h2>
                  <span className="step-number">1</span>Customer details
                </h2>
                <p>Who is this order for?</p>
              </div>
            </header>
            <div className="panel-body">
              <label>
                Customer name
                <input
                  autoFocus
                  maxLength={160}
                  placeholder="e.g. Nusantara Creative Studio"
                  value={customer}
                  onChange={(e) => {
                    setCustomer(e.target.value)
                    mutation.reset()
                  }}
                  disabled={mutation.isPending}
                  aria-invalid={!!fields.customer}
                />
                <FieldError message={fields.customer} />
              </label>
            </div>
          </section>
          <section className="panel">
            <header className="panel-heading">
              <div>
                <h2>
                  <span className="step-number">2</span>Order items
                </h2>
                <p>Each product can appear once per order.</p>
              </div>
              <span className="count-tag">
                {items.length} {items.length === 1 ? 'item' : 'items'}
              </span>
            </header>
            <div className="panel-body order-items">
              {products.isError && (
                <ErrorNotice error={products.error} retry={() => void products.refetch()} />
              )}
              {products.isPending && (
                <p role="status" className="muted">
                  Loading available products…
                </p>
              )}
              {products.data?.length === 0 && (
                <p className="muted">
                  No active products available.{' '}
                  <Link to="/products">Add or activate a product</Link> first.
                </p>
              )}
              {items.map((item, index) => {
                const product = products.data?.find((p) => p.sku === item.sku)
                const conflict = failed?.code === 'INSUFFICIENT_STOCK' && failed.sku === item.sku
                return (
                  <div className={`order-item ${conflict ? 'has-conflict' : ''}`} key={item.key}>
                    <div className="item-fields">
                      <label>
                        Product {index + 1}
                        <select
                          aria-label={`Product ${index + 1}`}
                          value={item.sku}
                          onChange={(e) => updateItem(item.key, 'sku', e.target.value)}
                          disabled={mutation.isPending || !products.data?.length}
                          aria-invalid={!!fields[`items.${index}.sku`] || conflict}
                        >
                          <option value="">Choose a product</option>
                          {products.data?.map((p) => (
                            <option
                              key={p.id}
                              value={p.sku}
                              disabled={items.some(
                                (other) => other.key !== item.key && other.sku === p.sku,
                              )}
                            >
                              {p.sku} — {p.name} ({p.stock} available)
                            </option>
                          ))}
                        </select>
                        <FieldError message={fields[`items.${index}.sku`]} />
                      </label>
                      <label className="quantity-field">
                        Quantity
                        <input
                          aria-label={`Quantity for item ${index + 1}`}
                          inputMode="numeric"
                          value={item.qty}
                          onChange={(e) => updateItem(item.key, 'qty', e.target.value)}
                          disabled={mutation.isPending}
                          aria-invalid={!!fields[`items.${index}.qty`] || conflict}
                        />
                        <FieldError message={fields[`items.${index}.qty`]} />
                      </label>
                      <button
                        type="button"
                        className="icon-button destructive remove-item"
                        aria-label={`Remove item ${index + 1}`}
                        disabled={mutation.isPending || items.length === 1}
                        onClick={() => {
                          setItems(items.filter((other) => other.key !== item.key))
                          setFields({})
                          mutation.reset()
                        }}
                      >
                        <Trash2 size={17} />
                      </button>
                    </div>
                    {product && (
                      <div className="item-meta">
                        <span>{rupiah(product.price)} / unit</span>
                        <span>{product.stock} units currently available</span>
                      </div>
                    )}
                    {conflict && (
                      <p className="field-error" role="alert">
                        {item.sku}: only {failed.available} available. You requested {item.qty}.
                      </p>
                    )}
                  </div>
                )
              })}
              <FieldError message={fields.items} />
              <button
                type="button"
                className="button secondary add-item"
                disabled={mutation.isPending || items.length >= 100}
                onClick={() => {
                  setItems([...items, { key: nextKey, sku: '', qty: '1' }])
                  setNextKey(nextKey + 1)
                }}
              >
                <Plus size={16} />
                Add another item
              </button>
            </div>
          </section>
        </div>
        <aside className="panel order-summary">
          <span className="card-icon teal">
            <ShoppingBag size={21} />
          </span>
          <h2>Order summary</h2>
          <p>Review before creating your order.</p>
          <div className="summary-line">
            <span>Product lines</span>
            <strong>{items.filter((item) => item.sku).length}</strong>
          </div>
          <div className="estimated-total">
            <span>Estimated total</span>
            <strong>{rupiah(estimated)}</strong>
          </div>
          <p className="form-hint">
            <Info size={15} />
            Final prices and stock are checked when you create the order. The saved total comes from
            the server.
          </p>
          <button
            className="button primary full-width"
            disabled={
              mutation.isPending || products.isPending || products.isError || !products.data?.length
            }
          >
            {mutation.isPending ? (
              <LoaderCircle className="spin" size={17} />
            ) : (
              <ArrowRight size={17} />
            )}
            {mutation.isPending ? 'Creating order…' : 'Create order'}
          </button>
          <Link
            to="/orders"
            className={`cancel-link ${mutation.isPending ? 'disabled-link' : ''}`}
            aria-disabled={mutation.isPending}
            tabIndex={mutation.isPending ? -1 : undefined}
          >
            Discard draft
          </Link>
        </aside>
      </form>
    </>
  )
}
