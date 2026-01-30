import apiClient from './api'
import type { Transaction } from '../types'

export interface TransactionFilters {
  start_date?: string
  end_date?: string
  asset?: string
  type?: string
  page?: number
  limit?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

export interface TransactionListResponse {
  transactions: Transaction[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface ImportCSVResponse {
  imported: number
  ignored: number
  errors: Array<{
    line: number
    message: string
  }>
}

export const transactionsApi = {
  // Récupérer les transactions d'un compte
  getByAccount: async (
    accountId: string,
    filters?: TransactionFilters
  ): Promise<TransactionListResponse> => {
    const response = await apiClient.get(`/accounts/${accountId}/transactions`, {
      params: filters,
    })
    return response.data
  },

  // Récupérer toutes les transactions
  getAll: async (filters?: TransactionFilters): Promise<TransactionListResponse> => {
    const response = await apiClient.get('/transactions', {
      params: filters,
    })
    return response.data
  },

  // Importer des transactions depuis un CSV
  importCSV: async (accountId: string, file: File): Promise<ImportCSVResponse> => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('account_id', accountId)

    const response = await apiClient.post('/transactions/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return response.data
  },
}
