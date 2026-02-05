import { useState } from 'react'
import { useAccounts } from '../../hooks'
import AccountCard from './AccountCard'
import AddAccountModal from './AddAccountModal'
import LoadingSpinner from '../common/LoadingSpinner'
import ErrorMessage from '../common/ErrorMessage'

export default function AccountList() {
  const [isModalOpen, setIsModalOpen] = useState(false)
  const { data: accounts, isLoading, isError, error, refetch } = useAccounts()

  if (isLoading) {
    return <LoadingSpinner />
  }

  if (isError) {
    return (
      <ErrorMessage
        message={error?.message || 'Erreur lors du chargement des comptes'}
        onRetry={() => refetch()}
      />
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">Mes Comptes</h2>
          <p className="text-text-muted mt-1">
            {accounts?.length || 0} compte(s) connecté(s)
          </p>
        </div>
        <button onClick={() => setIsModalOpen(true)} className="btn-primary">
          <svg
            className="w-5 h-5 inline-block mr-2"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 4v16m8-8H4"
            />
          </svg>
          Ajouter un compte
        </button>
      </div>

      {accounts && accounts.length > 0 ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {accounts.map((account) => (
            <AccountCard key={account.id} account={account} />
          ))}
        </div>
      ) : (
        <div className="card text-center py-12">
          <svg
            className="w-16 h-16 mx-auto text-text-muted mb-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
            />
          </svg>
          <h3 className="text-xl font-semibold text-text-primary mb-2">
            Aucun compte connecté
          </h3>
          <p className="text-text-muted mb-6">
            Commencez par ajouter votre premier compte pour synchroniser vos transactions
          </p>
          <button onClick={() => setIsModalOpen(true)} className="btn-primary">
            Ajouter mon premier compte
          </button>
        </div>
      )}

      <AddAccountModal isOpen={isModalOpen} onClose={() => setIsModalOpen(false)} />
    </div>
  )
}
