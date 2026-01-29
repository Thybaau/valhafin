# Getting Started with Valhafin 🔥⚔️

**Your journey to financial Valhalla begins here**

## 🚀 Quick Start

### 1. Installation

```bash
cd valhafin
go mod download
```

### 2. Configuration

Copie le fichier de configuration exemple :

```bash
cp config.yaml.example config.yaml
```

Édite `config.yaml` avec tes identifiants Trade Republic :

```yaml
secret:
  phone_number: "+33XXXXXXXXX"
  pin: "XXXX"

general:
  output_format: "csv"  # ou "json"
  output_folder: "out"
  extract_details: true
```

### 3. Exécution

```bash
# Avec Go
go run main.go

# Ou compile et exécute
make build
./valhafin
```

## 📊 Formats de sortie

### CSV (recommandé pour Excel)
- Séparateur : `;`
- Encodage : UTF-8 avec BOM
- Décimales : virgule (format français)

### JSON
- Format structuré pour intégration API
- Indentation pour lisibilité

## 🏗️ Architecture du projet

```
valhafin/
├── main.go                          # Point d'entrée
├── config/
│   └── config.go                    # Gestion de la configuration
├── models/
│   └── transaction.go               # Modèles de données
├── scrapers/
│   ├── traderepublic/              # ✅ Implémenté
│   │   ├── scraper.go
│   │   ├── auth.go
│   │   └── websocket.go
│   ├── binance/                     # 🚧 À implémenter
│   │   └── client.go
│   └── boursedirect/                # 🚧 À implémenter
│       └── scraper.go
└── utils/
    └── export.go                    # Export CSV/JSON
```

## 🔧 Développement

### Ajouter un nouveau scraper

1. Crée un nouveau package dans `scrapers/`
2. Implémente l'interface de scraping
3. Ajoute la configuration dans `config.yaml`
4. Intègre dans `main.go`

### Exemple pour Binance

```go
// scrapers/binance/client.go
package binance

import "valhafin/models"

type Client struct {
    apiKey string
    secret string
}

func (c *Client) FetchTransactions() ([]models.Transaction, error) {
    // Implémentation avec l'API Binance
}
```

### Tests

```bash
go test ./...
```

## 📝 Prochaines étapes

### Phase 1 : Binance (Facile - API officielle)
- [ ] Ajouter les credentials Binance dans config.yaml
- [ ] Implémenter le client API REST
- [ ] Mapper les données vers le modèle unifié
- [ ] Tester avec ton compte

### Phase 2 : Bourse Direct (Moyen - Scraping)
- [ ] Option A : Import CSV manuel
- [ ] Option B : Reverse engineering de l'API web
- [ ] Mapper les données vers le modèle unifié

### Phase 3 : Application Web
- [ ] Backend API (Go avec Gin ou Echo)
- [ ] Frontend (React/Vue.js)
- [ ] Base de données (PostgreSQL)
- [ ] Visualisations (Chart.js/Recharts)

## 🎯 Avantages de Go vs Python

- **Performance** : 10-50x plus rapide
- **Concurrence** : Goroutines natives pour scraping parallèle
- **Compilation** : Binaire unique, pas de dépendances
- **Typage** : Détection d'erreurs à la compilation
- **Déploiement** : Simple, pas besoin de venv

## 🐛 Troubleshooting

### Erreur de connexion WebSocket
- Vérifie ta connexion internet
- Vérifie que Trade Republic n'a pas changé son API

### Erreur d'authentification
- Vérifie ton numéro de téléphone (format international)
- Vérifie ton PIN
- Assure-toi de recevoir le code 2FA

### Erreur de compilation
```bash
go mod tidy
go clean -cache
go build
```
