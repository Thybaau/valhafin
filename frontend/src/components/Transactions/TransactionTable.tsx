import { useState } from 'react'
import type { Transaction } from '../../types'

interface TransactionTableProps {
  transactions: Transaction[]
  isLoading: boolean
  onSort?: (column: string, direction: 'asc' | 'desc') => void
  onAssetClick?: (isin: string) => void
  onEdit?: (transaction: Transaction) => void
  onUpdate?: (id: string, updates: Partial<Transaction>) => void
}

type SortColumn = 'timestamp' | 'amount'
type SortDirection = 'asc' | 'desc'

interface EditingField {
  transactionId: string
  field: string
  value: string
}

const TRANSACTION_TYPES = [
  { value: 'buy', label: 'Achat' },
  { value: 'sell', label: 'Vente' },
  { value: 'dividend', label: 'Dividende' },
  { value: 'interest', label: 'Intérêts' },
  { value: 'deposit', label: 'Dépôt' },
  { value: 'withdrawal', label: 'Retrait' },
  { value: 'fee', label: 'Frais' },
  { value: 'other', label: 'Autre' },
]

export default function TransactionTable({
  transactions,
  isLoading,
  onSort,
  onAssetClick,
  onUpdate,
}: TransactionTableProps) {
  const [sortColumn, setSortColumn] = useState<SortColumn>('timestamp')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')
  const [editingRow, setEditingRow] = useState<string | null>(null)
  const [editingField, setEditingField] = useState<EditingField | null>(null)

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
    // Gérer les valeurs NaN ou invalides
    const numValue = isNaN(value) ? 0 : value
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: currency || 'EUR',
    }).format(numValue)
  }

  const formatFees = (fees: string | number | null | undefined, currency: string) => {
    if (!fees) return formatAmount(0, currency)
    const numFees = typeof fees === 'string' ? parseFloat(fees) : fees
    return formatAmount(isNaN(numFees) ? 0 : numFees, currency)
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

  const handleEditClick = (transactionId: string) => {
    if (editingRow === transactionId) {
      setEditingRow(null)
      setEditingField(null)
    } else {
      setEditingRow(transactionId)
      setEditingField(null)
    }
  }

  const handleFieldEdit = (transactionId: string, field: string, currentValue: string) => {
    setEditingField({ transactionId, field, value: currentValue })
  }

  const handleFieldSave = async (transaction: Transaction) => {
    if (!editingField || !onUpdate) return

    const updates: Partial<Transaction> = {
      id: transaction.id,
      account_id: transaction.account_id,
      timestamp: transaction.timestamp,
      status: transaction.status,
    }

    switch (editingField.field) {
      case 'title':
        updates.title = editingField.value
        updates.subtitle = transaction.subtitle
        updates.amount_value = transaction.amount_value
        updates.amount_currency = transaction.amount_currency
        updates.fees = transaction.fees
        updates.quantity = transaction.quantity
        updates.transaction_type = transaction.transaction_type
        updates.isin = transaction.isin
        break
      case 'type':
        updates.title = transaction.title
        updates.subtitle = transaction.subtitle
        updates.amount_value = transaction.amount_value
        updates.amount_currency = transaction.amount_currency
        updates.fees = transaction.fees
        updates.quantity = transaction.quantity
        updates.transaction_type = editingField.value
        updates.isin = transaction.isin
        break
      case 'amount':
        updates.title = transaction.title
        updates.subtitle = transaction.subtitle
        updates.amount_value = parseFloat(editingField.value)
        updates.amount_currency = transaction.amount_currency
        updates.fees = transaction.fees
        updates.quantity = transaction.quantity
        updates.transaction_type = transaction.transaction_type
        updates.isin = transaction.isin
        break
      case 'fees':
        updates.title = transaction.title
        updates.subtitle = transaction.subtitle
        updates.amount_value = transaction.amount_value
        updates.amount_currency = transaction.amount_currency
        updates.fees = editingField.value
        updates.quantity = transaction.quantity
        updates.transaction_type = transaction.transaction_type
        updates.isin = transaction.isin
        break
      case 'quantity':
        updates.title = transaction.title
        updates.subtitle = transaction.subtitle
        updates.amount_value = transaction.amount_value
        updates.amount_currency = transaction.amount_currency
        updates.fees = transaction.fees
        updates.quantity = editingField.value ? parseFloat(editingField.value) : undefined
        updates.transaction_type = transaction.transaction_type
        updates.isin = transaction.isin
        break
      case 'isin':
        updates.title = transaction.title
        updates.subtitle = transaction.subtitle
        updates.amount_value = transaction.amount_value
        updates.amount_currency = transaction.amount_currency
        updates.fees = transaction.fees
        updates.quantity = transaction.quantity
        updates.transaction_type = transaction.transaction_type
        updates.isin = editingField.value || undefined
        break
    }

    await onUpdate(transaction.id, updates)
    setEditingField(null)
  }

  const handleFieldCancel = () => {
    setEditingField(null)
  }

  const EditIcon = ({ onClick }: { onClick: () => void }) => (
    <button
      onClick={onClick}
      className="ml-2 text-accent-primary hover:text-accent-hover transition-colors"
      title="Modifier"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-4 w-4"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
      </svg>
    </button>
  )

  const SaveIcon = ({ onClick }: { onClick: () => void }) => (
    <button
      onClick={onClick}
      className="ml-2 text-success hover:text-green-700 transition-colors"
      title="Enregistrer"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-4 w-4"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  )

  const CancelIcon = ({ onClick }: { onClick: () => void }) => (
    <button
      onClick={onClick}
      className="ml-2 text-error hover:text-red-700 transition-colors"
      title="Annuler"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-4 w-4"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  )

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
      {/* Desktop table view */}
      <div className="hidden lg:block overflow-x-auto">
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
              {onUpdate && (
                <th className="text-center py-3 px-4 text-text-muted font-medium">
                  Actions
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {transactions.map((transaction) => {
              const isEditing = editingRow === transaction.id
              return (
                <tr
                  key={transaction.id}
                  className={`border-b border-background-tertiary hover:bg-background-tertiary transition-colors ${
                    isEditing ? 'bg-blue-50' : ''
                  }`}
                >
                <td className="py-3 px-4 text-text-secondary">
                  {formatDate(transaction.timestamp)}
                </td>
                <td className="py-3 px-4">
                  <div>
                    <div className="font-medium flex items-center">
                      {editingField?.transactionId === transaction.id &&
                      editingField?.field === 'title' ? (
                        <>
                          <input
                            type="text"
                            value={editingField.value}
                            onChange={(e) =>
                              setEditingField({ ...editingField, value: e.target.value })
                            }
                            className="px-2 py-1 border border-accent-primary rounded"
                            autoFocus
                          />
                          <SaveIcon onClick={() => handleFieldSave(transaction)} />
                          <CancelIcon onClick={handleFieldCancel} />
                        </>
                      ) : (
                        <>
                          {transaction.isin ? (
                            <button
                              onClick={() => onAssetClick?.(transaction.isin!)}
                              className="text-accent-primary hover:text-accent-hover transition-colors text-left"
                            >
                              {transaction.title}
                            </button>
                          ) : (
                            <span
                              className={
                                transaction.transaction_type === 'buy' ||
                                transaction.transaction_type === 'sell'
                                  ? 'text-accent-primary'
                                  : 'text-text-primary'
                              }
                            >
                              {transaction.title}
                            </span>
                          )}
                          {isEditing && (
                            <EditIcon
                              onClick={() =>
                                handleFieldEdit(transaction.id, 'title', transaction.title)
                              }
                            />
                          )}
                        </>
                      )}
                    </div>
                    {transaction.subtitle && (
                      <div className="text-sm text-text-muted">
                        {transaction.subtitle}
                      </div>
                    )}
                    {transaction.isin && (
                      <div className="text-xs text-text-muted mt-1 flex items-center">
                        {editingField?.transactionId === transaction.id &&
                        editingField?.field === 'isin' ? (
                          <>
                            <input
                              type="text"
                              value={editingField.value}
                              onChange={(e) =>
                                setEditingField({ ...editingField, value: e.target.value })
                              }
                              className="px-2 py-1 border border-accent-primary rounded text-xs"
                              placeholder="Ex: US0378331005"
                              autoFocus
                            />
                            <SaveIcon onClick={() => handleFieldSave(transaction)} />
                            <CancelIcon onClick={handleFieldCancel} />
                          </>
                        ) : (
                          <>
                            {transaction.isin}
                            {isEditing && (
                              <EditIcon
                                onClick={() =>
                                  handleFieldEdit(
                                    transaction.id,
                                    'isin',
                                    transaction.isin || ''
                                  )
                                }
                              />
                            )}
                          </>
                        )}
                      </div>
                    )}
                    {!transaction.isin && isEditing && (
                      <div className="text-xs text-text-muted mt-1 flex items-center">
                        {editingField?.transactionId === transaction.id &&
                        editingField?.field === 'isin' ? (
                          <>
                            <input
                              type="text"
                              value={editingField.value}
                              onChange={(e) =>
                                setEditingField({ ...editingField, value: e.target.value })
                              }
                              className="px-2 py-1 border border-accent-primary rounded text-xs"
                              placeholder="Ex: US0378331005"
                              autoFocus
                            />
                            <SaveIcon onClick={() => handleFieldSave(transaction)} />
                            <CancelIcon onClick={handleFieldCancel} />
                          </>
                        ) : (
                          <button
                            onClick={() => handleFieldEdit(transaction.id, 'isin', '')}
                            className="text-accent-primary hover:text-accent-hover text-xs"
                          >
                            + Ajouter ISIN
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                </td>
                <td className="py-3 px-4">
                  {editingField?.transactionId === transaction.id &&
                  editingField?.field === 'type' ? (
                    <div className="flex items-center">
                      <select
                        value={editingField.value}
                        onChange={(e) =>
                          setEditingField({ ...editingField, value: e.target.value })
                        }
                        className="px-2 py-1 border border-accent-primary rounded"
                        autoFocus
                      >
                        {TRANSACTION_TYPES.map((type) => (
                          <option key={type.value} value={type.value}>
                            {type.label}
                          </option>
                        ))}
                      </select>
                      <SaveIcon onClick={() => handleFieldSave(transaction)} />
                      <CancelIcon onClick={handleFieldCancel} />
                    </div>
                  ) : (
                    <span className={`flex items-center ${getTypeColor(transaction.transaction_type)}`}>
                      {getTypeLabel(transaction.transaction_type)}
                      {isEditing && (
                        <EditIcon
                          onClick={() =>
                            handleFieldEdit(
                              transaction.id,
                              'type',
                              transaction.transaction_type
                            )
                          }
                        />
                      )}
                    </span>
                  )}
                </td>
                <td className="py-3 px-4 text-right text-text-secondary">
                  {editingField?.transactionId === transaction.id &&
                  editingField?.field === 'quantity' ? (
                    <div className="flex items-center justify-end">
                      <input
                        type="number"
                        step="0.000001"
                        value={editingField.value}
                        onChange={(e) =>
                          setEditingField({ ...editingField, value: e.target.value })
                        }
                        className="px-2 py-1 border border-accent-primary rounded w-24 text-right"
                        autoFocus
                      />
                      <SaveIcon onClick={() => handleFieldSave(transaction)} />
                      <CancelIcon onClick={handleFieldCancel} />
                    </div>
                  ) : (
                    <span className="flex items-center justify-end">
                      {transaction.quantity ? transaction.quantity.toFixed(4) : '-'}
                      {isEditing && (
                        <EditIcon
                          onClick={() =>
                            handleFieldEdit(
                              transaction.id,
                              'quantity',
                              transaction.quantity?.toString() || ''
                            )
                          }
                        />
                      )}
                    </span>
                  )}
                </td>
                <td
                  className={`py-3 px-4 text-right font-medium ${
                    transaction.amount_value >= 0 ? 'text-success' : 'text-error'
                  }`}
                >
                  {editingField?.transactionId === transaction.id &&
                  editingField?.field === 'amount' ? (
                    <div className="flex items-center justify-end">
                      <input
                        type="number"
                        step="0.01"
                        value={editingField.value}
                        onChange={(e) =>
                          setEditingField({ ...editingField, value: e.target.value })
                        }
                        className="px-2 py-1 border border-accent-primary rounded w-24 text-right"
                        autoFocus
                      />
                      <SaveIcon onClick={() => handleFieldSave(transaction)} />
                      <CancelIcon onClick={handleFieldCancel} />
                    </div>
                  ) : (
                    <span className="flex items-center justify-end">
                      {formatAmount(transaction.amount_value, transaction.amount_currency)}
                      {isEditing && (
                        <EditIcon
                          onClick={() =>
                            handleFieldEdit(
                              transaction.id,
                              'amount',
                              transaction.amount_value.toString()
                            )
                          }
                        />
                      )}
                    </span>
                  )}
                </td>
                <td className="py-3 px-4 text-right text-text-muted">
                  {editingField?.transactionId === transaction.id &&
                  editingField?.field === 'fees' ? (
                    <div className="flex items-center justify-end">
                      <input
                        type="number"
                        step="0.01"
                        value={editingField.value}
                        onChange={(e) =>
                          setEditingField({ ...editingField, value: e.target.value })
                        }
                        className="px-2 py-1 border border-accent-primary rounded w-24 text-right"
                        autoFocus
                      />
                      <SaveIcon onClick={() => handleFieldSave(transaction)} />
                      <CancelIcon onClick={handleFieldCancel} />
                    </div>
                  ) : (
                    <span className="flex items-center justify-end">
                      {formatFees(transaction.fees, transaction.amount_currency)}
                      {isEditing && (
                        <EditIcon
                          onClick={() =>
                            handleFieldEdit(
                              transaction.id,
                              'fees',
                              transaction.fees?.toString() || '0'
                            )
                          }
                        />
                      )}
                    </span>
                  )}
                </td>
                {onUpdate && (
                  <td className="py-3 px-4 text-center">
                    <button
                      onClick={() => handleEditClick(transaction.id)}
                      className={`transition-colors p-1 ${
                        isEditing
                          ? 'text-success hover:text-green-700'
                          : 'text-accent-primary hover:text-accent-hover'
                      }`}
                      title={isEditing ? 'Terminer' : 'Modifier'}
                    >
                      {isEditing ? (
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="h-5 w-5"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                        >
                          <path
                            fillRule="evenodd"
                            d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                            clipRule="evenodd"
                          />
                        </svg>
                      ) : (
                        <svg
                          xmlns="http://www.w3.org/2000/svg"
                          className="h-5 w-5"
                          viewBox="0 0 20 20"
                          fill="currentColor"
                        >
                          <path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z" />
                        </svg>
                      )}
                    </button>
                  </td>
                )}
              </tr>
            )})}
          </tbody>
        </table>
      </div>

      {/* Mobile card view */}
      <div className="lg:hidden space-y-4">
        {transactions.map((transaction) => (
          <div
            key={transaction.id}
            className="bg-background-tertiary rounded-lg p-4 space-y-3 hover:bg-opacity-80 transition-all duration-200"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                {transaction.isin ? (
                  <button
                    onClick={() => onAssetClick?.(transaction.isin!)}
                    className="text-accent-primary hover:text-accent-hover transition-colors text-left font-medium"
                  >
                    {transaction.title}
                  </button>
                ) : (
                  <div className="font-medium text-text-primary">{transaction.title}</div>
                )}
                {transaction.subtitle && (
                  <div className="text-sm text-text-muted mt-1">{transaction.subtitle}</div>
                )}
              </div>
              <span className={`text-xs px-2 py-1 rounded ${getTypeColor(transaction.transaction_type)} bg-background-secondary`}>
                {getTypeLabel(transaction.transaction_type)}
              </span>
            </div>

            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <div className="text-text-muted text-xs">Date</div>
                <div className="text-text-secondary">{formatDate(transaction.timestamp)}</div>
              </div>
              <div className="text-right">
                <div className="text-text-muted text-xs">Montant</div>
                <div className={`font-medium ${transaction.amount_value >= 0 ? 'text-success' : 'text-error'}`}>
                  {formatAmount(transaction.amount_value, transaction.amount_currency)}
                </div>
              </div>
              {transaction.quantity && (
                <div>
                  <div className="text-text-muted text-xs">Quantité</div>
                  <div className="text-text-secondary">{transaction.quantity.toFixed(4)}</div>
                </div>
              )}
              {(transaction.fees && parseFloat(transaction.fees.toString()) > 0) && (
                <div className="text-right">
                  <div className="text-text-muted text-xs">Frais</div>
                  <div className="text-text-muted">
                    {formatFees(transaction.fees, transaction.amount_currency)}
                  </div>
                </div>
              )}
            </div>

            {transaction.isin && (
              <div className="text-xs text-text-muted pt-2 border-t border-background-secondary">
                ISIN: {transaction.isin}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
