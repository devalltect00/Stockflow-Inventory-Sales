import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { App } from './app/App'
import { ErrorBoundary } from './app/ErrorBoundary'
import './styles/global.css'

const client = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 15_00
    },
    mutations: {
      retry: false
    }
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={client}>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>,
)
