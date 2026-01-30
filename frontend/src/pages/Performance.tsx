import { useState } from 'react'
import Header from '../components/Layout/Header'
import PerformanceChart from '../components/Performance/PerformanceChart'
import PerformanceMetrics from '../components/Performance/PerformanceMetrics'
import { useGlobalPerformance } from '../hooks/usePerformance'
import { useAccounts } from '../hooks/useAccounts'
import LoadingSpinner from '../components/common/LoadingSpinner'
import ErrorMessage from '../components/common/ErrorMessage'

type Period = '1m' | '3m' | '1y' | 'all'

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
                    <div
                      key={account.id}
                      className="flex items-center justify-between p-4 bg-background-tertiary rounded-lg"
                    >
                      <div>
                        <p className="font-medium text-text-primary">{account.name}</p>
                        <p className="text-sm text-text-muted capitalize">{account.platform}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-text-muted text-sm">Dernière sync</p>
                        <p className="text-text-secondary text-sm">
                          {account.last_sync
                            ? new Date(account.last_sync).toLocaleDateString('fr-FR')
                            : 'Jamais'}
                        </p>
                      </div>
                    </div>
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
