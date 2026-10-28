import { Component } from 'react'
import type { ReactNode } from 'react'

/** Render a visible recovery screen if a descendant fails during rendering. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }
  /** React calls this lifecycle hook after a descendant throws. */
  static getDerivedStateFromError() {
    return { failed: true }
  }
  /** Normal operation returns children; failure provides a full reload recovery action. */
  render() {
    if (this.state.failed)
      return (
        <main className="empty-state">
          <h1>Something interrupted StockFlow</h1>
          <p>Reload the workspace to try again.</p>
          <button className="button primary" onClick={() => window.location.reload()}>
            Reload workspace
          </button>
        </main>
      )
    return this.props.children
  }
}
