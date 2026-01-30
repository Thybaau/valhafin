import { Link } from 'react-router-dom'
import Header from '../components/Layout/Header'
import { useAccounts, useGlobalPerformance } from '../hooks'
import LoadingSpinner from '../components/common/LoadingSpinner'

export default function Dashboard() {
  const { data: accounts, isLoading: accountsLoading } = useAccounts()
  const { data: performance, isLoading: perfLoading } = useGlobalPerformance('1m')
  return (
    <div>
      <Header 
        title="Dashboard" 
        subtitle="Vue d'ensemble de votre portefeuille"
      />
      
      <div className="p-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {/* Placeholder cards for metrics */}
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Valeur Totale</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <>
                <p className="text-3xl font-bold text-text-primary">
                  €{performance?.total_value.toFixed(2) || '0.00'}
                </p>
                <p className={`text-sm mt-2 ${(performance?.performance_pct || 0) >= 0 ? 'text-success' : 'text-error'}`}>
                  {(performance?.performance_pct || 0) >= 0 ? '+' : ''}
                  {performance?.performance_pct.toFixed(2) || '0.00'}%
                </p>
              </>
            )}
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Investissement</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <p className="text-3xl font-bold text-text-primary">
                €{performance?.total_invested.toFixed(2) || '0.00'}
              </p>
            )}
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Gains/Pertes</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <p className={`text-3xl font-bold ${(performance?.unrealized_gains || 0) >= 0 ? 'text-success' : 'text-error'}`}>
                {(performance?.unrealized_gains || 0) >= 0 ? '+' : ''}
                €{performance?.unrealized_gains.toFixed(2) || '0.00'}
              </p>
            )}
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Frais Totaux</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <p className="text-3xl font-bold text-text-primary">
                €{performance?.total_fees.toFixed(2) || '0.00'}
              </p>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card">
            <h2 className="text-xl font-semibold mb-4">Performance Globale</h2>
            <div className="h-64 flex items-center justify-center text-text-muted">
              Graphique de performance à venir
            </div>
          </div>

          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">Comptes</h2>
              <Link to="/accounts" className="text-accent-primary hover:text-accent-hover text-sm">
                Voir tout →
              </Link>
            </div>
            <div className="space-y-3">
              {accountsLoading ? (
                <LoadingSpinner />
              ) : accounts && accounts.length > 0 ? (
                accounts.slice(0, 3).map((account) => (
                  <div
                    key={account.id}
                    className="flex items-center justify-between p-3 bg-background-tertiary rounded-md"
                  >
                    <div>
                      <p className="text-text-primary font-medium">{account.name}</p>
                      <p className="text-text-muted text-sm capitalize">{account.platform}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-text-primary text-sm">
                        {account.last_sync
                          ? new Date(account.last_sync).toLocaleDateString('fr-FR')
                          : 'Non synchronisé'}
                      </p>
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-8">
                  <p className="text-text-muted mb-4">Aucun compte connecté</p>
                  <Link to="/accounts" className="btn-primary inline-block">
                    Ajouter un compte
                  </Link>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="card mt-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold">Dernières Transactions</h2>
            <Link to="/transactions" className="text-accent-primary hover:text-accent-hover text-sm">
              Voir tout →
            </Link>
          </div>
          <div className="text-text-muted text-center py-8">
            Aucune transaction
          </div>
        </div>
      </div>
    </div>
  )
}
