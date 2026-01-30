import apiClient from './api'
import type { AssetPrice } from '../types'

export const assetsApi = {
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
}
