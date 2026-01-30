import { useState, useCallback } from 'react'
import Header from '../components/Layout/Header'
import TransactionTable from '../components/Transactions/TransactionTable'
import TransactionFilters from '../components/Transactions/TransactionFilters'
import AssetPerformanceModal from '../components/Transactions/AssetPerformanceModal'
import ImportCSVModal from '../components/Transactions/ImportCSVModal'
import Pagination from '../components/common/Pagination'
import { useTransactions } from '../hooks/useTransactions'
import type { FilterValues } from '../components/Transactions/TransactionFilters'

export default function Transactions() {
  const [filters, setFilters] = useState<FilterValues>({})
  const [page, setPage] = useState(1)
  const [sortBy, setSortBy] = useState<string>('timestamp')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [selectedAsset, setSelectedAsset] = useState<string | null>(null)
  const [showImportModal, setShowImportModal] = useState(false)

  const limit = 50

  const { data, isLoading } = useTransactions({
    ...filters,
    page,
    limit,
    sort_by: sortBy,
    sort_order: sortOrder,
  })

  const handleFilterChange = useCallback((newFilters: FilterValues) => {
    setFilters(newFilters)
    setPage(1) // Reset to first page when filters change
  }, [])

  const handleSort = (column: string, direction: 'asc' | 'desc') => {
    setSortBy(column)
    setSortOrder(direction)
  }

  const handleAssetClick = (isin: string) => {
    setSelectedAsset(isin)
  }

  const handlePageChange = (newPage: number) => {
    console.log('Page change:', newPage)
    setPage(newPage)
  }

  return (
    <div>
      <Header
        title="Transactions"
        subtitle="Historique de toutes vos transactions"
        actions={
          <button
            className="btn-primary"
            onClick={() => setShowImportModal(true)}
          >
            Importer CSV
          </button>
        }
      />

      <div className="p-8">
        <TransactionFilters
          onFilterChange={handleFilterChange}
          initialFilters={filters}
        />

        <TransactionTable
          transactions={data?.transactions || []}
          isLoading={isLoading}
          onSort={handleSort}
          onAssetClick={handleAssetClick}
        />

        {data && data.total_pages > 1 && (
          <div className="mt-6">
            <Pagination
              currentPage={page}
              totalPages={data.total_pages}
              onPageChange={handlePageChange}
            />
          </div>
        )}
      </div>

      <AssetPerformanceModal
        isin={selectedAsset || ''}
        isOpen={!!selectedAsset}
        onClose={() => setSelectedAsset(null)}
      />

      <ImportCSVModal
        isOpen={showImportModal}
        onClose={() => setShowImportModal(false)}
      />
    </div>
  )
}
