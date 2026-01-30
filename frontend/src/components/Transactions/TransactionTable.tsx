import { useState } from 'react'
import type { Transaction } from '../../types'

interface TransactionTableProps {
  transactions: Transaction[]
  isLoading: boolean
  onSort?: (column: string, direction: 'asc' | 'desc') => void
  onAssetClick?: (isin: string) => void
}

type SortColumn = 'timestamp' | 'amount'
type SortDirection = 'asc' | 'desc'

export default function TransactionTable({
  transactions,
  isLoading,
  onSort,
  onAssetClick,
}: TransactionTableProps) {
  const [sortColumn, setSortColumn] = useState<SortColumn>('timestamp')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')

  const handleSort = (column: SortColumn) => {
    const newDirection =
      sortColumn === column && sortDirection === 'asc' ? 'desc' : 'asc'
    setSortColumn(column)
    setSortDirection(newDirection)
    onSort?.(column, newDirection)
  }

  const formatDate = (timestamp: string) => {
    return new Date(timestamp).toLocaleDateString('fr-FR', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const formatAmount = (value: number, currency: string) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: currency || 'EUR',
    }).format(value)
  }

  const getTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      buy: 'Achat',
      sell: 'Vente',
      dividend: 'Dividende',
      fee: 'Frais',
      deposit: 'Dépôt',
      withdrawal: 'Retrait',
      interest: 'Intérêts',
      other: 'Autre',
    }
    return labels[type] || type
  }

  const getTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      buy: 'text-accent-primary',
      sell: 'text-error',
      dividend: 'text-success',
      fee: 'text-warning',
      deposit: 'text-success',
      withdrawal: 'text-error',
      interest: 'text-success',
      other: 'text-text-muted',
    }
    return colors[type] || 'text-text-secondary'
  }

  const SortIcon = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) {
      return <span className="text-text-muted ml-1">⇅</span>
    }
    return (
      <span className="text-accent-primary ml-1">
        {sortDirection === 'asc' ? '↑' : '↓'}
      </span>
    )
  }

  if (isLoading) {
    return (
      <div className="card">
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-primary"></div>
        </div>
      </div>
    )
  }

  if (transactions.length === 0) {
    return (
      <div className="card">
        <div className="text-center py-12 text-text-muted">
          Aucune transaction trouvée
        </div>
      </div>
    )
  }

  return (
    <div className="card">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-background-tertiary">
              <th
                className="text-left py-3 px-4 text-text-muted font-medium cursor-pointer hover:text-text-primary transition-colors"
                onClick={() => handleSort('timestamp')}
              >
                Date
                <SortIcon column="timestamp" />
              </th>
              <th className="text-left py-3 px-4 text-text-muted font-medium">
                Actif
              </th>
              <th className="text-left py-3 px-4 text-text-muted font-medium">
                Type
              </th>
              <th className="text-right py-3 px-4 text-text-muted font-medium">
                Quantité
              </th>
              <th
                className="text-right py-3 px-4 text-text-muted font-medium cursor-pointer hover:text-text-primary transition-colors"
                onClick={() => handleSort('amount')}
              >
                Montant
                <SortIcon column="amount" />
              </th>
              <th className="text-right py-3 px-4 text-text-muted font-medium">
                Frais
              </th>
            </tr>
          </thead>
          <tbody>
            {transactions.map((transaction) => (
              <tr
                key={transaction.id}
                className="border-b border-background-tertiary hover:bg-background-tertiary transition-colors"
              >
                <td className="py-3 px-4 text-text-secondary">
                  {formatDate(transaction.timestamp)}
                </td>
                <td className="py-3 px-4">
                  {transaction.isin ? (
                    <button
                      onClick={() => onAssetClick?.(transaction.isin!)}
                      className="text-accent-primary hover:text-accent-hover transition-colors text-left"
                    >
                      <div className="font-medium">{transaction.title}</div>
                      {transaction.subtitle && (
                        <div className="text-sm text-text-muted">
                          {transaction.subtitle}
                        </div>
                      )}
                      <div className="text-xs text-text-muted mt-1">
                        {transaction.isin}
                      </div>
                    </button>
                  ) : (
                    <div>
                      <div className="font-medium text-text-primary">
                        {transaction.title}
                      </div>
                      {transaction.subtitle && (
                        <div className="text-sm text-text-muted">
                          {transaction.subtitle}
                        </div>
                      )}
                    </div>
                  )}
                </td>
                <td className="py-3 px-4">
                  <span className={getTypeColor(transaction.transaction_type)}>
                    {getTypeLabel(transaction.transaction_type)}
                  </span>
                </td>
                <td className="py-3 px-4 text-right text-text-secondary">
                  {transaction.quantity
                    ? transaction.quantity.toFixed(4)
                    : '-'}
                </td>
                <td
                  className={`py-3 px-4 text-right font-medium ${
                    transaction.amount_value >= 0
                      ? 'text-success'
                      : 'text-error'
                  }`}
                >
                  {formatAmount(
                    transaction.amount_value,
                    transaction.amount_currency
                  )}
                </td>
                <td className="py-3 px-4 text-right text-text-muted">
                  {transaction.fees
                    ? formatAmount(
                        parseFloat(transaction.fees.toString()),
                        transaction.amount_currency
                      )
                    : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
