import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowDownUp,
  Boxes,
  Check,
  CircleCheck,
  History,
  Package,
  Pencil,
  Plus,
  Search,
  Trash2,
  TriangleAlert,
  Wallet,
  X,
} from 'lucide-react'
import { api } from '../../api/client'
import { keys } from '../../api/queries'
import type { Product, ProductFilter } from '../../api/types'
import { EmptyState, ErrorNotice, TableSkeleton } from '../../components/Feedback'
import { Pagination } from '../../components/Pagination'
import { useDebounce } from '../../hooks/useDebounce'
import { dateTime, relativeTime, rupiah } from '../../lib/format'
import { DeleteProductDialog, HistoryDialog, ProductDialog, StockDialog } from './ProductDialogs'

/** A discriminated union prevents multiple conflicting product dialogs from opening together. */
type ActiveDialog =
  { type: 'create' } | { type: 'edit' | 'stock' | 'history' | 'delete'; product: Product } | null

/** Filter labels are presentation; their values match the backend's documented extension. */
const filters: { value: ProductFilter; label: string }[] = [
  { value: 'all', label: 'All products' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'low-stock', label: 'Low stock' },
]

/** Summary cards always query the complete inventory rather than the currently visible table page. */
function SummaryCards() {
  const query = useQuery({ queryKey: keys.summary, queryFn: ({ signal }) => api.summary(signal) })
  if (query.isError) return <ErrorNotice error={query.error} retry={() => void query.refetch()} />
  const cards = [
    {
      title: 'Total products',
      value: query.data?.total_products,
      description: 'Across your catalog',
      icon: Package,
      tone: 'teal',
    },
    {
      title: 'Active products',
      value: query.data?.active_products,
      description: 'Available for sale',
      icon: CircleCheck,
      tone: 'blue',
    },
    {
      title: 'Inventory value',
      value: query.data ? rupiah(BigInt(query.data.total_stock_value)) : undefined,
      description: 'Current stock × unit price',
      icon: Wallet,
      tone: 'violet',
    },
    {
      title: 'Low-stock products',
      value: query.data?.low_stock_count,
      description: '10 units or fewer',
      icon: TriangleAlert,
      tone: 'amber',
    },
  ]
  return (
    <section className="summary-grid" aria-label="Inventory summary">
      {cards.map((card) => (
        <article className="summary-card" key={card.title}>
          <div className="summary-top">
            <span>{card.title}</span>
            <span className={`card-icon ${card.tone}`}>
              <card.icon size={18} />
            </span>
          </div>
          {card.value === undefined ? (
            <span className="skeleton summary-skeleton" />
          ) : (
            <strong className={card.title === 'Inventory value' ? 'money-value' : ''}>
              {card.value}
            </strong>
          )}
          <small>{card.description}</small>
        </article>
      ))}
    </section>
  )
}

