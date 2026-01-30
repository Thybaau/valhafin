import { useEffect } from 'react'

interface AssetPerformanceModalProps {
  isin: string
  isOpen: boolean
  onClose: () => void
}

export default function AssetPerformanceModal({
  isin,
  isOpen,
  onClose,
}: AssetPerformanceModalProps) {
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose()
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
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-background-secondary rounded-lg shadow-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-background-tertiary">
          <div>
            <h2 className="text-2xl font-bold text-text-primary">
              Performance de l'actif
            </h2>
            <p className="text-text-muted mt-1">ISIN: {isin}</p>
          </div>
          <button
            onClick={onClose}
            className="text-text-muted hover:text-text-primary transition-colors"
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
        <div className="p-6">
          <div className="text-center py-12 text-text-muted">
            Fonctionnalité en cours de développement
            <br />
            <span className="text-sm">
              Les graphiques de performance par actif seront disponibles
              prochainement
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
