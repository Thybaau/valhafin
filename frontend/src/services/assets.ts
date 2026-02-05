import apiClient from './api'
import type { AssetPrice, AssetPosition } from '../types'

export const assetsApi = {
  // Récupérer tous les actifs avec positions
  getAssets: async (): Promise<AssetPosition[]> => {
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

  // Forcer le rechargement des prix d'un actif
  refreshAssetPrices: async (isin: string): Promise<void> => {
    await apiClient.post(`/assets/${isin}/price/refresh`)
  },
}
