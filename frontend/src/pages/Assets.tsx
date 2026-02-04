import { useState, useEffect } from 'react';
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
import { Search, AlertCircle } from 'lucide-react';
import { assetsApi } from '../services';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorMessage from '../components/common/ErrorMessage';
import Header from '../components/Layout/Header';
import SymbolSearchModal from '../components/Assets/SymbolSearchModal';

interface AssetPosition {
  isin: string;
  name: string;
  symbol?: string;
  symbol_verified: boolean;
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

interface PriceHistory {
  id: number;
  isin: string;
  price: number;
  currency: string;
  timestamp: string;
}

export default function Assets() {
  const [selectedAsset, setSelectedAsset] = useState<AssetPosition | null>(null);
  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const [assetToSearch, setAssetToSearch] = useState<AssetPosition | null>(null);

  const { data: assets, isLoading, error, refetch } = useQuery<AssetPosition[]>({
    queryKey: ['assets'],
    queryFn: assetsApi.getAssets,
  });

  // Select first asset by default when assets are loaded
  useEffect(() => {
    if (assets && assets.length > 0 && !selectedAsset) {
      setSelectedAsset(assets[0]);
    }
  }, [assets, selectedAsset]);

  // Get price history for selected asset
  const { data: priceHistory } = useQuery<PriceHistory[]>({
    queryKey: ['assetHistory', selectedAsset?.isin],
    queryFn: () => {
      if (!selectedAsset) return Promise.resolve([]);
      const startDate = new Date();
      startDate.setFullYear(startDate.getFullYear() - 2);
      const endDate = new Date();
      return assetsApi.getAssetPriceHistory(
        selectedAsset.isin,
        startDate.toISOString().split('T')[0],
        endDate.toISOString().split('T')[0]
      );
    },
    enabled: !!selectedAsset,
  });

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage message="Erreur lors du chargement des actifs" />;
  if (!assets || assets.length === 0) {
    return (
      <div className="min-h-screen bg-background-primary p-6">
        <Header title="Mes Actifs" />
        <div className="text-center py-12">
          <p className="text-text-muted">Aucun actif trouvé</p>
        </div>
      </div>
    );
  }

  const formatCurrency = (value: number, currency: string = 'EUR') => {
    return new Intl.NumberFormat('fr-FR', {
      style: 'currency',
      currency: currency,
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
  };

  const handleOpenSearchModal = (asset: AssetPosition) => {
    setAssetToSearch(asset);
    setSearchModalOpen(true);
  };

  const handleSymbolSelected = () => {
    refetch();
  };

  // Prepare chart data
  const chartData = priceHistory?.map((point) => ({
    date: new Date(point.timestamp).toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: '2-digit',
    }),
    timestamp: point.timestamp,
    price: point.price,
  })) || [];

  // Create purchase markers
  const purchaseMarkers = selectedAsset?.purchases.map((purchase) => {
    const purchaseDate = new Date(purchase.date);
    const closestPoint = chartData.find((point) => {
      const pointDate = new Date(point.timestamp);
      return pointDate.toDateString() === purchaseDate.toDateString();
    });

    return {
      date: new Date(purchase.date).toLocaleDateString('fr-FR', {
        day: '2-digit',
        month: '2-digit',
      }),
      price: closestPoint?.price || purchase.price,
      quantity: purchase.quantity,
    };
  }) || [];

  return (
    <div className="min-h-screen bg-background-primary p-6">
      <Header title="Mes Actifs" />

      <SymbolSearchModal
        isOpen={searchModalOpen}
        onClose={() => setSearchModalOpen(false)}
        asset={assetToSearch || { isin: '', name: '', symbol_verified: false }}
        onSymbolSelected={handleSymbolSelected}
      />

      <div className="space-y-6 mt-6">
        {/* Chart Section */}
        {selectedAsset && (
          <div className="bg-background-secondary rounded-lg p-6 border border-background-tertiary">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="text-xl font-semibold text-text-primary">
                  {selectedAsset.name}
                </h2>
                <p className="text-sm text-text-muted">{selectedAsset.isin}</p>
              </div>
              {!selectedAsset.symbol_verified && (
                <button
                  onClick={() => handleOpenSearchModal(selectedAsset)}
                  className="flex items-center gap-2 px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700"
                >
                  <AlertCircle className="w-4 h-4" />
                  <span>Verify Symbol</span>
                </button>
              )}
            </div>

            {/* Summary Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
              <div className="bg-background-tertiary rounded-lg p-4">
                <div className="text-sm text-text-muted">Quantité</div>
                <div className="text-lg font-semibold text-text-primary mt-1">
                  {selectedAsset.quantity.toFixed(6)}
                </div>
              </div>
              <div className="bg-background-tertiary rounded-lg p-4">
                <div className="text-sm text-text-muted">Prix Moyen</div>
                <div className="text-lg font-semibold text-text-primary mt-1">
                  {formatCurrency(selectedAsset.average_buy_price, selectedAsset.currency)}
                </div>
              </div>
              <div className="bg-background-tertiary rounded-lg p-4">
                <div className="text-sm text-text-muted">Prix Actuel</div>
                <div className="text-lg font-semibold text-text-primary mt-1">
                  {formatCurrency(selectedAsset.current_price, selectedAsset.currency)}
                </div>
              </div>
              <div className="bg-background-tertiary rounded-lg p-4">
                <div className="text-sm text-text-muted">Gain/Perte</div>
                <div
                  className={`text-lg font-semibold mt-1 ${
                    selectedAsset.unrealized_gain >= 0 ? 'text-success' : 'text-error'
                  }`}
                >
                  {formatCurrency(selectedAsset.unrealized_gain, selectedAsset.currency)}
                  <span className="text-sm ml-2">
                    ({formatPercent(selectedAsset.unrealized_gain_pct)})
                  </span>
                </div>
              </div>
            </div>

            {/* Price Chart */}
            {chartData.length > 0 ? (
              <div className="bg-background-tertiary rounded-lg p-4">
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                    <XAxis
                      dataKey="date"
                      tick={{ fontSize: 12, fill: '#9CA3AF' }}
                      stroke="#4B5563"
                    />
                    <YAxis
                      tick={{ fontSize: 12, fill: '#9CA3AF' }}
                      stroke="#4B5563"
                      domain={['auto', 'auto']}
                      tickFormatter={(value) => `${value.toFixed(2)}`}
                    />
                    <Tooltip
                      formatter={(value: number | undefined) =>
                        value !== undefined
                          ? formatCurrency(value, selectedAsset.currency)
                          : ''
                      }
                      contentStyle={{
                        backgroundColor: '#1F2937',
                        border: '1px solid #374151',
                        borderRadius: '0.5rem',
                        color: '#F3F4F6',
                      }}
                      labelStyle={{ color: '#F3F4F6' }}
                    />
                    <Line
                      type="monotone"
                      dataKey="price"
                      stroke="#3B82F6"
                      strokeWidth={2}
                      dot={false}
                    />
                    {purchaseMarkers.map((marker, index) => (
                      <ReferenceDot
                        key={index}
                        x={marker.date}
                        y={marker.price}
                        r={5}
                        fill="#10B981"
                        stroke="#1F2937"
                        strokeWidth={2}
                      />
                    ))}
                  </LineChart>
                </ResponsiveContainer>
                <div className="mt-4 flex items-center gap-4 text-sm text-text-muted">
                  <div className="flex items-center gap-2">
                    <div className="w-4 h-0.5 bg-accent-primary"></div>
                    <span>Prix du marché</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="w-3 h-3 rounded-full bg-success"></div>
                    <span>Achats</span>
                  </div>
                </div>
              </div>
            ) : (
              <div className="bg-background-tertiary rounded-lg p-8 text-center">
                <p className="text-text-muted">Aucune donnée de prix disponible</p>
              </div>
            )}
          </div>
        )}

        {/* Assets Table */}
        <div className="bg-background-secondary rounded-lg overflow-hidden border border-background-tertiary">
          <table className="min-w-full divide-y divide-background-tertiary">
            <thead className="bg-background-tertiary">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-text-muted uppercase tracking-wider">
                  Actif
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  Quantité
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  Prix Moyen
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  Prix Actuel
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  Valeur
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  Gain/Perte
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-text-muted uppercase tracking-wider">
                  %
                </th>
                <th className="px-6 py-3 text-center text-xs font-medium text-text-muted uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-background-tertiary">
              {assets.map((asset) => (
                <tr
                  key={asset.isin}
                  onClick={() => setSelectedAsset(asset)}
                  className={`cursor-pointer transition-colors ${
                    selectedAsset?.isin === asset.isin
                      ? 'bg-background-tertiary'
                      : 'hover:bg-background-tertiary/50'
                  }`}
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-2">
                      <div className="flex flex-col">
                        <div className="text-sm font-medium text-text-primary">
                          {asset.name}
                        </div>
                        <div className="text-xs text-text-muted">{asset.isin}</div>
                      </div>
                      {!asset.symbol_verified && (
                        <AlertCircle className="w-4 h-4 text-orange-500" />
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-right text-sm text-text-primary">
                    {asset.quantity.toFixed(6)}
                  </td>
                  <td className="px-6 py-4 text-right text-sm text-text-primary">
                    {formatCurrency(asset.average_buy_price, asset.currency)}
                  </td>
                  <td className="px-6 py-4 text-right text-sm text-text-primary">
                    {formatCurrency(asset.current_price, asset.currency)}
                  </td>
                  <td className="px-6 py-4 text-right text-sm font-medium text-text-primary">
                    {formatCurrency(asset.current_value, asset.currency)}
                  </td>
                  <td
                    className={`px-6 py-4 text-right text-sm font-medium ${
                      asset.unrealized_gain >= 0 ? 'text-success' : 'text-error'
                    }`}
                  >
                    {formatCurrency(asset.unrealized_gain, asset.currency)}
                  </td>
                  <td
                    className={`px-6 py-4 text-right text-sm font-medium ${
                      asset.unrealized_gain_pct >= 0 ? 'text-success' : 'text-error'
                    }`}
                  >
                    {formatPercent(asset.unrealized_gain_pct)}
                  </td>
                  <td className="px-6 py-4 text-center">
                    {!asset.symbol_verified && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleOpenSearchModal(asset);
                        }}
                        className="inline-flex items-center gap-1 px-3 py-1 text-xs bg-orange-600 text-white rounded hover:bg-orange-700"
                      >
                        <Search className="w-3 h-3" />
                        Verify
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
