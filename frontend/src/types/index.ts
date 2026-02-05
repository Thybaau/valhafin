// Common types for the application

export interface Account {
  id: string
  name: string
  platform: 'traderepublic' | 'binance' | 'boursedirect'
  created_at: string
  updated_at: string
  last_sync: string | null
}

export interface Transaction {
  id: string
  account_id: string
  timestamp: string
  title: string
  subtitle?: string
  isin?: string
  quantity?: number
  amount_value: number
  amount_currency: string
  fees: string | number // Can be string from API
  transaction_type: string
  status: string
}

export interface Asset {
  isin: string
  name: string
  symbol: string
  type: 'stock' | 'etf' | 'crypto'
  currency: string
  last_updated: string
}

export interface AssetPrice {
  id: number
  isin: string
  price: number
  currency: string
  timestamp: string
}

export interface AssetPosition {
  isin: string
  name: string
  symbol?: string
  symbol_verified: boolean
  quantity: number
  average_buy_price: number
  current_price: number
  current_value: number
  total_invested: number
  unrealized_gain: number
  unrealized_gain_pct: number
  total_fees: number
  currency: string
  purchases?: Array<{
    date: string
    quantity: number
    price: number
  }>
}

export interface Performance {
  total_value: number
  total_invested: number
  cash_balance: number
  total_fees: number
  realized_gains: number
  unrealized_gains: number
  performance_pct: number
  time_series: PerformancePoint[]
}

export interface PerformancePoint {
  date: string
  value: number
  invested: number
}

export interface ErrorResponse {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}
