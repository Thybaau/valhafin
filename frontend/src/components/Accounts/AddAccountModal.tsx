import { useState } from 'react'
import { useCreateAccount } from '../../hooks'
import type { CreateAccountRequest } from '../../services'

interface AddAccountModalProps {
  isOpen: boolean
  onClose: () => void
}

type Platform = 'traderepublic' | 'binance' | 'boursedirect'

export default function AddAccountModal({ isOpen, onClose }: AddAccountModalProps) {
  const [name, setName] = useState('')
  const [platform, setPlatform] = useState<Platform>('traderepublic')
  const [credentials, setCredentials] = useState({
    phone_number: '',
    pin: '',
    api_key: '',
    api_secret: '',
    username: '',
    password: '',
  })

  const createMutation = useCreateAccount()

  if (!isOpen) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const data: CreateAccountRequest = {
      name,
      platform,
      credentials: {},
    }

    // Ajouter uniquement les credentials nécessaires selon la plateforme
    if (platform === 'traderepublic') {
      data.credentials.phone_number = credentials.phone_number
      data.credentials.pin = credentials.pin
    } else if (platform === 'binance') {
      data.credentials.api_key = credentials.api_key
      data.credentials.api_secret = credentials.api_secret
    } else if (platform === 'boursedirect') {
      data.credentials.username = credentials.username
      data.credentials.password = credentials.password
    }

    createMutation.mutate(data, {
      onSuccess: () => {
        // Réinitialiser le formulaire
        setName('')
        setPlatform('traderepublic')
        setCredentials({
          phone_number: '',
          pin: '',
          api_key: '',
          api_secret: '',
          username: '',
          password: '',
        })
        onClose()
      },
    })
  }

  const renderCredentialFields = () => {
    switch (platform) {
      case 'traderepublic':
        return (
          <>
            <div>
              <label className="block text-sm text-text-muted mb-2">
                Numéro de téléphone
              </label>
              <input
                type="tel"
                value={credentials.phone_number}
                onChange={(e) =>
                  setCredentials({ ...credentials, phone_number: e.target.value })
                }
                className="input w-full"
                placeholder="+33612345678"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-text-muted mb-2">Code PIN</label>
              <input
                type="password"
                value={credentials.pin}
                onChange={(e) => setCredentials({ ...credentials, pin: e.target.value })}
                className="input w-full"
                placeholder="1234"
                required
                maxLength={4}
              />
            </div>
          </>
        )

      case 'binance':
        return (
          <>
            <div>
              <label className="block text-sm text-text-muted mb-2">API Key</label>
              <input
                type="text"
                value={credentials.api_key}
                onChange={(e) =>
                  setCredentials({ ...credentials, api_key: e.target.value })
                }
                className="input w-full"
                placeholder="Votre clé API Binance"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-text-muted mb-2">API Secret</label>
              <input
                type="password"
                value={credentials.api_secret}
                onChange={(e) =>
                  setCredentials({ ...credentials, api_secret: e.target.value })
                }
                className="input w-full"
                placeholder="Votre secret API Binance"
                required
              />
            </div>
          </>
        )

      case 'boursedirect':
        return (
          <>
            <div>
              <label className="block text-sm text-text-muted mb-2">
                Nom d'utilisateur
              </label>
              <input
                type="text"
                value={credentials.username}
                onChange={(e) =>
                  setCredentials({ ...credentials, username: e.target.value })
                }
                className="input w-full"
                placeholder="Votre identifiant"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-text-muted mb-2">Mot de passe</label>
              <input
                type="password"
                value={credentials.password}
                onChange={(e) =>
                  setCredentials({ ...credentials, password: e.target.value })
                }
                className="input w-full"
                placeholder="Votre mot de passe"
                required
              />
            </div>
          </>
        )
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background-secondary rounded-lg max-w-md w-full max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold text-text-primary">Ajouter un compte</h2>
            <button
              onClick={onClose}
              className="text-text-muted hover:text-text-primary transition-colors"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm text-text-muted mb-2">Nom du compte</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="input w-full"
                placeholder="Mon compte principal"
                required
              />
            </div>

            <div>
              <label className="block text-sm text-text-muted mb-2">Plateforme</label>
              <select
                value={platform}
                onChange={(e) => setPlatform(e.target.value as Platform)}
                className="input w-full"
              >
                <option value="traderepublic">Trade Republic</option>
                <option value="binance">Binance</option>
                <option value="boursedirect">Bourse Direct</option>
              </select>
            </div>

            {renderCredentialFields()}

            <div className="bg-warning/10 border border-warning rounded-md p-3">
              <p className="text-sm text-warning">
                ⚠️ Vos identifiants seront chiffrés avant d'être stockés
              </p>
            </div>

            {createMutation.isError && (
              <div className="bg-error/10 border border-error rounded-md p-3">
                <p className="text-sm text-error">
                  Erreur lors de la création du compte. Vérifiez vos identifiants.
                </p>
              </div>
            )}

            <div className="flex gap-3 pt-4">
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="btn-primary flex-1"
              >
                {createMutation.isPending ? 'Création...' : 'Créer le compte'}
              </button>
              <button type="button" onClick={onClose} className="btn-secondary flex-1">
                Annuler
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
