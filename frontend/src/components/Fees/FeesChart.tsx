import { XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Area, AreaChart } from 'recharts'

interface FeesChartProps {
  data: Array<{
    date: string
    fees: number
  }>
  isLoading?: boolean
}

export default function FeesChart({ data, isLoading }: FeesChartProps) {
  if (isLoading) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Évolution des Frais</h2>
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-primary"></div>
        </div>
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className="card">
        <h2 className="text-xl font-semibold mb-4">Évolution des Frais</h2>
        <div className="text-center py-12 text-text-muted">
          Aucune donnée de frais disponible
        </div>
      </div>
    )
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('fr-FR', { month: 'short', day: 'numeric' })
  }

  const formatCurrency = (value: number) => {
    // Handle NaN, null, undefined, or invalid values
    if (!value || isNaN(value)) {
      return '0,00 €'
    }
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: 'EUR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  }

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload
      return (
        <div className="bg-background-secondary border border-background-tertiary rounded-lg p-3 shadow-lg">
          <p className="text-text-secondary text-sm mb-1">
            {new Date(data.date).toLocaleDateString('fr-FR', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })}
          </p>
          <p className="text-warning font-semibold">
            {formatCurrency(data.fees)}
          </p>
        </div>
      )
    }
    return null
  }

  return (
    <div className="card">
      <h2 className="text-xl font-semibold mb-4">Évolution des Frais</h2>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <defs>
            <linearGradient id="colorFees" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#f59e0b" stopOpacity={0.1} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" opacity={0.3} />
          <XAxis
            dataKey="date"
            tickFormatter={formatDate}
            stroke="#9CA3AF"
            style={{ fontSize: '12px' }}
          />
          <YAxis
            tickFormatter={formatCurrency}
            stroke="#9CA3AF"
            style={{ fontSize: '12px' }}
          />
          <Tooltip content={<CustomTooltip />} />
          <Area
            type="monotone"
            dataKey="fees"
            stroke="#f59e0b"
            strokeWidth={2}
            fill="url(#colorFees)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
