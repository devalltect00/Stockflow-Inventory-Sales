import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowUpRight, Plus, ShoppingBag } from 'lucide-react'
import { Link } from 'react-router-dom'
import { api } from '../../api/client'
import { keys } from '../../api/queries'
import { EmptyState, ErrorNotice, TableSkeleton } from '../../components/Feedback'
import { Pagination } from '../../components/Pagination'
import { dateTime, orderNumber, rupiah } from '../../lib/format'

export function OrdersPage() {
  const [page, setPage] = useState(1)
  const query = useQuery({
    queryKey: [...keys.orders, 'list', page],
    queryFn: ({ signal }) => api.orders(page, signal),
  })
  return (
    <>
      <header className="page-heading">
        <div>
          <div className="eyebrow">FROM SHELF TO CUSTOMER</div>
          <h1>Sales orders</h1>
          <p>Keep every sale connected to your inventory.</p>
        </div>
        <Link className="button primary" to="/orders/new">
          <Plus size={18} />
          Create order
        </Link>
      </header>
      <div className="info-strip">
        <span className="card-icon teal">
          <ShoppingBag size={20} />
        </span>
        <div>
          <strong>Sales, with stock in sync.</strong>
          <p>Creating an order reserves its stock immediately. Cancelling restores every item.</p>
        </div>
      </div>
      <section className="panel">
        <header className="panel-heading">
          <div>
            <h2>
              Order history <span className="count-tag">{query.data?.total ?? '—'}</span>
            </h2>
            <p>All sales orders, newest first.</p>
          </div>
        </header>
        {query.isPending ? (
          <TableSkeleton />
        ) : query.isError ? (
          <div className="panel-feedback">
            <ErrorNotice error={query.error} retry={() => void query.refetch()} />
          </div>
        ) : !query.data.data.length ? (
          <EmptyState
            title="Ready for your first sale"
            description="Create an order to record a sale and update your inventory together."
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <caption className="sr-only">Sales orders</caption>
              <thead>
                <tr>
                  <th scope="col">Order</th>
                  <th scope="col">Customer</th>
                  <th scope="col">Date created</th>
                  <th scope="col">Status</th>
                  <th scope="col">Total</th>
                  <th scope="col">
                    <span className="sr-only">Open order</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {query.data.data.map((order) => (
                  <tr key={order.id}>
                    <td>
                      <Link className="order-link" to={`/orders/${order.id}`}>
                        {orderNumber(order.id)}
                      </Link>
                    </td>
                    <td>
                      <strong>{order.customer}</strong>
                    </td>
                    <td className="muted">
                      <time dateTime={order.created_at}>{dateTime(order.created_at)}</time>
                    </td>
                    <td>
                      <span
                        className={`badge ${order.status === 'CREATED' ? 'active' : 'inactive'}`}
                      >
                        <span className="status-dot" />
                        {order.status === 'CREATED' ? 'Created' : 'Cancelled'}
                      </span>
                    </td>
                    <td className="numeric">{rupiah(order.total)}</td>
                    <td>
                      <Link
                        className="icon-button"
                        aria-label={`View ${orderNumber(order.id)}`}
                        to={`/orders/${order.id}`}
                      >
                        <ArrowUpRight size={17} />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {query.data && (
          <Pagination
            page={page}
            limit={query.data.limit}
            total={query.data.total}
            onPage={setPage}
          />
        )}
      </section>
    </>
  )
}
