import type { Performance } from '../../types'

interface PerformanceMetricsProps {
  performance: Performance
  isLoading?: boolean
}

export default function PerformanceMetrics({ performance, isLoading }: PerformanceMetricsProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="card">
            <div className="animate-pulse">
              <div className="h-4 bg-background-tertiary rounded w-1/2 mb-2"></div>
              <div className="h-8 bg-background-tertiary rounded w-3/4"></div>
            </div>
          </div>
        ))}
      </div>
    )
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  }

  const formatPercentage = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
  }

  const totalGains = performance.realized_gains + performance.unrealized_gains

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
      <div className="card">
        <div className="text-text-muted text-sm mb-1">Valeur totale</div>
        <div className="text-2xl font-bold text-text-primary">
          {formatCurrency(performance.total_value)}
        </div>
      </div>

      <div className="card">
        <div className="text-text-muted text-sm mb-1">Investissement</div>
        <div className="text-2xl font-bold text-text-primary">
          {formatCurrency(performance.total_invested)}
        </div>
      </div>

      <div className="card">
        <div className="text-text-muted text-sm mb-1">Espèces</div>
        <div className="text-2xl font-bold text-text-primary">
          {formatCurrency(performance.cash_balance)}
        </div>
      </div>

      <div className="card">
        <div className="text-text-muted text-sm mb-1">Gains/Pertes</div>
        <div
          className={`text-2xl font-bold ${
            totalGains >= 0 ? 'text-success' : 'text-error'
          }`}
        >
          {formatCurrency(totalGains)}
        </div>
        <div
          className={`text-sm ${
            performance.performance_pct >= 0 ? 'text-success' : 'text-error'
          }`}
        >
          {formatPercentage(performance.performance_pct)}
        </div>
        <div className="text-xs text-text-muted mt-2">
          Non réalisés: {formatCurrency(performance.unrealized_gains)}
          <br />
          Réalisés: {formatCurrency(performance.realized_gains)}
        </div>
      </div>

      <div className="card">
        <div className="text-text-muted text-sm mb-1">Frais totaux</div>
        <div className="text-2xl font-bold text-warning">
          {formatCurrency(performance.total_fees)}
        </div>
      </div>
    </div>
  )
}
