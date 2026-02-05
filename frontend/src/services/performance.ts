import apiClient from './api'
import type { Performance } from '../types'

export const performanceApi = {
  // Récupérer la performance globale
  getGlobal: async (period?: string): Promise<Performance> => {
    const response = await apiClient.get('/performance', {
      params: period ? { period } : undefined,
    })
    return response.data
  },

  // Récupérer la performance d'un compte
  getByAccount: async (accountId: string, period?: string): Promise<Performance> => {
    const response = await apiClient.get(`/accounts/${accountId}/performance`, {
      params: period ? { period } : undefined,
    })
    return response.data
  },

  // Récupérer la performance d'un actif
  getByAsset: async (isin: string, period?: string): Promise<Performance> => {
    const response = await apiClient.get(`/assets/${isin}/performance`, {
      params: period ? { period } : undefined,
    })
    return response.data
  },
}
