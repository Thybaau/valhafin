import { useQuery } from '@tanstack/react-query'
import { performanceApi, assetsApi } from '../services'

// Query keys
export const performanceKeys = {
  all: ['performance'] as const,
  global: (period?: string) => ['performance', 'global', period] as const,
  account: (accountId: string, period?: string) =>
    ['performance', 'account', accountId, period] as const,
  asset: (isin: string, period?: string) => ['performance', 'asset', isin, period] as const,
}

export const assetKeys = {
  all: ['assets'] as const,
  price: (isin: string) => ['assets', 'price', isin] as const,
  history: (isin: string) => ['assets', 'history', isin] as const,
}

// Hook pour récupérer la performance globale
export function useGlobalPerformance(period?: string) {
  return useQuery({
    queryKey: performanceKeys.global(period),
    queryFn: () => performanceApi.getGlobal(period),
    staleTime: 2 * 60 * 1000, // 2 minutes
  })
}

// Hook pour récupérer la performance d'un compte
export function useAccountPerformance(accountId: string, period?: string) {
  return useQuery({
    queryKey: performanceKeys.account(accountId, period),
    queryFn: () => performanceApi.getByAccount(accountId, period),
    enabled: !!accountId,
    staleTime: 2 * 60 * 1000,
  })
}

// Hook pour récupérer la performance d'un actif
export function useAssetPerformance(isin: string, enabled: boolean = true) {
  return useQuery({
    queryKey: performanceKeys.asset(isin),
    queryFn: () => performanceApi.getByAsset(isin),
    enabled: enabled && !!isin,
    staleTime: 2 * 60 * 1000,
  })
}

// Hook pour récupérer le prix actuel d'un actif
export function useAssetPrice(isin: string, enabled: boolean = true) {
  return useQuery({
    queryKey: assetKeys.price(isin),
    queryFn: () => assetsApi.getPrice(isin),
    enabled: enabled && !!isin,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}

// Hook pour récupérer l'historique des prix d'un actif
export function useAssetPriceHistory(isin: string) {
  return useQuery({
    queryKey: assetKeys.history(isin),
    queryFn: () => assetsApi.getPriceHistory(isin),
    enabled: !!isin,
    staleTime: 5 * 60 * 1000,
  })
}
