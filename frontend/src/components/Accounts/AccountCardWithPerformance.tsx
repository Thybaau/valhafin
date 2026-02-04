import { Link } from 'react-router-dom'
import { useAccountPerformance } from '../../hooks'
import type { Account } from '../../types'

interface AccountCardWithPerformanceProps {
  account: Account
}

export default function AccountCardWithPerformance({ account }: AccountCardWithPerformanceProps) {
  const { data: accountPerf } = useAccountPerformance(account.id, 'all')

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: 'short',
      year: 'numeric'
    })
  }

  return (
    <Link
      to="/accounts"
      className="block p-4 bg-background-tertiary rounded-lg hover:bg-background-tertiary/80 transition-colors"
    >
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <p className="text-text-primary font-medium">{account.name}</p>
          <p className="text-text-muted text-sm capitalize">{account.platform}</p>
        </div>
        <div className="text-right">
          {accountPerf ? (
            <>
              <p className="text-text-primary font-semibold">
                {formatCurrency(accountPerf.total_value)}
              </p>
              <p className={`text-sm ${accountPerf.performance_pct >= 0 ? 'text-success' : 'text-error'}`}>
                {accountPerf.performance_pct >= 0 ? '+' : ''}
                {accountPerf.performance_pct.toFixed(2)}%
              </p>
            </>
          ) : (
            <p className="text-text-muted text-sm">
              {account.last_sync
                ? `Sync: ${formatDate(account.last_sync)}`
                : 'Non synchronisé'}
            </p>
          )}
        </div>
      </div>
    </Link>
  )
}
