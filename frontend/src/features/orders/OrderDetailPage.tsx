import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ArrowLeft, Check, CircleCheck, RotateCcw } from 'lucide-react'
import { Link, useLocation, useParams } from 'react-router-dom'
import { api } from '../../api/client'
import { keys, useRefreshInventory } from '../../api/queries'
import { Dialog } from '../../components/Dialog'
import { ErrorNotice, TableSkeleton } from '../../components/Feedback'
import { dateTime, orderNumber, rupiah } from '../../lib/format'

export function OrderDetailPage() {
  const { id } = useParams()
  const orderID = Number(id)
  const validID = Number.isInteger(orderID) && orderID > 0 && orderID <= 2_147_483_647
  const location = useLocation()
  const [confirm, setConfirm] = useState(false)
  const refresh = useRefreshInventory()
  const query = useQuery({
    queryKey: [...keys.orders, 'detail', orderID],
    queryFn: ({ signal }) => api.order(orderID, signal),
    enabled: validID,
  })
  const mutation = useMutation({
    mutationFn: () => api.cancelOrder(orderID),
    onSuccess: async () => {
      await refresh()
      setConfirm(false)
    },
  })
  if (!validID) return <ErrorNotice error={new Error('This order ID is invalid.')} />
  if (query.isPending) return <TableSkeleton />
  if (query.isError) return <ErrorNotice error={query.error} retry={() => void query.refetch()} />
  const order = query.data
  return (
    <>
      <Link to="/orders" className="back-link">
        <ArrowLeft size={16} />
        All sales orders
      </Link>
      <header className="page-heading">
        <div>
          <div className="eyebrow">ORDER DETAILS</div>
          <h1>
            {orderNumber(order.id)}{' '}
            <span className={`badge ${order.status === 'CREATED' ? 'active' : 'inactive'}`}>
              <span className="status-dot" />
              {order.status === 'CREATED' ? 'Created' : 'Cancelled'}
            </span>
          </h1>
          <p>Created {dateTime(order.created_at)}</p>
        </div>
        {order.status === 'CREATED' && (
          <button className="button secondary destructive" onClick={() => setConfirm(true)}>
            <RotateCcw size={16} />
            Cancel order
          </button>
        )}
      </header>
      {order.status === 'CANCELLED' ? (
        <div className="success-notice" role="status">
          <CircleCheck size={19} />
          <span>
            <strong>Order cancelled.</strong> All items have been restored to inventory.{' '}
            <Link to="/products">View inventory →</Link>
          </span>
        </div>
      ) : (
        location.state?.created && (
          <div className="success-notice" role="status">
            <Check size={19} />
            <span>
              <strong>Order created successfully.</strong> Stock has been updated for every item.
            </span>
          </div>
        )
      )}
      <div className="detail-meta">
        <div>
          <span>Customer</span>
          <strong>{order.customer}</strong>
        </div>
        <div>
          <span>Order reference</span>
          <strong>{orderNumber(order.id)}</strong>
        </div>
        <div>
          <span>{order.cancelled_at ? 'Cancelled on' : 'Items'}</span>
          <strong>
            {order.cancelled_at
              ? dateTime(order.cancelled_at)
              : `${order.items.length} product ${order.items.length === 1 ? 'line' : 'lines'}`}
          </strong>
        </div>
      </div>
      <section className="panel">
        <header className="panel-heading">
          <div>
            <h2>Order items</h2>
            <p>Prices recorded at the time of sale.</p>
          </div>
        </header>
        <div className="table-scroll">
          <table className="data-table">
            <caption className="sr-only">Items in {orderNumber(order.id)}</caption>
            <thead>
              <tr>
                <th scope="col">Product SKU</th>
                <th scope="col">Unit price</th>
                <th scope="col">Quantity</th>
                <th scope="col" className="align-right">
                  Line total
                </th>
              </tr>
            </thead>
            <tbody>
              {order.items.map((item) => (
                <tr key={item.id}>
                  <td>
                    <strong className="sku">{item.sku}</strong>
                  </td>
                  <td className="numeric">{rupiah(item.unit_price)}</td>
                  <td>{item.qty} units</td>
                  <td className="numeric align-right">{rupiah(item.qty * item.unit_price)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <footer className="order-total">
          <div>
            <span>Order total</span>
            <small>Calculated and saved by the server</small>
          </div>
          <strong>{rupiah(order.total)}</strong>
        </footer>
      </section>
      {confirm && (
        <Dialog
          title="Cancel this order?"
          description={orderNumber(order.id)}
          onClose={() => setConfirm(false)}
          busy={mutation.isPending}
        >
          <div className="dialog-body form-stack">
            <p>
              This will restore every item to inventory. The original order and its movement history
              will be preserved.
            </p>
            {mutation.isError && <ErrorNotice error={mutation.error} />}
            <footer className="dialog-actions">
              <button
                className="button secondary"
                disabled={mutation.isPending}
                onClick={() => setConfirm(false)}
              >
                Keep order
              </button>
              <button
                className="button danger"
                disabled={mutation.isPending}
                onClick={() => mutation.mutate()}
              >
                {mutation.isPending ? 'Restoring stock…' : 'Confirm cancellation'}
              </button>
            </footer>
          </div>
        </Dialog>
      )}
    </>
  )
}
