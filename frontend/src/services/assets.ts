import apiClient from './api'
import type { AssetPrice } from '../types'

export interface PriceHistoryFilters {
  start_date?: string
  end_date?: string
}

export const assetsApi = {
  // Récupérer le prix actuel d'un actif
  getCurrentPrice: async (isin: string): Promise<AssetPrice> => {
    const response = await apiClient.get(`/assets/${isin}/price`)
    return response.data
  },

  // Récupérer l'historique des prix d'un actif
  getPriceHistory: async (
    isin: string,
    filters?: PriceHistoryFilters
  ): Promise<AssetPrice[]> => {
    const response = await apiClient.get(`/assets/${isin}/history`, {
      params: filters,
    })
    return response.data
  },
}
