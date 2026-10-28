import { Boxes, ChevronRight, CircleHelp, LayoutGrid, Package, ShoppingBag } from 'lucide-react'
import { Link, NavLink, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { ProductsPage } from '../features/products/ProductsPage'
import { OrdersPage } from '../features/orders/OrdersPage'
import { CreateOrderPage } from '../features/orders/CreateOrderPage'
import { OrderDetailPage } from '../features/orders/OrderDetailPage'

export function App() {
  const location = useLocation()
  const section = location.pathname.startsWith('/orders') ? 'Sales orders' : 'Inventory'
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        Skip to content
      </a>
      <aside className="sidebar">
        <Link to="/products" className="brand" aria-label="StockFlow home">
          <span className="brand-symbol">
            <Boxes size={25} />
          </span>
          <span>
            stock<span className="brand-accent">flow</span>
            <small>INVENTORY & SALES</small>
          </span>
        </Link>
        <div className="workspace-label">
          <span className="workspace-avatar">
            <LayoutGrid size={17} />
          </span>
          <div>
            <strong>Main warehouse</strong>
            <small>Your workspace</small>
          </div>
        </div>
        <div className="nav-label">OPERATIONS</div>
        <nav aria-label="Main navigation">
          <NavLink to="/products">
            <Package size={19} />
            Inventory
          </NavLink>
          <NavLink to="/orders">
            <ShoppingBag size={19} />
            Sales orders
          </NavLink>
        </nav>
        <div className="sidebar-footer">
          <span className="sidebar-note-icon">
            <CircleHelp size={19} />
          </span>
          <strong>A place for every product.</strong>
          <p>
            Keep stock moving.
            <br />
            Keep your day in flow.
          </p>
          <span className="edition-label">STOCKFLOW · MINI ERP</span>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="breadcrumb">
            <span>Workspace</span>
            <ChevronRight size={14} />
            <strong>{section}</strong>
          </div>
          <div className="topbar-right">
            <span className="local-badge">
              <span className="status-dot" />
              Warehouse operations
            </span>
            <span className="profile-avatar" aria-label="Warehouse supervisor">
              WS
            </span>
          </div>
        </header>
        <main id="main-content" className="main-content">
          <Routes>
            <Route path="/" element={<Navigate to="/products" replace />} />
            <Route path="/products" element={<ProductsPage />} />
            <Route path="/orders" element={<OrdersPage />} />
            <Route path="/orders/new" element={<CreateOrderPage />} />
            <Route path="/orders/:id" element={<OrderDetailPage />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </main>
        <footer className="app-footer">
          <span>StockFlow</span>
          <span>Simple inventory. Confident decisions.</span>
        </footer>
      </div>
    </div>
  )
}

function NotFound() {
  return (
    <div className="empty-state">
      <h1>Page not found</h1>
      <p>This workspace page does not exist.</p>
      <Link className="button primary" to="/products">
        Go to inventory
      </Link>
    </div>
  )
}
