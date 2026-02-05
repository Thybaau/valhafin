import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import type { PerformancePoint } from '../../types'
import { useMemo } from 'react'

interface PerformanceChartProps {
  data: PerformancePoint[]
  isLoading?: boolean
}

interface TooltipProps {
  active?: boolean
  payload?: Array<{
    payload: {
      date: string
      value: number
      invested: number
    }
  }>
}

const CustomTooltip = ({ active, payload }: TooltipProps) => {
  const formatValue = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value)
  }

  if (active && payload && payload.length) {
    const data = payload[0].payload
    const gain = data.value - data.invested
    const gainPct = data.invested > 0 ? (gain / data.invested) * 100 : 0
    
    return (
      <div className="bg-background-secondary border border-background-tertiary rounded-lg p-3 shadow-lg">
        <p className="text-text-secondary text-sm mb-2">
          {new Date(data.date).toLocaleDateString('fr-FR', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          })}
        </p>
        <div className="space-y-1">
          <div>
            <p className="text-xs text-text-muted">Valeur actuelle</p>
            <p className="text-success font-semibold text-lg">
              {formatValue(data.value)}
            </p>
          </div>
          <div>
            <p className="text-xs text-text-muted">Montant investi</p>
            <p className="text-text-secondary text-sm">
              {formatValue(data.invested)}
            </p>
          </div>
          <div className="pt-1 border-t border-background-tertiary">
            <p className="text-xs text-text-muted">Gain/Perte</p>
            <p className={`text-sm font-semibold ${gain >= 0 ? 'text-success' : 'text-error'}`}>
              {formatValue(gain)} ({gainPct >= 0 ? '+' : ''}{gainPct.toFixed(2)}%)
            </p>
          </div>
        </div>
      </div>
    )
  }
  return null
}

export default function PerformanceChart({ data, isLoading }: PerformanceChartProps) {
  // No calculation needed - just display the value in EUR
  const chartData = useMemo(() => {
    if (!data || data.length === 0) return []
    
    // Filter out points with zero value (no positions yet)
    return data.filter(d => d.value > 0)
  }, [data])

  // Calculate min and max for Y axis
  const { minValue, maxValue } = useMemo(() => {
    if (chartData.length === 0) return { minValue: 0, maxValue: 1000 }
    
    const values = chartData.map(d => d.value)
    const min = Math.min(...values)
    const max = Math.max(...values)
    
    // Add padding to min/max
    const padding = (max - min) * 0.1
    
    return {
      minValue: Math.max(0, Math.floor(min - padding)),
      maxValue: Math.ceil(max + padding)
    }
  }, [chartData])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-primary"></div>
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className="text-center py-12 text-text-muted">
        Aucune donnée de performance disponible
      </div>
    )
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { month: 'short', year: '2-digit' })
  }

  const formatValue = (value: number) => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value)
  }

  // Line color - always green for assets value
  const lineColor = '#10B981'

  return (
    <div className="w-full">
      <ResponsiveContainer width="100%" height={400} className="min-h-[350px]">
        <LineChart 
          data={chartData} 
          margin={{ 
            top: 20, 
            right: 30, 
            left: 10, 
            bottom: 20 
          }}
        >
          <defs>
            <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#10B981" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#10B981" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid 
            strokeDasharray="3 3" 
            stroke="#374151" 
            opacity={0.2}
            vertical={false}
          />
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            stroke="#6B7280"
            style={{ fontSize: '12px' }}
            tickLine={false}
            axisLine={false}
            dy={10}
          />
          <YAxis
            domain={[minValue, maxValue]}
            tickFormatter={formatValue}
            stroke="#6B7280"
            style={{ fontSize: '12px' }}
            tickLine={false}
            axisLine={false}
            width={80}
            orientation="right"
          />
          <Tooltip content={<CustomTooltip />} cursor={{ stroke: '#6B7280', strokeWidth: 1, strokeDasharray: '5 5' }} />
          
          {/* Invested line (dashed, less visible) */}
          <Line
            type="monotone"
            dataKey="invested"
            stroke="#6B7280"
            strokeWidth={2}
            strokeDasharray="5 5"
            dot={false}
            opacity={0.5}
          />
          
          {/* Value line (solid, green) */}
          <Line
            type="monotone"
            dataKey="value"
            stroke={lineColor}
            strokeWidth={3}
            dot={false}
            activeDot={{ r: 6, fill: lineColor, strokeWidth: 2, stroke: '#fff' }}
            fill="url(#colorValue)"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
