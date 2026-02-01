import apiClient from './api'
import type { AssetPrice } from '../types'

export const assetsApi = {
  // Récupérer tous les actifs avec positions
  getAssets: async (): Promise<any[]> => {
    const response = await apiClient.get('/assets')
    return response.data
  },

  // Récupérer le prix actuel d'un actif
  getPrice: async (isin: string): Promise<AssetPrice> => {
    const response = await apiClient.get(`/assets/${isin}/price`)
    return response.data
  },

  // Récupérer l'historique des prix d'un actif
  getPriceHistory: async (isin: string): Promise<AssetPrice[]> => {
    const response = await apiClient.get(`/assets/${isin}/history`)
    return response.data
  },

  // Récupérer l'historique des prix avec dates
  getAssetPriceHistory: async (
    isin: string,
    startDate: string,
    endDate: string
  ): Promise<AssetPrice[]> => {
    const response = await apiClient.get(`/assets/${isin}/history`, {
      params: { start_date: startDate, end_date: endDate },
    })
    return response.data
  },
}
