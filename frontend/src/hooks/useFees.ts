import { useQuery } from '@tanstack/react-query'
import { feesApi } from '../services'
import type { FeesFilters } from '../services'

// Query keys
export const feesKeys = {
  all: ['fees'] as const,
  global: (filters?: FeesFilters) => ['fees', 'global', filters] as const,
  byAccount: (accountId: string, filters?: FeesFilters) =>
    ['fees', 'account', accountId, filters] as const,
}

// Hook pour récupérer les métriques de frais globales
export function useGlobalFees(filters?: FeesFilters) {
  return useQuery({
    queryKey: feesKeys.global(filters),
    queryFn: () => feesApi.getGlobal(filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

// Hook pour récupérer les métriques de frais d'un compte
export function useAccountFees(accountId: string, filters?: FeesFilters) {
  return useQuery({
    queryKey: feesKeys.byAccount(accountId, filters),
    queryFn: () => feesApi.getByAccount(accountId, filters),
    enabled: !!accountId,
    staleTime: 5 * 60 * 1000,
  })
}
