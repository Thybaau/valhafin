import { useState, useRef, useEffect } from 'react'
import { useAccounts } from '../../hooks/useAccounts'
import { useImportCSV } from '../../hooks/useTransactions'

interface ImportCSVModalProps {
  isOpen: boolean
  onClose: () => void
}

export default function ImportCSVModal({
  isOpen,
  onClose,
}: ImportCSVModalProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [selectedAccountId, setSelectedAccountId] = useState<string>('')
  const [dragActive, setDragActive] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const { data: accountsData } = useAccounts()
  const importMutation = useImportCSV()

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !importMutation.isPending) {
        handleClose()
      }
    }

    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = 'unset'
    }
  }, [isOpen, importMutation.isPending])

  const handleClose = () => {
    if (!importMutation.isPending) {
      setSelectedFile(null)
      setSelectedAccountId('')
      importMutation.reset()
      onClose()
    }
  }

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const file = e.dataTransfer.files[0]
      if (file.name.endsWith('.csv')) {
        setSelectedFile(file)
      }
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0])
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!selectedFile || !selectedAccountId) {
      return
    }

    try {
      await importMutation.mutateAsync({
        accountId: selectedAccountId,
        file: selectedFile,
      })
    } catch (error) {
      // Error is handled by the mutation
      console.error('Import failed:', error)
    }
  }

  if (!isOpen) return null

  const accounts = accountsData || []

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={handleClose}
      />

      {/* Modal */}
      <div className="relative bg-background-secondary rounded-lg shadow-lg max-w-2xl w-full">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-background-tertiary">
          <h2 className="text-2xl font-bold text-text-primary">
            Importer des transactions
          </h2>
          <button
            onClick={handleClose}
            disabled={importMutation.isPending}
            className="text-text-muted hover:text-text-primary transition-colors disabled:opacity-50"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <form onSubmit={handleSubmit} className="p-6">
          {/* Account Selection */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-text-primary mb-2">
              Compte
            </label>
            <select
              className="input w-full"
              value={selectedAccountId}
              onChange={(e) => setSelectedAccountId(e.target.value)}
              required
              disabled={importMutation.isPending}
            >
              <option value="">Sélectionner un compte</option>
              {accounts.map((account) => (
                <option key={account.id} value={account.id}>
                  {account.name} ({account.platform})
                </option>
              ))}
            </select>
          </div>

          {/* File Upload */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-text-primary mb-2">
              Fichier CSV
            </label>
            <div
              className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
                dragActive
                  ? 'border-accent-primary bg-accent-primary/10'
                  : 'border-background-tertiary hover:border-accent-primary/50'
              }`}
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
            >
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv"
                onChange={handleFileChange}
                className="hidden"
                disabled={importMutation.isPending}
              />

              {selectedFile ? (
                <div className="space-y-2">
                  <div className="text-accent-primary">
                    <svg
                      className="w-12 h-12 mx-auto"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                      />
                    </svg>
                  </div>
                  <p className="text-text-primary font-medium">
                    {selectedFile.name}
                  </p>
                  <p className="text-text-muted text-sm">
                    {(selectedFile.size / 1024).toFixed(2)} KB
                  </p>
                  <button
                    type="button"
                    onClick={() => setSelectedFile(null)}
                    disabled={importMutation.isPending}
                    className="text-accent-primary hover:text-accent-hover text-sm disabled:opacity-50"
                  >
                    Changer de fichier
                  </button>
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="text-text-muted">
                    <svg
                      className="w-12 h-12 mx-auto"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                      />
                    </svg>
                  </div>
                  <p className="text-text-primary">
                    Glissez-déposez votre fichier CSV ici
                  </p>
                  <p className="text-text-muted text-sm">ou</p>
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="btn-secondary"
                  >
                    Parcourir les fichiers
                  </button>
                </div>
              )}
            </div>
            <p className="text-text-muted text-sm mt-2">
              Format attendu: timestamp, isin, amount_value, fees
            </p>
          </div>

          {/* Success Message */}
          {importMutation.isSuccess && (
            <div className="mb-6 p-4 bg-success/10 border border-success rounded-lg">
              <div className="flex items-start">
                <svg
                  className="w-5 h-5 text-success mt-0.5 mr-3"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
                <div className="flex-1">
                  <p className="text-success font-medium">
                    Import réussi !
                  </p>
                  <p className="text-success/80 text-sm mt-1">
                    {importMutation.data?.imported} transaction(s) importée(s)
                    {importMutation.data?.ignored > 0 &&
                      `, ${importMutation.data.ignored} ignorée(s)`}
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Error Message */}
          {importMutation.isError && (
            <div className="mb-6 p-4 bg-error/10 border border-error rounded-lg">
              <div className="flex items-start">
                <svg
                  className="w-5 h-5 text-error mt-0.5 mr-3"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
                <div className="flex-1">
                  <p className="text-error font-medium">
                    Erreur lors de l'import
                  </p>
                  <p className="text-error/80 text-sm mt-1">
                    {importMutation.error instanceof Error
                      ? importMutation.error.message
                      : 'Une erreur est survenue'}
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={handleClose}
              disabled={importMutation.isPending}
              className="btn-secondary"
            >
              {importMutation.isSuccess ? 'Fermer' : 'Annuler'}
            </button>
            {!importMutation.isSuccess && (
              <button
                type="submit"
                disabled={
                  !selectedFile ||
                  !selectedAccountId ||
                  importMutation.isPending
                }
                className="btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {importMutation.isPending ? (
                  <span className="flex items-center">
                    <svg
                      className="animate-spin -ml-1 mr-2 h-4 w-4"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      />
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      />
                    </svg>
                    Import en cours...
                  </span>
                ) : (
                  'Importer'
                )}
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  )
}
