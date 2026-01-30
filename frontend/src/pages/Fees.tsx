import { useState } from 'react'
import Header from '../components/Layout/Header'

type Period = '1m' | '3m' | '1y' | 'all'

export default function Fees() {
  const [period, setPeriod] = useState<Period>('1m')

  const periods: { value: Period; label: string }[] = [
    { value: '1m', label: '1 Mois' },
    { value: '3m', label: '3 Mois' },
    { value: '1y', label: '1 An' },
    { value: 'all', label: 'Tout' },
  ]

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

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Frais Totaux</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Frais Moyens</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
            <p className="text-text-muted text-sm mt-2">par transaction</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Impact sur Performance</p>
            <p className="text-3xl font-bold text-error">-0.00%</p>
          </div>
        </div>

        <div className="card mb-6">
          <h2 className="text-xl font-semibold mb-4">Évolution des Frais</h2>
          <div className="h-64 flex items-center justify-center text-text-muted">
            Graphique d'évolution des frais à venir
          </div>
        </div>

        <div className="card">
          <h2 className="text-xl font-semibold mb-4">Répartition par Type</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between py-3 border-b border-background-tertiary">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-accent-primary"></div>
                <span className="text-text-primary">Frais d'achat</span>
              </div>
              <span className="text-text-primary font-semibold">€0.00</span>
            </div>
            
            <div className="flex items-center justify-between py-3 border-b border-background-tertiary">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-accent-light"></div>
                <span className="text-text-primary">Frais de vente</span>
              </div>
              <span className="text-text-primary font-semibold">€0.00</span>
            </div>
            
            <div className="flex items-center justify-between py-3">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-warning"></div>
                <span className="text-text-primary">Autres frais</span>
              </div>
              <span className="text-text-primary font-semibold">€0.00</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