/** ProductsPage owns view state; all persisted state comes from the API and query cache. */
export function ProductsPage() {
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search)
  const [filter, setFilter] = useState<ProductFilter>('all')
  const [page, setPage] = useState(1)
  const [dialog, setDialog] = useState<ActiveDialog>(null)
  const [notice, setNotice] = useState('')
  const query = useQuery({
    queryKey: [...keys.products, debouncedSearch, filter, page],
    queryFn: ({ signal }) => api.products(debouncedSearch, filter, page, 12, signal),
  })

  /** Closing a successful mutation keeps its confirmation visible after the modal unmounts. */
  function completed(message: string) {
    setDialog(null)
    setNotice(message)
  }

  return (
    <>
      <header className="page-heading">
        <div>
          <div className="eyebrow">YOUR WORKSPACE, IN SYNC</div>
          <h1>Inventory</h1>
          <p>A clear view of your products. Every unit accounted for.</p>
        </div>
        <button className="button primary" onClick={() => setDialog({ type: 'create' })}>
          <Plus size={18} />
          Add product
        </button>
      </header>
      <SummaryCards />
      {notice && (
        <div className="success-notice" role="status">
          <Check size={18} />
          <span>{notice}</span>
          <button
            className="icon-button"
            aria-label="Dismiss notification"
            onClick={() => setNotice('')}
          >
            <X size={16} />
          </button>
        </div>
      )}
      <section className="panel catalog-panel" aria-labelledby="catalog-title">
        <header className="panel-heading">
          <div>
            <h2 id="catalog-title">
              Product catalog <span className="count-tag">{query.data?.total ?? '—'}</span>
            </h2>
            <p>Manage your stock, pricing, and product availability.</p>
          </div>
          <span className="subtle-label">
            <Boxes size={15} /> Live inventory
          </span>
        </header>
        <div className="catalog-toolbar">
          <div className="filter-tabs" role="group" aria-label="Filter products">
            {filters.map((item) => (
              <button
                key={item.value}
                className={filter === item.value ? 'selected' : ''}
                aria-pressed={filter === item.value}
                onClick={() => {
                  setFilter(item.value)
                  setPage(1)
                }}
              >
                {item.label}
              </button>
            ))}
          </div>
          <label className="search-field">
            <Search size={17} />
            <span className="sr-only">Search products</span>
            <input
              value={search}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
              placeholder="Search name or SKU…"
            />
            {search && (
              <button
                className="icon-button"
                aria-label="Clear search"
                onClick={() => {
                  setSearch('')
                  setPage(1)
                }}
              >
                <X size={14} />
              </button>
            )}
          </label>
        </div>
        {query.isPending ? (
          <TableSkeleton columns={7} />
        ) : query.isError ? (
          <div className="panel-feedback">
            <ErrorNotice error={query.error} retry={() => void query.refetch()} />
          </div>
        ) : !query.data.data.length ? (
          <EmptyState
            title={search || filter !== 'all' ? 'No matching products' : 'Your catalog starts here'}
            description={
              search || filter !== 'all'
                ? 'Try a different search or filter to find what you need.'
                : 'Add your first product, then receive stock into your warehouse.'
            }
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <caption className="sr-only">Products and current warehouse stock</caption>
              <thead>
                <tr>
                  <th scope="col">Product / SKU</th>
                  <th scope="col">Unit price</th>
                  <th scope="col">Stock</th>
                  <th scope="col">Status</th>
                  <th scope="col">Last movement</th>
                  <th scope="col" className="actions-heading">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                {query.data.data.map((product, index) => (
                  <tr key={product.id}>
                    <td>
                      <div className="product-cell">
                        <span className={`product-icon tone-${index % 4}`}>
                          <Package size={20} />
                        </span>
                        <div>
                          <strong>{product.name}</strong>
                          <span className="sku">{product.sku}</span>
                        </div>
                      </div>
                    </td>
                    <td className="numeric price-cell">{rupiah(product.price)}</td>
                    <td>
                      <span className={`stock-value ${product.stock <= 10 ? 'low' : ''}`}>
                        {product.stock <= 10 && <span className="status-dot" />}
                        {product.stock}
                        <span className="stock-units"> units</span>
                      </span>
                    </td>
                    <td>
                      <span className={`badge ${product.is_active ? 'active' : 'inactive'}`}>
                        <span className="status-dot" />
                        {product.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="movement-time">
                      <time
                        dateTime={product.last_movement_at ?? undefined}
                        title={
                          product.last_movement_at ? dateTime(product.last_movement_at) : undefined
                        }
                      >
                        {relativeTime(product.last_movement_at)}
                      </time>
                    </td>
                    <td>
                      <div className="row-actions">
                        <button
                          className="icon-button"
                          title="Adjust stock"
                          aria-label={`Adjust stock for ${product.sku}`}
                          onClick={() => setDialog({ type: 'stock', product })}
                        >
                          <ArrowDownUp size={16} />
                        </button>
                        <button
                          className="icon-button"
                          title="Movement history"
                          aria-label={`View history for ${product.sku}`}
                          onClick={() => setDialog({ type: 'history', product })}
                        >
                          <History size={16} />
                        </button>
                        <button
                          className="icon-button"
                          title="Edit product"
                          aria-label={`Edit ${product.sku}`}
                          onClick={() => setDialog({ type: 'edit', product })}
                        >
                          <Pencil size={15} />
                        </button>
                        <button
                          className="icon-button destructive"
                          title="Delete product"
                          aria-label={`Delete ${product.sku}`}
                          onClick={() => setDialog({ type: 'delete', product })}
                        >
                          <Trash2 size={15} />
                        </button>
                      </div>
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
      <p className="workspace-note">
        <span className="status-dot" />
        Inventory updates are saved with a permanent movement record.
      </p>
      {dialog?.type === 'create' && (
        <ProductDialog onClose={() => setDialog(null)} onSuccess={completed} />
      )}
      {dialog?.type === 'edit' && (
        <ProductDialog
          product={dialog.product}
          onClose={() => setDialog(null)}
          onSuccess={completed}
        />
      )}
      {dialog?.type === 'stock' && (
        <StockDialog
          product={dialog.product}
          onClose={() => setDialog(null)}
          onSuccess={completed}
        />
      )}
      {dialog?.type === 'history' && (
        <HistoryDialog product={dialog.product} onClose={() => setDialog(null)} />
      )}
      {dialog?.type === 'delete' && (
        <DeleteProductDialog
          product={dialog.product}
          onClose={() => setDialog(null)}
          onSuccess={completed}
        />
      )}
    </>
  )
}
