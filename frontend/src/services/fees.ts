import apiClient from './api'

type Period = '1m' | '3m' | '1y' | 'all'

export interface FeesMetrics {
  total_fees: number
  average_fees: number
  fees_by_type: {
    buy: number
    sell: number
    transfer: number
    other: number
  }
  time_series: Array<{
    date: string
    fees: number
  }>
}

export interface FeesFilters {
  start_date?: string
  end_date?: string
  period?: Period
}

export const feesApi = {
  // Récupérer les métriques de frais d'un compte
  getByAccount: async (accountId: string, filters?: FeesFilters): Promise<FeesMetrics> => {
    const response = await apiClient.get(`/accounts/${accountId}/fees`, {
      params: filters,
    })
    return response.data
  },

  // Récupérer les métriques de frais globales
  getGlobal: async (filters?: FeesFilters): Promise<FeesMetrics> => {
    const response = await apiClient.get('/fees', {
      params: filters,
    })
    return response.data
  },
}
