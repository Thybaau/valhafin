import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { transactionsApi } from '../services'
import type { TransactionFilters } from '../services'
import type { Transaction } from '../types'

// Query keys
export const transactionKeys = {
  all: ['transactions'] as const,
  list: (filters?: TransactionFilters) => ['transactions', 'list', filters] as const,
  byAccount: (accountId: string, filters?: TransactionFilters) =>
    ['transactions', 'account', accountId, filters] as const,
}

// Hook pour récupérer toutes les transactions
export function useTransactions(filters?: TransactionFilters) {
  return useQuery({
    queryKey: transactionKeys.list(filters),
    queryFn: () => transactionsApi.getAll(filters),
    staleTime: 30 * 1000, // 30 seconds - shorter to ensure fresh data on page change
  })
}

// Hook pour récupérer les transactions d'un compte
export function useAccountTransactions(accountId: string, filters?: TransactionFilters) {
  return useQuery({
    queryKey: transactionKeys.byAccount(accountId, filters),
    queryFn: () => transactionsApi.getByAccount(accountId, filters),
    enabled: !!accountId,
    staleTime: 2 * 60 * 1000,
  })
}

// Hook pour mettre à jour une transaction
export function useUpdateTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, transaction }: { id: string; transaction: Partial<Transaction> }) =>
      transactionsApi.update(id, transaction),
    onSuccess: () => {
      // Invalider toutes les transactions et performances
      queryClient.invalidateQueries({ queryKey: transactionKeys.all })
      queryClient.invalidateQueries({ queryKey: ['performance'] })
      queryClient.invalidateQueries({ queryKey: ['fees'] })
    },
  })
}

// Hook pour importer un CSV
export function useImportCSV() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ accountId, file }: { accountId: string; file: File }) =>
      transactionsApi.importCSV(accountId, file),
    onSuccess: () => {
      // Invalider toutes les transactions et performances
      queryClient.invalidateQueries({ queryKey: transactionKeys.all })
      queryClient.invalidateQueries({ queryKey: ['performance'] })
      queryClient.invalidateQueries({ queryKey: ['fees'] })
    },
  })
}
