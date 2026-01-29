import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      retry: 3,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="min-h-screen" style={{ backgroundColor: 'var(--color-background-primary)' }}>
          <div className="container mx-auto px-4 py-8">
            <h1 className="text-4xl font-bold mb-4" style={{ color: 'var(--color-accent-primary)' }}>
              Valhafin
            </h1>
            <p style={{ color: 'var(--color-text-secondary)' }}>
              Application Web de Gestion de Portefeuille Financier
            </p>
            <div className="mt-8 card">
              <h2 className="text-2xl font-semibold mb-4">Bienvenue</h2>
              <p className="mb-4" style={{ color: 'var(--color-text-secondary)' }}>
                L'infrastructure de base est configurée. Le développement des fonctionnalités commence maintenant.
              </p>
              <button className="btn-primary">
                Commencer
              </button>
            </div>
          </div>
        </div>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
