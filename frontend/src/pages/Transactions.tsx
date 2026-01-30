import Header from '../components/Layout/Header'

export default function Transactions() {
  return (
    <div>
      <Header 
        title="Transactions" 
        subtitle="Historique de toutes vos transactions"
        actions={
          <button className="btn-primary">
            Importer CSV
          </button>
        }
      />
      
      <div className="p-8">
        <div className="card mb-6">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm text-text-muted mb-2">Date de début</label>
              <input 
                type="date" 
                className="input w-full"
                placeholder="Date de début"
              />
            </div>
            
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm text-text-muted mb-2">Date de fin</label>
              <input 
                type="date" 
                className="input w-full"
                placeholder="Date de fin"
              />
            </div>
            
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm text-text-muted mb-2">Type</label>
              <select className="input w-full">
                <option value="">Tous les types</option>
                <option value="buy">Achat</option>
                <option value="sell">Vente</option>
                <option value="dividend">Dividende</option>
                <option value="fee">Frais</option>
              </select>
            </div>
            
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm text-text-muted mb-2">Actif</label>
              <input 
                type="text" 
                className="input w-full"
                placeholder="Rechercher un actif..."
              />
            </div>
          </div>
        </div>

        <div className="card">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-background-tertiary">
                  <th className="text-left py-3 px-4 text-text-muted font-medium">Date</th>
                  <th className="text-left py-3 px-4 text-text-muted font-medium">Actif</th>
                  <th className="text-left py-3 px-4 text-text-muted font-medium">Type</th>
                  <th className="text-right py-3 px-4 text-text-muted font-medium">Quantité</th>
                  <th className="text-right py-3 px-4 text-text-muted font-medium">Montant</th>
                  <th className="text-right py-3 px-4 text-text-muted font-medium">Frais</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td colSpan={6} className="text-center py-12 text-text-muted">
                    Aucune transaction trouvée
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  )
}
