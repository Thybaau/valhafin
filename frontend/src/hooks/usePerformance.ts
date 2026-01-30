import { useQuery } from '@tanstack/react-query'
import { performanceApi } from '../services'
import type { Period } from '../services'

// Query keys
export const performanceKeys = {
  all: ['performance'] as const,
  global: (period: Period) => ['performance', 'global', period] as const,
  byAccount: (accountId: string, period: Period) =>
    ['performance', 'account', accountId, period] as const,
  byAsset: (isin: string, period: Period) =>
    ['performance', 'asset', isin, period] as const,
}

// Hook pour récupérer la performance globale
export function useGlobalPerformance(period: Period = '1m') {
  return useQuery({
    queryKey: performanceKeys.global(period),
    queryFn: () => performanceApi.getGlobal(period),
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

// Hook pour récupérer la performance d'un compte
export function useAccountPerformance(accountId: string, period: Period = '1m') {
  return useQuery({
    queryKey: performanceKeys.byAccount(accountId, period),
    queryFn: () => performanceApi.getByAccount(accountId, period),
    enabled: !!accountId,
    staleTime: 5 * 60 * 1000,
  })
}

// Hook pour récupérer la performance d'un actif
export function useAssetPerformance(isin: string, period: Period = '1m') {
  return useQuery({
    queryKey: performanceKeys.byAsset(isin, period),
    queryFn: () => performanceApi.getByAsset(isin, period),
    enabled: !!isin,
    staleTime: 5 * 60 * 1000,
  })
}
