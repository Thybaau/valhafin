import { useState } from 'react'
import type { Account } from '../../types'
import { useSyncAccount, useDeleteAccount, useInitSync, useCompleteSync } from '../../hooks'

interface AccountCardProps {
  account: Account
}

export default function AccountCard({ account }: AccountCardProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [show2FAModal, setShow2FAModal] = useState(false)
  const [processId, setProcessId] = useState('')
  const [twoFACode, setTwoFACode] = useState('')
  
  const syncMutation = useSyncAccount()
  const deleteMutation = useDeleteAccount()
  const initSyncMutation = useInitSync()
  const completeSyncMutation = useCompleteSync()

  const platformLabels: Record<string, string> = {
    traderepublic: 'Trade Republic',
    binance: 'Binance',
    boursedirect: 'Bourse Direct',
  }

  const platformColors: Record<string, string> = {
    traderepublic: 'bg-blue-500',
    binance: 'bg-yellow-500',
    boursedirect: 'bg-green-500',
  }

  const handleSync = () => {
    // Pour Trade Republic, on doit d'abord initier la 2FA
    if (account.platform === 'traderepublic') {
      initSyncMutation.mutate(account.id, {
        onSuccess: (data) => {
          if (data.requires_two_factor && data.process_id) {
            setProcessId(data.process_id)
            setShow2FAModal(true)
          } else {
            // Pas de 2FA nécessaire (ne devrait pas arriver)
            syncMutation.mutate(account.id)
          }
        },
      })
    } else {
      // Pour les autres plateformes, sync directe
      syncMutation.mutate(account.id)
    }
  }

  const handleComplete2FA = () => {
    if (!twoFACode || !processId) return

    completeSyncMutation.mutate(
      { id: account.id, data: { process_id: processId, code: twoFACode } },
      {
        onSuccess: () => {
          setShow2FAModal(false)
          setTwoFACode('')
          setProcessId('')
        },
      }
    )
  }

  const handleDelete = () => {
    deleteMutation.mutate(account.id, {
      onSuccess: () => setShowDeleteConfirm(false),
    })
  }

  const formatDate = (dateString: string | null) => {
    if (!dateString) return 'Jamais'
    return new Date(dateString).toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const isLoading = syncMutation.isPending || initSyncMutation.isPending || completeSyncMutation.isPending

  return (
    <>
      <div className="card">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className={`w-3 h-3 rounded-full ${platformColors[account.platform]}`}></div>
            <div>
              <h3 className="text-lg font-semibold text-text-primary">{account.name}</h3>
              <p className="text-sm text-text-muted">{platformLabels[account.platform]}</p>
            </div>
          </div>

          {!showDeleteConfirm && (
            <button
              onClick={() => setShowDeleteConfirm(true)}
              className="text-error hover:text-error/80 transition-colors"
              title="Supprimer le compte"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                />
              </svg>
            </button>
          )}
        </div>

        {showDeleteConfirm && (
          <div className="mb-4 p-3 bg-error/10 border border-error rounded-md">
            <p className="text-sm text-error mb-3">
              Êtes-vous sûr de vouloir supprimer ce compte ? Toutes les transactions seront supprimées.
            </p>
            <div className="flex gap-2">
              <button
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="btn-primary bg-error hover:bg-error/80 text-sm"
              >
                {deleteMutation.isPending ? 'Suppression...' : 'Confirmer'}
              </button>
              <button
                onClick={() => setShowDeleteConfirm(false)}
                className="btn-secondary text-sm"
              >
                Annuler
              </button>
            </div>
          </div>
        )}

        <div className="space-y-2 mb-4">
          <div className="flex justify-between text-sm">
            <span className="text-text-muted">Dernière sync:</span>
            <span className="text-text-primary">{formatDate(account.last_sync)}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-text-muted">Créé le:</span>
            <span className="text-text-primary">{formatDate(account.created_at)}</span>
          </div>
        </div>

        <button
          onClick={handleSync}
          disabled={isLoading}
          className="btn-primary w-full flex items-center justify-center gap-2"
        >
          {isLoading ? (
            <>
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
              Synchronisation...
            </>
          ) : (
            <>
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                />
              </svg>
              Synchroniser
            </>
          )}
        </button>

        {syncMutation.isSuccess && (
          <div className="mt-3 p-2 bg-success/10 border border-success rounded-md">
            <p className="text-sm text-success">
              ✓ {syncMutation.data.transactions_added} transaction(s) ajoutée(s)
            </p>
          </div>
        )}

        {completeSyncMutation.isSuccess && (
          <div className="mt-3 p-2 bg-success/10 border border-success rounded-md">
            <p className="text-sm text-success">
              ✓ {completeSyncMutation.data.transactions_added} transaction(s) ajoutée(s)
            </p>
          </div>
        )}

        {(syncMutation.isError || initSyncMutation.isError || completeSyncMutation.isError) && (
          <div className="mt-3 p-2 bg-error/10 border border-error rounded-md">
            <p className="text-sm text-error">
              Erreur lors de la synchronisation
            </p>
          </div>
        )}
      </div>

      {/* Modal 2FA pour Trade Republic */}
      {show2FAModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-background-secondary rounded-lg max-w-md w-full p-6">
            <h3 className="text-xl font-bold text-text-primary mb-4">
              Code de vérification Trade Republic
            </h3>
            <p className="text-text-secondary mb-4">
              Consultez votre application Trade Republic pour obtenir le code de vérification.
            </p>
            <input
              type="text"
              value={twoFACode}
              onChange={(e) => setTwoFACode(e.target.value)}
              placeholder="Entrez le code"
              className="input w-full mb-4"
              maxLength={4}
              autoFocus
            />
            <div className="flex gap-3">
              <button
                onClick={handleComplete2FA}
                disabled={!twoFACode || completeSyncMutation.isPending}
                className="btn-primary flex-1"
              >
                {completeSyncMutation.isPending ? 'Vérification...' : 'Valider'}
              </button>
              <button
                onClick={() => {
                  setShow2FAModal(false)
                  setTwoFACode('')
                }}
                className="btn-secondary flex-1"
              >
                Annuler
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
