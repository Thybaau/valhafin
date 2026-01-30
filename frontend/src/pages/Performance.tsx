import { useState } from 'react'
import Header from '../components/Layout/Header'

type Period = '1m' | '3m' | '1y' | 'all'

export default function Performance() {
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
        title="Performance" 
        subtitle="Évolution de votre portefeuille"
      />
      
      <div className="p-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Valeur Actuelle</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Investissement Total</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Performance</p>
            <p className="text-3xl font-bold text-success">+0.00%</p>
            <p className="text-success text-sm mt-2">+€0.00</p>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold">Évolution de la Valeur</h2>
            
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

          <div className="h-96 flex items-center justify-center text-text-muted">
            Graphique de performance à venir
          </div>
        </div>

        <div className="card mt-6">
          <h2 className="text-xl font-semibold mb-4">Performance par Compte</h2>
          <div className="space-y-4">
            <p className="text-text-muted text-center py-8">
              Aucun compte connecté
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
