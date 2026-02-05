import type { FeesMetrics } from '../../services/fees'

interface FeesOverviewProps {
  metrics: FeesMetrics
  isLoading?: boolean
}

export default function FeesOverview({ metrics, isLoading }: FeesOverviewProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {[1, 2, 3].map((i) => (
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
    // Handle NaN, null, undefined, or invalid values
    if (!value || isNaN(value)) {
      return '0,00 €'
    }
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  }

  const feesByType = [
    { label: 'Frais d\'achat', value: metrics.fees_by_type.buy || 0, color: 'bg-accent-primary' },
    { label: 'Frais de vente', value: metrics.fees_by_type.sell || 0, color: 'bg-accent-light' },
    { label: 'Frais de transfert', value: metrics.fees_by_type.transfer || 0, color: 'bg-warning' },
    { label: 'Autres frais', value: metrics.fees_by_type.other || 0, color: 'bg-error' },
  ]

  return (
    <div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="card">
          <p className="text-text-muted text-sm mb-2">Frais Totaux</p>
          <p className="text-3xl font-bold text-text-primary">
            {formatCurrency(metrics.total_fees || 0)}
          </p>
        </div>
        
        <div className="card">
          <p className="text-text-muted text-sm mb-2">Frais Moyens</p>
          <p className="text-3xl font-bold text-text-primary">
            {formatCurrency(metrics.average_fees || 0)}
          </p>
          <p className="text-text-muted text-sm mt-2">par transaction</p>
        </div>
        
        <div className="card">
          <p className="text-text-muted text-sm mb-2">Impact sur Performance</p>
          <p className="text-3xl font-bold text-error">
            {metrics.total_fees && !isNaN(metrics.total_fees) && metrics.total_fees > 0
              ? `-${((metrics.total_fees / (metrics.total_fees + 10000)) * 100).toFixed(2)}%`
              : '0,00%'}
          </p>
        </div>
      </div>

      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Répartition par Type</h2>
        <div className="space-y-4">
          {feesByType.map((fee, index) => (
            <div
              key={fee.label}
              className={`flex items-center justify-between py-3 ${
                index < feesByType.length - 1 ? 'border-b border-background-tertiary' : ''
              }`}
            >
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${fee.color}`}></div>
                <span className="text-text-primary">{fee.label}</span>
              </div>
              <span className="text-text-primary font-semibold">
                {formatCurrency(fee.value)}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
