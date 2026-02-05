import { useState } from 'react';
import { X, Search, Check, AlertCircle } from 'lucide-react';

interface SymbolSearchResult {
  symbol: string;
  longname?: string;
  shortname?: string;
  quoteType: string;
  typeDisp: string;
  exchange: string;
  exchDisp: string;
  sector?: string;
  industry?: string;
  score: number;
}

interface SymbolSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  asset: {
    isin: string;
    name: string;
    symbol?: string;
    symbol_verified?: boolean;
  };
  onSymbolSelected: () => void;
}

export default function SymbolSearchModal({ isOpen, onClose, asset, onSymbolSelected }: SymbolSearchModalProps) {
  const [searchQuery, setSearchQuery] = useState(asset.name || '');
  const [results, setResults] = useState<SymbolSearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [updating, setUpdating] = useState(false);

  if (!isOpen) return null;

  const handleSearch = async () => {
    if (!searchQuery.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`http://localhost:8080/api/symbols/search?query=${encodeURIComponent(searchQuery)}`);
      
      const data = await response.json();
      
      if (!response.ok) {
        // Display the error message from the API
        const errorMessage = data.error?.message || 'Search failed';
        setError(errorMessage);
        setResults([]);
        return;
      }

      setResults(data.results || []);
    } catch (err) {
      setError('Failed to search symbols. Please try again.');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectSymbol = async (symbol: string) => {
    setUpdating(true);
    setError(null);

    try {
      const response = await fetch(`http://localhost:8080/api/assets/${asset.isin}/symbol`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          symbol: symbol,
          symbol_verified: true,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to update symbol');
      }

      onSymbolSelected();
      onClose();
    } catch (err) {
      setError('Failed to update symbol. Please try again.');
      console.error('Update error:', err);
    } finally {
      setUpdating(false);
    }
  };

  const handleMarkUnavailable = async () => {
    setUpdating(true);
    setError(null);

    try {
      const response = await fetch(`http://localhost:8080/api/assets/${asset.isin}/symbol`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          symbol: '',
          symbol_verified: true,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to mark as unavailable');
      }

      onSymbolSelected();
      onClose();
    } catch (err) {
      setError('Failed to mark as unavailable. Please try again.');
      console.error('Update error:', err);
    } finally {
      setUpdating(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/30 backdrop-blur-sm flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">Search Symbol</h2>
            <p className="text-sm text-gray-600 mt-1">
              {asset.name} ({asset.isin})
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Search */}
        <div className="p-6 border-b">
          <div className="flex gap-2">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="Search by name or symbol..."
              className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-900 bg-white placeholder-gray-400"
            />
            <button
              onClick={handleSearch}
              disabled={loading || !searchQuery.trim()}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed flex items-center gap-2"
            >
              <Search className="w-4 h-4" />
              {loading ? 'Searching...' : 'Search'}
            </button>
          </div>

          {error && (
            <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-red-700">
              <div className="flex items-start gap-2">
                <AlertCircle className="w-5 h-5 flex-shrink-0 mt-0.5" />
                <div className="flex-1">
                  <p className="font-medium">Search Error</p>
                  <p className="text-sm mt-1">{error}</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Results */}
        <div className="p-6 overflow-y-auto max-h-[50vh]">
          {results.length === 0 && !loading && (
            <div className="text-center text-gray-500 py-8">
              {searchQuery ? 'No results found. Try a different search term.' : 'Enter a search term to find symbols.'}
            </div>
          )}

          {results.length > 0 && (
            <div className="space-y-3">
              {results.map((result, index) => (
                <div
                  key={index}
                  className="border border-gray-200 rounded-lg p-4 hover:border-blue-500 transition-colors"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <span className="font-mono font-semibold text-lg text-gray-900">
                          {result.symbol}
                        </span>
                        <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded">
                          {result.typeDisp}
                        </span>
                        <span className="text-sm text-gray-600">
                          {result.exchDisp}
                        </span>
                      </div>
                      <p className="text-gray-700 mb-2">{result.longname || result.shortname}</p>
                      <div className="flex items-center gap-4 text-sm text-gray-600">
                        <span>Exchange: {result.exchange}</span>
                        {result.sector && <span>Sector: {result.sector}</span>}
                        {result.industry && <span>Industry: {result.industry}</span>}
                      </div>
                    </div>
                    <button
                      onClick={() => handleSelectSymbol(result.symbol)}
                      disabled={updating}
                      className="ml-4 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:bg-gray-300 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                      <Check className="w-4 h-4" />
                      Use This
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-6 border-t bg-gray-50 flex items-center justify-between">
          <button
            onClick={handleMarkUnavailable}
            disabled={updating}
            className="px-4 py-2 text-gray-700 hover:text-gray-900 disabled:text-gray-400"
          >
            Mark as unavailable
          </button>
          <button
            onClick={onClose}
            className="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
}
