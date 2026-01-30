import { useState, useEffect } from 'react'

export interface FilterValues {
  start_date?: string
  end_date?: string
  type?: string
  asset?: string
}

interface TransactionFiltersProps {
  onFilterChange: (filters: FilterValues) => void
  initialFilters?: FilterValues
}

export default function TransactionFilters({
  onFilterChange,
  initialFilters = {},
}: TransactionFiltersProps) {
  const [filters, setFilters] = useState<FilterValues>(initialFilters)

  useEffect(() => {
    // Debounce filter changes
    const timer = setTimeout(() => {
      onFilterChange(filters)
    }, 300)

    return () => clearTimeout(timer)
  }, [filters, onFilterChange])

  const handleChange = (field: keyof FilterValues, value: string) => {
    setFilters((prev) => ({
      ...prev,
      [field]: value || undefined,
    }))
  }

  const handleReset = () => {
    setFilters({})
  }

  const hasActiveFilters = Object.values(filters).some((v) => v)

  return (
    <div className="card mb-6">
      <div className="flex flex-wrap gap-4">
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm text-text-muted mb-2">
            Date de début
          </label>
          <input
            type="date"
            className="input w-full"
            value={filters.start_date || ''}
            onChange={(e) => handleChange('start_date', e.target.value)}
          />
        </div>

        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm text-text-muted mb-2">
            Date de fin
          </label>
          <input
            type="date"
            className="input w-full"
            value={filters.end_date || ''}
            onChange={(e) => handleChange('end_date', e.target.value)}
          />
        </div>

        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm text-text-muted mb-2">Type</label>
          <select
            className="input w-full"
            value={filters.type || ''}
            onChange={(e) => handleChange('type', e.target.value)}
          >
            <option value="">Tous les types</option>
            <option value="buy">Achat</option>
            <option value="sell">Vente</option>
            <option value="dividend">Dividende</option>
            <option value="fee">Frais</option>
            <option value="deposit">Dépôt</option>
            <option value="withdrawal">Retrait</option>
            <option value="other">Autre</option>
          </select>
        </div>

        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm text-text-muted mb-2">Actif</label>
          <input
            type="text"
            className="input w-full"
            placeholder="ISIN ou nom..."
            value={filters.asset || ''}
            onChange={(e) => handleChange('asset', e.target.value)}
          />
        </div>

        {hasActiveFilters && (
          <div className="flex items-end">
            <button
              onClick={handleReset}
              className="btn-secondary whitespace-nowrap"
            >
              Réinitialiser
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
