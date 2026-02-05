import { useState, useMemo } from 'react'
import Header from '../components/Layout/Header'
import FeesOverview from '../components/Fees/FeesOverview'
import FeesChart from '../components/Fees/FeesChart'
import ErrorMessage from '../components/common/ErrorMessage'
import { useGlobalFees } from '../hooks/useFees'

type Period = '1m' | '3m' | '1y' | 'all'

export default function Fees() {
  const [period, setPeriod] = useState<Period>('1m')

  const periods: { value: Period; label: string }[] = [
    { value: '1m', label: '1 Mois' },
    { value: '3m', label: '3 Mois' },
    { value: '1y', label: '1 An' },
    { value: 'all', label: 'Tout' },
  ]

  // Calculate date range based on period
  const filters = useMemo(() => {
    const endDate = new Date()
    const startDate = new Date()

    switch (period) {
      case '1m':
        startDate.setMonth(startDate.getMonth() - 1)
        break
      case '3m':
        startDate.setMonth(startDate.getMonth() - 3)
        break
      case '1y':
        startDate.setFullYear(startDate.getFullYear() - 1)
        break
      case 'all':
        return { period: 'all' as const }
    }

    return {
      start_date: startDate.toISOString().split('T')[0],
      end_date: endDate.toISOString().split('T')[0],
    }
  }, [period])

  const { data: feesData, isLoading, error } = useGlobalFees(filters)

  return (
    <div>
      <Header 
        title="Frais" 
        subtitle="Analyse détaillée de vos frais de transaction"
      />
      
      <div className="p-8">
        <div className="flex gap-2 mb-6">
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

        {error && (
          <div className="mb-6">
            <ErrorMessage message="Erreur lors du chargement des données de frais" />
          </div>
        )}

        {feesData && (
          <>
            <FeesOverview metrics={feesData} isLoading={isLoading} />
            
            <div className="mt-8">
              <FeesChart data={feesData.time_series} isLoading={isLoading} />
            </div>
          </>
        )}

        {!feesData && !isLoading && !error && (
          <div className="card text-center py-12 text-text-muted">
            Aucune donnée de frais disponible
          </div>
        )}
      </div>
    </div>
  )
}
