import { useState } from 'react'
import { Link } from 'react-router-dom'
import Header from '../components/Layout/Header'
import { useAccounts, useGlobalPerformance, useTransactions } from '../hooks'
import LoadingSpinner from '../components/common/LoadingSpinner'
import PerformanceChart from '../components/Performance/PerformanceChart'
import AccountCardWithPerformance from '../components/Accounts/AccountCardWithPerformance'

export default function Dashboard() {
  const [period, setPeriod] = useState<'1m' | '3m' | '1y' | 'all'>('all')
  
  const { data: accounts, isLoading: accountsLoading } = useAccounts()
  const { data: performance, isLoading: perfLoading } = useGlobalPerformance(period)
  const { data: transactions, isLoading: transactionsLoading } = useTransactions({
    limit: 5,
    sort_by: 'timestamp',
    sort_order: 'desc'
  })

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

  const getTransactionTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      'buy': 'Achat',
      'sell': 'Vente',
      'dividend': 'Dividende',
      'fee': 'Frais',
      'deposit': 'Dépôt',
      'withdrawal': 'Retrait',
      'interest': 'Intérêts'
    }
    return labels[type.toLowerCase()] || type
  }

  const getTransactionTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      'buy': 'text-accent-primary',
      'sell': 'text-warning',
      'dividend': 'text-success',
      'fee': 'text-error',
      'deposit': 'text-success',
      'withdrawal': 'text-error',
      'interest': 'text-success'
    }
    return colors[type.toLowerCase()] || 'text-text-primary'
  }

  return (
    <div>
      <Header 
        title="Dashboard" 
        subtitle="Vue d'ensemble de votre portefeuille"
      />
      
      <div className="p-8">
        {/* Metrics Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Valeur Totale</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <>
                <p className="text-3xl font-bold text-text-primary">
                  {formatCurrency(performance?.total_value || 0)}
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
                {formatCurrency(performance?.total_invested || 0)}
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
              <>
                <p className={`text-3xl font-bold ${((performance?.realized_gains || 0) + (performance?.unrealized_gains || 0)) >= 0 ? 'text-success' : 'text-error'}`}>
                  {((performance?.realized_gains || 0) + (performance?.unrealized_gains || 0)) >= 0 ? '+' : ''}
                  {formatCurrency((performance?.realized_gains || 0) + (performance?.unrealized_gains || 0))}
                </p>
                <p className="text-xs text-text-muted mt-1">
                  Non réalisés: {formatCurrency(performance?.unrealized_gains || 0)}
                </p>
              </>
            )}
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Frais Totaux</p>
            {perfLoading ? (
              <div className="h-10 flex items-center">
                <div className="animate-pulse bg-background-tertiary h-8 w-24 rounded"></div>
              </div>
            ) : (
              <p className="text-3xl font-bold text-warning">
                {formatCurrency(performance?.total_fees || 0)}
              </p>
            )}
          </div>
        </div>

        {/* Performance Chart */}
        <div className="mb-8">
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">Performance Globale</h2>
              <div className="flex gap-2">
                {(['1m', '3m', '1y', 'all'] as const).map((p) => (
                  <button
                    key={p}
                    onClick={() => setPeriod(p)}
                    className={`px-3 py-1 rounded text-sm transition-colors ${
                      period === p
                        ? 'bg-accent-primary text-white'
                        : 'bg-background-tertiary text-text-secondary hover:bg-background-tertiary/80'
                    }`}
                  >
                    {p === '1m' ? '1M' : p === '3m' ? '3M' : p === '1y' ? '1A' : 'Tout'}
                  </button>
                ))}
              </div>
            </div>
            {perfLoading ? (
              <div className="h-64 flex items-center justify-center">
                <LoadingSpinner />
              </div>
            ) : performance?.time_series && performance.time_series.length > 0 ? (
              <PerformanceChart data={performance.time_series} />
            ) : (
              <div className="h-64 flex items-center justify-center text-text-muted">
                Aucune donnée de performance disponible
              </div>
            )}
          </div>
        </div>

        {/* Accounts and Transactions */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Accounts */}
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
                  <AccountCardWithPerformance key={account.id} account={account} />
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

          {/* Recent Transactions */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">Dernières Transactions</h2>
              <Link to="/transactions" className="text-accent-primary hover:text-accent-hover text-sm">
                Voir tout →
              </Link>
            </div>
            <div className="space-y-3">
              {transactionsLoading ? (
                <LoadingSpinner />
              ) : transactions?.transactions && transactions.transactions.length > 0 ? (
                transactions.transactions.map((transaction) => (
                  <div
                    key={transaction.id}
                    className="flex items-center justify-between p-3 bg-background-tertiary rounded-lg"
                  >
                    <div className="flex-1">
                      <p className="text-text-primary font-medium text-sm">
                        {transaction.title || transaction.subtitle || 'Transaction'}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <span className={`text-xs font-medium ${getTransactionTypeColor(transaction.transaction_type)}`}>
                          {getTransactionTypeLabel(transaction.transaction_type)}
                        </span>
                        <span className="text-text-muted text-xs">
                          {formatDate(transaction.timestamp)}
                        </span>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className={`font-semibold ${
                        transaction.transaction_type.toLowerCase() === 'buy' || 
                        transaction.transaction_type.toLowerCase() === 'fee' ||
                        transaction.transaction_type.toLowerCase() === 'withdrawal'
                          ? 'text-error' 
                          : 'text-success'
                      }`}>
                        {(transaction.transaction_type.toLowerCase() === 'buy' || 
                          transaction.transaction_type.toLowerCase() === 'fee' ||
                          transaction.transaction_type.toLowerCase() === 'withdrawal') && '-'}
                        {formatCurrency(Math.abs(transaction.amount_value))}
                      </p>
                      {transaction.fees && parseFloat(transaction.fees.toString()) > 0 && (
                        <p className="text-xs text-text-muted">
                          Frais: {formatCurrency(parseFloat(transaction.fees.toString()))}
                        </p>
                      )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-8 text-text-muted">
                  Aucune transaction
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
