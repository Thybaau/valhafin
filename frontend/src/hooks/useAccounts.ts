import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { accountsApi } from '../services'
import type { CreateAccountRequest } from '../services'

// Query keys
export const accountKeys = {
  all: ['accounts'] as const,
  detail: (id: string) => ['accounts', id] as const,
}

// Hook pour récupérer tous les comptes
export function useAccounts() {
  return useQuery({
    queryKey: accountKeys.all,
    queryFn: accountsApi.getAll,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

// Hook pour récupérer un compte par ID
export function useAccount(id: string) {
  return useQuery({
    queryKey: accountKeys.detail(id),
    queryFn: () => accountsApi.getById(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  })
}

// Hook pour créer un compte
export function useCreateAccount() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateAccountRequest) => accountsApi.create(data),
    onSuccess: () => {
      // Invalider la liste des comptes pour forcer un refetch
      queryClient.invalidateQueries({ queryKey: accountKeys.all })
    },
  })
}

// Hook pour supprimer un compte
export function useDeleteAccount() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => accountsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: accountKeys.all })
    },
  })
}

// Hook pour synchroniser un compte
export function useSyncAccount() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => accountsApi.sync(id),
    onSuccess: (_, id) => {
      // Invalider les données du compte et les transactions
      queryClient.invalidateQueries({ queryKey: accountKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['assets'] })
      queryClient.invalidateQueries({ queryKey: ['performance'] })
    },
  })
}

// Hook pour initier la synchronisation (Trade Republic 2FA)
export function useInitSync() {
  return useMutation({
    mutationFn: (id: string) => accountsApi.initSync(id),
  })
}

// Hook pour compléter la synchronisation avec le code 2FA
export function useCompleteSync() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { process_id: string; code: string } }) =>
      accountsApi.completeSync(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: accountKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['assets'] })
      queryClient.invalidateQueries({ queryKey: ['performance'] })
    },
  })
}
