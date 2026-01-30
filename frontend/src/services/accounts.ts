import apiClient from './api'
import type { Account } from '../types'

export interface CreateAccountRequest {
  name: string
  platform: 'traderepublic' | 'binance' | 'boursedirect'
  credentials: {
    phone_number?: string
    pin?: string
    api_key?: string
    api_secret?: string
    username?: string
    password?: string
  }
}

export interface SyncResponse {
  success: boolean
  transactions_added: number
  message: string
}

export interface InitSyncResponse {
  requires_two_factor: boolean
  process_id?: string
  message: string
}

export interface CompleteSyncRequest {
  process_id: string
  code: string
}

export const accountsApi = {
  // Récupérer tous les comptes
  getAll: async (): Promise<Account[]> => {
    const response = await apiClient.get('/accounts')
    return response.data
  },

  // Récupérer un compte par ID
  getById: async (id: string): Promise<Account> => {
    const response = await apiClient.get(`/accounts/${id}`)
    return response.data
  },

  // Créer un nouveau compte
  create: async (data: CreateAccountRequest): Promise<Account> => {
    const response = await apiClient.post('/accounts', data)
    return response.data
  },

  // Supprimer un compte
  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/accounts/${id}`)
  },

  // Synchroniser un compte (pour Binance et Bourse Direct)
  sync: async (id: string): Promise<SyncResponse> => {
    const response = await apiClient.post(`/accounts/${id}/sync`)
    return response.data
  },

  // Initier la synchronisation (pour Trade Republic - déclenche 2FA)
  initSync: async (id: string): Promise<InitSyncResponse> => {
    const response = await apiClient.post(`/accounts/${id}/sync/init`)
    return response.data
  },

  // Compléter la synchronisation avec le code 2FA
  completeSync: async (id: string, data: CompleteSyncRequest): Promise<SyncResponse> => {
    const response = await apiClient.post(`/accounts/${id}/sync/complete`, data)
    return response.data
  },
}
