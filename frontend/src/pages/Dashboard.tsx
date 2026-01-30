import Header from '../components/Layout/Header'

export default function Dashboard() {
  return (
    <div>
      <Header 
        title="Dashboard" 
        subtitle="Vue d'ensemble de votre portefeuille"
      />
      
      <div className="p-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {/* Placeholder cards for metrics */}
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Valeur Totale</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
            <p className="text-success text-sm mt-2">+0.00%</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Investissement</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Gains/Pertes</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
          
          <div className="card">
            <p className="text-text-muted text-sm mb-2">Frais Totaux</p>
            <p className="text-3xl font-bold text-text-primary">€0.00</p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card">
            <h2 className="text-xl font-semibold mb-4">Performance Globale</h2>
            <div className="h-64 flex items-center justify-center text-text-muted">
              Graphique de performance à venir
            </div>
          </div>

          <div className="card">
            <h2 className="text-xl font-semibold mb-4">Comptes</h2>
            <div className="space-y-3">
              <p className="text-text-muted text-center py-8">
                Aucun compte connecté
              </p>
            </div>
          </div>
        </div>

        <div className="card mt-6">
          <h2 className="text-xl font-semibold mb-4">Dernières Transactions</h2>
          <div className="text-text-muted text-center py-8">
            Aucune transaction
          </div>
        </div>
      </div>
    </div>
  )
}
