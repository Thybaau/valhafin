import { useQuery } from '@tanstack/react-query'
import { assetsApi } from '../services'
import type { PriceHistoryFilters } from '../services'

// Query keys
export const assetKeys = {
  all: ['assets'] as const,
  price: (isin: string) => ['assets', isin, 'price'] as const,
  history: (isin: string, filters?: PriceHistoryFilters) =>
    ['assets', isin, 'history', filters] as const,
}

// Hook pour récupérer le prix actuel d'un actif
export function useAssetPrice(isin: string) {
  return useQuery({
    queryKey: assetKeys.price(isin),
    queryFn: () => assetsApi.getCurrentPrice(isin),
    enabled: !!isin,
    staleTime: 60 * 1000, // 1 minute
  })
}

// Hook pour récupérer l'historique des prix d'un actif
export function useAssetPriceHistory(isin: string, filters?: PriceHistoryFilters) {
  return useQuery({
    queryKey: assetKeys.history(isin, filters),
    queryFn: () => assetsApi.getPriceHistory(isin, filters),
    enabled: !!isin,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })
}
