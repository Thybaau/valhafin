import apiClient from './api'
import type { Performance } from '../types'

export type Period = '1m' | '3m' | '1y' | 'all'

export interface AssetPerformance {
  isin: string
  name: string
  symbol: string
  current_price: number
  quantity: number
  total_value: number
  total_invested: number
  total_fees: number
  realized_gains: number
  unrealized_gains: number
  performance_pct: number
  time_series: Array<{
    date: string
    value: number
  }>
}

export const performanceApi = {
  // Récupérer la performance d'un compte
  getByAccount: async (accountId: string, period: Period = '1m'): Promise<Performance> => {
    const response = await apiClient.get(`/accounts/${accountId}/performance`, {
      params: { period },
    })
    return response.data
  },

  // Récupérer la performance globale
  getGlobal: async (period: Period = '1m'): Promise<Performance> => {
    const response = await apiClient.get('/performance', {
      params: { period },
    })
    return response.data
  },

  // Récupérer la performance d'un actif
  getByAsset: async (isin: string, period: Period = '1m'): Promise<AssetPerformance> => {
    const response = await apiClient.get(`/assets/${isin}/performance`, {
      params: { period },
    })
    return response.data
  },
}
