import { useState, useEffect } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
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
import { Search, AlertCircle, CheckCircle, RefreshCw } from 'lucide-react';
import { assetsApi } from '../services';
import type { AssetPosition } from '../types';
import LoadingSpinner from '../components/common/LoadingSpinner';
import ErrorMessage from '../components/common/ErrorMessage';
import Header from '../components/Layout/Header';
import SymbolSearchModal from '../components/Assets/SymbolSearchModal';

interface PriceHistory {
  id: number;
  isin: string;
  price: number;
  currency: string;
  timestamp: string;
}

export default function Assets() {
  const queryClient = useQueryClient();
  const [selectedAsset, setSelectedAsset] = useState<AssetPosition | null>(null);
  const [searchModalOpen, setSearchModalOpen] = useState(false);
  const [assetToSearch, setAssetToSearch] = useState<AssetPosition | null>(null);
  const [timeRange, setTimeRange] = useState<'1W' | '1M' | '6M' | '1Y' | '2Y' | '5Y' | 'MAX'>('1Y');
  const [refreshingAsset, setRefreshingAsset] = useState<string | null>(null);

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
    queryKey: ['assetHistory', selectedAsset?.isin, timeRange],
    queryFn: () => {
      if (!selectedAsset) return Promise.resolve([]);
      const endDate = new Date();
      const startDate = new Date();
      
      // Calculate start date based on time range
      switch (timeRange) {
        case '1W':
          startDate.setDate(startDate.getDate() - 7);
          break;
        case '1M':
          startDate.setMonth(startDate.getMonth() - 1);
          break;
        case '6M':
          startDate.setMonth(startDate.getMonth() - 6);
          break;
        case '1Y':
          startDate.setFullYear(startDate.getFullYear() - 1);
          break;
        case '2Y':
          startDate.setFullYear(startDate.getFullYear() - 2);
          break;
        case '5Y':
          startDate.setFullYear(startDate.getFullYear() - 5);
          break;
        case 'MAX':
          startDate.setFullYear(startDate.getFullYear() - 20); // 20 years max
          break;
      }
      
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

  const handleRefreshPrices = async (isin: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setRefreshingAsset(isin);
    try {
      await assetsApi.refreshAssetPrices(isin);
      // Refetch assets
      refetch();
      // Invalidate price history cache to force refetch
      queryClient.invalidateQueries({ queryKey: ['assetHistory', isin] });
    } catch (error) {
      console.error('Failed to refresh prices:', error);
    } finally {
      setRefreshingAsset(null);
    }
  };

  // Prepare chart data
  const chartData = priceHistory?.map((point) => ({
    date: new Date(point.timestamp).toLocaleDateString('fr-FR', {
      day: '2-digit',
      month: '2-digit',
      year: '2-digit',
    }),
    timestamp: point.timestamp,
    price: point.price,
  })) || [];

  // Calculate period gain/loss
  const periodGain = chartData.length > 0 && selectedAsset ? (() => {
    const firstPrice = chartData[0].price;
    const lastPrice = chartData[chartData.length - 1].price;
    const currentQuantity = selectedAsset.quantity;
    
    // Calculate the start date of the period
    const periodStartDate = new Date(chartData[0].timestamp);
    
    // Get the date of the first purchase
    const firstPurchaseDate = selectedAsset.purchases && selectedAsset.purchases.length > 0
      ? new Date(Math.min(...selectedAsset.purchases.map(p => new Date(p.date).getTime())))
      : new Date();
    
    // If the period starts before or at the first purchase, use total gain/loss
    if (periodStartDate <= firstPurchaseDate) {
      return {
        gain: selectedAsset.unrealized_gain,
        gainPct: selectedAsset.unrealized_gain_pct,
        firstPrice,
        lastPrice,
        valueAtStart: selectedAsset.total_invested,
        valueNow: selectedAsset.current_value,
        netInvestments: selectedAsset.total_invested,
        quantityAtStart: 0,
        isTotal: true,
      };
    }
    
    // Filter purchases that happened during the period
    const purchasesDuringPeriod = selectedAsset.purchases?.filter(purchase => {
      const purchaseDate = new Date(purchase.date);
      return purchaseDate >= periodStartDate;
    }) || [];
    
    // Calculate total quantity bought during the period
    const quantityBoughtDuringPeriod = purchasesDuringPeriod.reduce(
      (sum, purchase) => sum + purchase.quantity,
      0
    );
    
    // Calculate quantity owned at the start of the period
    const quantityAtStart = currentQuantity - quantityBoughtDuringPeriod;
    
    // Calculate net investments during the period (money spent on purchases)
    const netInvestments = purchasesDuringPeriod.reduce(
      (sum, purchase) => sum + (purchase.quantity * purchase.price),
      0
    );
    
    // Calculate values
    const valueAtStart = quantityAtStart * firstPrice;
    const valueNow = currentQuantity * lastPrice;
    
    // Gain = Current value - Initial value - Net investments during period
    const gain = valueNow - valueAtStart - netInvestments;
    const totalInvested = valueAtStart + netInvestments;
    const gainPct = totalInvested > 0 ? (gain / totalInvested) * 100 : 0;
    
    return {
      gain,
      gainPct,
      firstPrice,
      lastPrice,
      valueAtStart,
      valueNow,
      netInvestments,
      quantityAtStart,
      isTotal: false,
    };
  })() : null;

  // Create purchase markers - only include purchases that have corresponding chart data
  const purchaseMarkers = selectedAsset?.purchases
    ?.map((purchase) => {
      const purchaseDate = new Date(purchase.date);
      
      // Find the closest point in chart data (within a few days)
      let closestPoint = null;
      let minDiff = Infinity;
      
      for (const point of chartData) {
        const pointDate = new Date(point.timestamp);
        const diff = Math.abs(pointDate.getTime() - purchaseDate.getTime());
        const daysDiff = diff / (1000 * 60 * 60 * 24);
        
        // Only consider points within 7 days
        if (daysDiff <= 7 && diff < minDiff) {
          minDiff = diff;
          closestPoint = point;
        }
      }

      // Only return marker if we found a close point
      if (closestPoint) {
        return {
          date: closestPoint.date,
          price: closestPoint.price,
          quantity: purchase.quantity,
          originalDate: purchase.date,
        };
      }
      return null;
    })
    .filter((marker): marker is NonNullable<typeof marker> => marker !== null) || [];

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

            {/* Time Range Selector */}
            <div className="mb-4 flex gap-2">
              {(['1W', '1M', '6M', '1Y', '2Y', '5Y', 'MAX'] as const).map((range) => (
                <button
                  key={range}
                  onClick={() => setTimeRange(range)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    timeRange === range
                      ? 'bg-accent-primary text-white'
                      : 'bg-background-tertiary text-text-muted hover:bg-background-tertiary/70'
                  }`}
                >
                  {range}
                </button>
              ))}
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
                <div className="text-sm text-text-muted">
                  Gain/Perte ({timeRange})
                </div>
                <div
                  className={`text-lg font-semibold mt-1 ${
                    periodGain && periodGain.gain >= 0 ? 'text-success' : 'text-error'
                  }`}
                >
                  {periodGain ? (
                    <>
                      {formatCurrency(periodGain.gain, selectedAsset.currency)}
                      <span className="text-sm ml-2">
                        ({formatPercent(periodGain.gainPct)})
                      </span>
                    </>
                  ) : (
                    '-'
                  )}
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
                      {asset.symbol_verified ? (
                        <div className="relative group">
                          <CheckCircle className="w-4 h-4 text-success cursor-help" />
                          <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 hidden group-hover:block z-10 w-48">
                            <div className="bg-background-tertiary border border-background-primary rounded-lg px-3 py-2 text-xs text-text-primary shadow-lg">
                              Symbole vérifié sur Yahoo Finance ({asset.symbol})
                            </div>
                          </div>
                        </div>
                      ) : (
                        <div className="relative group">
                          <AlertCircle className="w-4 h-4 text-orange-500 cursor-help" />
                          <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 hidden group-hover:block z-10 w-48">
                            <div className="bg-background-tertiary border border-background-primary rounded-lg px-3 py-2 text-xs text-text-primary shadow-lg">
                              Symbole non vérifié - Cliquez sur "Verify" pour rechercher
                            </div>
                          </div>
                        </div>
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
                    <div className="flex items-center justify-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleOpenSearchModal(asset);
                        }}
                        className={`inline-flex items-center gap-1 px-3 py-1 text-xs rounded transition-colors ${
                          asset.symbol_verified
                            ? 'bg-background-tertiary text-text-muted hover:bg-background-tertiary/70 hover:text-text-primary'
                            : 'bg-orange-600 text-white hover:bg-orange-700'
                        }`}
                      >
                        <Search className="w-3 h-3" />
                        {asset.symbol_verified ? 'Modify' : 'Verify'}
                      </button>
                      <button
                        onClick={(e) => handleRefreshPrices(asset.isin, e)}
                        disabled={refreshingAsset === asset.isin}
                        className="inline-flex items-center justify-center p-2 text-text-muted hover:text-accent-primary hover:bg-background-tertiary rounded transition-colors disabled:opacity-50"
                        title="Recharger les prix"
                      >
                        <RefreshCw
                          className={`w-4 h-4 ${refreshingAsset === asset.isin ? 'animate-spin' : ''}`}
                        />
                      </button>
                    </div>
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
