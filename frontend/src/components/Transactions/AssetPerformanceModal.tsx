import { useEffect } from 'react'
import { useAssetPerformance, useAssetPrice } from '../../hooks/usePerformance'
import PerformanceChart from '../Performance/PerformanceChart'
import LoadingSpinner from '../common/LoadingSpinner'
import ErrorMessage from '../common/ErrorMessage'

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
  const { data: performance, isLoading: perfLoading, error: perfError } = useAssetPerformance(isin, isOpen)
  const { data: priceData, isLoading: priceLoading } = useAssetPrice(isin, isOpen)

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

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  }

  const formatPercentage = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`
  }

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
            {priceData && (
              <p className="text-accent-primary mt-1">
                Prix actuel: {formatCurrency(priceData.price)}
              </p>
            )}
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
        <div className="p-6 space-y-6">
          {perfError && (
            <ErrorMessage message="Impossible de charger les données de performance" />
          )}

          {(perfLoading || priceLoading) && <LoadingSpinner />}

          {!perfLoading && !priceLoading && performance && (
            <>
              {/* Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="card">
                  <div className="text-text-muted text-sm mb-1">Valeur actuelle</div>
                  <div className="text-xl font-bold text-text-primary">
                    {formatCurrency(performance.total_value)}
                  </div>
                </div>

                <div className="card">
                  <div className="text-text-muted text-sm mb-1">Investissement</div>
                  <div className="text-xl font-bold text-text-primary">
                    {formatCurrency(performance.total_invested)}
                  </div>
                </div>

                <div className="card">
                  <div className="text-text-muted text-sm mb-1">Gains/Pertes</div>
                  <div
                    className={`text-xl font-bold ${
                      performance.unrealized_gains >= 0 ? 'text-success' : 'text-error'
                    }`}
                  >
                    {formatCurrency(performance.unrealized_gains)}
                  </div>
                  <div
                    className={`text-sm ${
                      performance.performance_pct >= 0 ? 'text-success' : 'text-error'
                    }`}
                  >
                    {formatPercentage(performance.performance_pct)}
                  </div>
                </div>
              </div>

              {/* Chart */}
              {performance.time_series && performance.time_series.length > 0 && (
                <div>
                  <h3 className="text-lg font-semibold text-text-primary mb-4">
                    Évolution de la valeur
                  </h3>
                  <PerformanceChart data={performance.time_series} />
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
