import { useState } from 'react'
import Header from '../components/Layout/Header'
import PerformanceChart from '../components/Performance/PerformanceChart'
import PerformanceMetrics from '../components/Performance/PerformanceMetrics'
import { useGlobalPerformance, useAccountPerformance } from '../hooks/usePerformance'
import { useAccounts } from '../hooks/useAccounts'
import LoadingSpinner from '../components/common/LoadingSpinner'
import ErrorMessage from '../components/common/ErrorMessage'

type Period = '1m' | '3m' | '1y' | 'all'

interface Account {
  id: string
  name: string
  platform: string
  last_sync?: string
}

function AccountPerformanceCard({ account, period }: { account: Account; period: Period }) {
  const { data: accountPerf, isLoading } = useAccountPerformance(account.id, period)

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
    }).format(value)
  }

  const formatPercent = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
  }

  if (isLoading) {
    return (
      <div className="p-4 bg-background-tertiary rounded-lg">
        <div className="animate-pulse flex space-x-4">
          <div className="flex-1 space-y-2">
            <div className="h-4 bg-background-primary rounded w-3/4"></div>
            <div className="h-3 bg-background-primary rounded w-1/2"></div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="p-4 bg-background-tertiary rounded-lg">
      <div className="flex items-start justify-between mb-3">
        <div>
          <p className="font-medium text-text-primary text-lg">{account.name}</p>
          <p className="text-sm text-text-muted capitalize">{account.platform}</p>
        </div>
        <div className="text-right">
          <p className="text-text-muted text-xs">Dernière sync</p>
          <p className="text-text-secondary text-sm">
            {account.last_sync
              ? new Date(account.last_sync).toLocaleDateString('fr-FR')
              : 'Jamais'}
          </p>
        </div>
      </div>

      {accountPerf && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mt-3">
          <div className="bg-background-primary rounded-lg p-3">
            <p className="text-xs text-text-muted mb-1">Valeur</p>
            <p className="text-sm font-semibold text-text-primary">
              {formatCurrency(accountPerf.total_value)}
            </p>
          </div>
          <div className="bg-background-primary rounded-lg p-3">
            <p className="text-xs text-text-muted mb-1">Investi</p>
            <p className="text-sm font-semibold text-text-primary">
              {formatCurrency(accountPerf.total_invested)}
            </p>
          </div>
          <div className="bg-background-primary rounded-lg p-3">
            <p className="text-xs text-text-muted mb-1">Gain/Perte</p>
            <p className={`text-sm font-semibold ${accountPerf.unrealized_gains >= 0 ? 'text-success' : 'text-error'}`}>
              {formatCurrency(accountPerf.unrealized_gains)}
            </p>
          </div>
          <div className="bg-background-primary rounded-lg p-3">
            <p className="text-xs text-text-muted mb-1">Performance</p>
            <p className={`text-sm font-semibold ${accountPerf.performance_pct >= 0 ? 'text-success' : 'text-error'}`}>
              {formatPercent(accountPerf.performance_pct)}
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

export default function Performance() {
  const [period, setPeriod] = useState<Period>('all')

  const { data: performance, isLoading, error } = useGlobalPerformance(period)
  const { data: accounts } = useAccounts()

  const periods: { value: Period; label: string }[] = [
    { value: '1m', label: '1 Mois' },
    { value: '3m', label: '3 Mois' },
    { value: '1y', label: '1 An' },
    { value: 'all', label: 'Tout' },
  ]

  return (
    <div>
      <Header 
        title="Performance" 
        subtitle="Évolution de votre portefeuille"
      />
      
      <div className="p-8 space-y-6">
        {error && <ErrorMessage message="Impossible de charger les données de performance" />}
        
        {isLoading && <LoadingSpinner />}

        {!isLoading && performance && (
          <>
            <PerformanceMetrics performance={performance} />

            <div className="card">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-xl font-semibold text-text-primary">Évolution de la Valeur</h2>
                
                <div className="flex gap-2">
                  {periods.map((p) => (
                    <button
                      key={p.value}
                      onClick={() => setPeriod(p.value)}
                      className={`px-4 py-2 rounded-md transition-colors ${
                        period === p.value
                          ? 'bg-accent-primary text-white'
                          : 'bg-background-tertiary text-text-secondary hover:bg-background-primary'
                      }`}
                    >
                      {p.label}
                    </button>
                  ))}
                </div>
              </div>

              <PerformanceChart data={performance.time_series} />
            </div>

            {accounts && accounts.length > 0 && (
              <div className="card">
                <h2 className="text-xl font-semibold mb-4 text-text-primary">Performance par Compte</h2>
                <div className="space-y-4">
                  {accounts.map((account) => (
                    <AccountPerformanceCard 
                      key={account.id} 
                      account={account} 
                      period={period}
                    />
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
