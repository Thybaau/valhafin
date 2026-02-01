import { useQuery } from '@tanstack/react-query';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceDot,
} from 'recharts';
import { assetsApi } from '../../services';

interface AssetPosition {
  isin: string;
  name: string;
  quantity: number;
  average_buy_price: number;
  current_price: number;
  current_value: number;
  total_invested: number;
  unrealized_gain: number;
  unrealized_gain_pct: number;
  currency: string;
  purchases: Array<{
    date: string;
    quantity: number;
    price: number;
  }>;
}

interface Props {
  asset: AssetPosition;
  onClose: () => void;
}

interface PriceHistory {
  id: number;
  isin: string;
  price: number;
  currency: string;
  timestamp: string;
}

export default function AssetDetailModal({ asset, onClose }: Props) {
  // Get price history
  const { data: priceHistory } = useQuery<PriceHistory[]>({
    queryKey: ['assetHistory', asset.isin],
    queryFn: () => {
      const startDate = new Date();
      startDate.setFullYear(startDate.getFullYear() - 2); // Last 2 years
      const endDate = new Date();
      return assetsApi.getAssetPriceHistory(
        asset.isin,
        startDate.toISOString().split('T')[0],
        endDate.toISOString().split('T')[0]
      );
    },
  });

  const formatCurrency = (value: number, currency: string = 'EUR') => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: currency,
    }).format(value);
  };

  // Prepare chart data
  const chartData = priceHistory?.map((point) => ({
    date: new Date(point.timestamp).toLocaleDateString('fr-FR'),
    timestamp: point.timestamp,
    price: point.price,
  })) || [];

  // Create purchase markers
  const purchaseMarkers = asset.purchases.map((purchase) => {
    // Find closest price point to purchase date
    const purchaseDate = new Date(purchase.date);
    const closestPoint = chartData.find((point) => {
      const pointDate = new Date(point.timestamp);
      return pointDate.toDateString() === purchaseDate.toDateString();
    });

    return {
      date: new Date(purchase.date).toLocaleDateString('fr-FR'),
      price: closestPoint?.price || purchase.price,
      quantity: purchase.quantity,
    };
  });

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex justify-between items-start">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">{asset.name}</h2>
            <p className="text-sm text-gray-500 mt-1">{asset.isin}</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Summary Cards */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="text-sm text-gray-500">Quantité</div>
              <div className="text-lg font-semibold text-gray-900 mt-1">
                {asset.quantity.toFixed(6)}
              </div>
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="text-sm text-gray-500">Prix Moyen</div>
              <div className="text-lg font-semibold text-gray-900 mt-1">
                {formatCurrency(asset.average_buy_price, asset.currency)}
              </div>
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="text-sm text-gray-500">Prix Actuel</div>
              <div className="text-lg font-semibold text-gray-900 mt-1">
                {formatCurrency(asset.current_price, asset.currency)}
              </div>
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <div className="text-sm text-gray-500">Gain/Perte</div>
              <div
                className={`text-lg font-semibold mt-1 ${
                  asset.unrealized_gain >= 0
                    ? 'text-green-600'
                    : 'text-red-600'
                }`}
              >
                {formatCurrency(asset.unrealized_gain, asset.currency)}
                <span className="text-sm ml-2">
                  ({asset.unrealized_gain_pct >= 0 ? '+' : ''}
                  {asset.unrealized_gain_pct.toFixed(2)}%)
                </span>
              </div>
            </div>
          </div>

          {/* Price Chart */}
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              Évolution du Prix
            </h3>
            {chartData.length > 0 ? (
              <ResponsiveContainer width="100%" height={400}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis
                    dataKey="date"
                    tick={{ fontSize: 12 }}
                    angle={-45}
                    textAnchor="end"
                    height={80}
                  />
                  <YAxis
                    tick={{ fontSize: 12 }}
                    domain={['auto', 'auto']}
                    tickFormatter={(value) =>
                      `${value.toFixed(2)} ${asset.currency}`
                    }
                  />
                  <Tooltip
                    formatter={(value: number | undefined) =>
                      value !== undefined ? formatCurrency(value, asset.currency) : ''
                    }
                    labelStyle={{ color: '#000' }}
                  />
                  <Line
                    type="monotone"
                    dataKey="price"
                    stroke="#3b82f6"
                    strokeWidth={2}
                    dot={false}
                  />
                  {/* Purchase markers */}
                  {purchaseMarkers.map((marker, index) => (
                    <ReferenceDot
                      key={index}
                      x={marker.date}
                      y={marker.price}
                      r={6}
                      fill="#10b981"
                      stroke="#fff"
                      strokeWidth={2}
                    />
                  ))}
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="text-center py-12 text-gray-500">
                Aucune donnée de prix disponible
              </div>
            )}
            <div className="mt-4 flex items-center gap-4 text-sm text-gray-600">
              <div className="flex items-center gap-2">
                <div className="w-4 h-0.5 bg-blue-500"></div>
                <span>Prix du marché</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-full bg-green-500 border-2 border-white"></div>
                <span>Achats</span>
              </div>
            </div>
          </div>

          {/* Purchase History */}
          <div className="bg-white border border-gray-200 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">
              Historique d'Achats
            </h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Date
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Quantité
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Prix
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                      Total
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {asset.purchases.map((purchase, index) => (
                    <tr key={index}>
                      <td className="px-4 py-3 text-sm text-gray-900">
                        {new Date(purchase.date).toLocaleDateString('fr-FR')}
                      </td>
                      <td className="px-4 py-3 text-sm text-right text-gray-900">
                        {purchase.quantity.toFixed(6)}
                      </td>
                      <td className="px-4 py-3 text-sm text-right text-gray-900">
                        {formatCurrency(purchase.price, asset.currency)}
                      </td>
                      <td className="px-4 py-3 text-sm text-right font-medium text-gray-900">
                        {formatCurrency(
                          purchase.quantity * purchase.price,
                          asset.currency
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
