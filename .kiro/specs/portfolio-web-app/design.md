# Document de Design

## Vue d'Ensemble

Valhafin est une application web de gestion de portefeuille financier composée de deux parties principales :

1. **Backend Go** : API RESTful qui réutilise les scrapers existants, gère la base de données PostgreSQL, et récupère les prix des actifs depuis des API financières externes
2. **Frontend Web** : Interface utilisateur moderne avec thème sombre, construite avec React et TypeScript, affichant les transactions, graphiques de performance et métriques

L'architecture suit un modèle client-serveur classique où le frontend communique avec le backend via HTTP/REST. Le backend orchestre les scrapers existants, stocke les données dans PostgreSQL, et enrichit les transactions avec les prix actuels des actifs.

## Architecture

### Architecture Globale

```mermaid
graph TB
    subgraph "Frontend - React + TypeScript"
        UI[Interface Utilisateur]
        Charts[Bibliothèque de Graphiques]
        State[Gestion d'État]
    end
    
    subgraph "Backend - Go"
        API[API REST Handler]
        Auth[Middleware CORS]
        DB[Database Layer]
        Scrapers[Scrapers Existants]
        PriceService[Service de Prix]
        Scheduler[Planificateur]
    end
    
    subgraph "Données Externes"
        TR[Trade Republic API]
        BN[Binance API]
        BD[Bourse Direct]
        Finance[API Financière<br/>Yahoo Finance]
    end
    
    subgraph "Stockage"
        PG[(PostgreSQL)]
    end
    
    UI --> API
    Charts --> UI
    State --> UI
    
    API --> Auth
    API --> DB
    API --> Scrapers
    API --> PriceService
    
    Scrapers --> TR
    Scrapers --> BN
    Scrapers --> BD
    
    PriceService --> Finance
    
    DB --> PG
    
    Scheduler --> Scrapers
    Scheduler --> PriceService
```

### Flux de Données Principaux

**1. Synchronisation des Transactions**
```
Utilisateur → Frontend → API /sync → Scraper → Plateforme Externe → Transactions → PostgreSQL
```

**2. Récupération des Prix**
```
Scheduler → PriceService → Yahoo Finance API → Prix Actuels → PostgreSQL
```

**3. Affichage des Performances**
```
Frontend → API /performance → DB (Transactions + Prix) → Calcul Performance → Frontend (Graphique)
```

## Composants et Interfaces

### Backend Go

#### 1. Structure des Packages

```
valhafin/
├── main.go                    # Point d'entrée, initialisation serveur HTTP
├── api/
│   ├── handlers.go           # Handlers HTTP pour les endpoints REST
│   ├── middleware.go         # CORS, logging, error handling
│   └── routes.go             # Définition des routes
├── config/
│   └── config.go             # Configuration existante
├── database/
│   ├── db.go                 # Connexion PostgreSQL
│   ├── migrations.go         # Migrations de schéma
│   ├── accounts.go           # CRUD pour les comptes
│   ├── transactions.go       # CRUD pour les transactions
│   └── prices.go             # CRUD pour les prix des actifs
├── models/
│   ├── transaction.go        # Modèle existant
│   ├── account.go            # Modèle de compte financier
│   ├── asset.go              # Modèle d'actif avec ISIN
│   └── price.go              # Modèle de prix d'actif
├── scrapers/
│   ├── traderepublic/        # Scraper existant
│   ├── binance/              # Scraper existant
│   └── boursedirect/         # Scraper existant
├── services/
│   ├── price_service.go      # Service de récupération des prix
│   ├── performance.go        # Calcul de performance
│   └── scheduler.go          # Planification des tâches
└── utils/
    └── export.go             # Utilitaires existants
```

#### 2. Modèles de Données

**Account (Nouveau)**
```go
type Account struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Platform    string    `json:"platform" db:"platform"` // "traderepublic", "binance", "boursedirect"
    Credentials string    `json:"-" db:"credentials"`     // Chiffré
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    LastSync    time.Time `json:"last_sync" db:"last_sync"`
}
```

**Asset (Nouveau)**
```go
type Asset struct {
    ISIN        string    `json:"isin" db:"isin"`
    Name        string    `json:"name" db:"name"`
    Symbol      string    `json:"symbol" db:"symbol"`
    Type        string    `json:"type" db:"type"` // "stock", "etf", "crypto"
    Currency    string    `json:"currency" db:"currency"`
    LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}
```

**AssetPrice (Nouveau)**
```go
type AssetPrice struct {
    ID        int64     `json:"id" db:"id"`
    ISIN      string    `json:"isin" db:"isin"`
    Price     float64   `json:"price" db:"price"`
    Currency  string    `json:"currency" db:"currency"`
    Timestamp time.Time `json:"timestamp" db:"timestamp"`
}
```

**Transaction (Étendu)**
```go
// Étendre le modèle existant avec :
type Transaction struct {
    // ... champs existants ...
    AccountID string  `json:"account_id" db:"account_id"`
    ISIN      string  `json:"isin" db:"isin"`
    Quantity  float64 `json:"quantity" db:"quantity"`
}
```

#### 3. API REST Endpoints

**Gestion des Comptes**
- `POST /api/accounts` - Ajouter un compte
- `GET /api/accounts` - Lister tous les comptes
- `GET /api/accounts/:id` - Détails d'un compte
- `DELETE /api/accounts/:id` - Supprimer un compte
- `POST /api/accounts/:id/sync` - Synchroniser les transactions

**Transactions**
- `GET /api/accounts/:id/transactions` - Lister les transactions d'un compte
  - Query params: `?start_date=`, `?end_date=`, `?asset=`, `?type=`, `?page=`, `?limit=`
- `GET /api/transactions` - Lister toutes les transactions (tous comptes)
- `POST /api/transactions/import` - Importer depuis CSV

**Performance**
- `GET /api/accounts/:id/performance` - Performance d'un compte
  - Query params: `?period=1m|3m|1y|all`
- `GET /api/performance` - Performance globale
- `GET /api/assets/:isin/performance` - Performance d'un actif spécifique

**Métriques**
- `GET /api/accounts/:id/fees` - Métriques de frais par compte
- `GET /api/fees` - Métriques de frais globales
  - Query params: `?start_date=`, `?end_date=`

**Prix des Actifs**
- `GET /api/assets/:isin/price` - Prix actuel d'un actif
- `GET /api/assets/:isin/history` - Historique des prix
  - Query params: `?start_date=`, `?end_date=`

#### 4. Service de Prix (PriceService)

**Interface**
```go
type PriceService interface {
    GetCurrentPrice(isin string) (*AssetPrice, error)
    GetPriceHistory(isin string, startDate, endDate time.Time) ([]AssetPrice, error)
    UpdateAllPrices() error
}
```

**Implémentation avec Yahoo Finance**
```go
type YahooFinancePriceService struct {
    client *http.Client
    cache  *PriceCache
}

func (s *YahooFinancePriceService) GetCurrentPrice(isin string) (*AssetPrice, error) {
    // 1. Vérifier le cache (< 1 heure)
    // 2. Convertir ISIN en symbole Yahoo Finance
    // 3. Appeler l'API Yahoo Finance
    // 4. Parser la réponse
    // 5. Stocker dans la base de données
    // 6. Retourner le prix
}
```

**API Alternative** : Alpha Vantage comme fallback si Yahoo Finance échoue

#### 5. Service de Performance

**Interface**
```go
type PerformanceService interface {
    CalculateAccountPerformance(accountID string, period string) (*Performance, error)
    CalculateGlobalPerformance(period string) (*Performance, error)
    CalculateAssetPerformance(isin string, period string) (*AssetPerformance, error)
}
```

**Modèle de Performance**
```go
type Performance struct {
    TotalValue        float64            `json:"total_value"`
    TotalInvested     float64            `json:"total_invested"`
    TotalFees         float64            `json:"total_fees"`
    RealizedGains     float64            `json:"realized_gains"`
    UnrealizedGains   float64            `json:"unrealized_gains"`
    PerformancePct    float64            `json:"performance_pct"`
    TimeSeries        []PerformancePoint `json:"time_series"`
}

type PerformancePoint struct {
    Date  time.Time `json:"date"`
    Value float64   `json:"value"`
}
```

**Algorithme de Calcul**
```
1. Récupérer toutes les transactions pour la période
2. Pour chaque actif détenu :
   a. Calculer la quantité totale détenue
   b. Récupérer le prix actuel via PriceService
   c. Calculer la valeur actuelle = quantité × prix_actuel
3. Calculer l'investissement total = somme des achats
4. Calculer les frais totaux = somme de tous les frais
5. Calculer les gains réalisés = ventes - achats correspondants
6. Calculer les gains non réalisés = valeur_actuelle - investissement - gains_réalisés
7. Performance % = ((valeur_actuelle - investissement_total - frais_totaux) / investissement_total) × 100
```

#### 6. Planificateur (Scheduler)

**Tâches Planifiées**
```go
type Scheduler struct {
    priceService PriceService
    db           *database.DB
}

func (s *Scheduler) Start() {
    // Mise à jour des prix toutes les heures
    go s.scheduleTask(1*time.Hour, s.updatePrices)
    
    // Synchronisation automatique des comptes (quotidienne)
    go s.scheduleTask(24*time.Hour, s.syncAccounts)
}
```

### Frontend React + TypeScript

#### 1. Structure des Composants

```
frontend/
├── src/
│   ├── components/
│   │   ├── Layout/
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── Layout.tsx
│   │   ├── Accounts/
│   │   │   ├── AccountList.tsx
│   │   │   ├── AccountCard.tsx
│   │   │   └── AddAccountModal.tsx
│   │   ├── Transactions/
│   │   │   ├── TransactionList.tsx
│   │   │   ├── TransactionTable.tsx
│   │   │   ├── TransactionFilters.tsx
│   │   │   └── ImportCSVModal.tsx
│   │   ├── Performance/
│   │   │   ├── PerformanceChart.tsx
│   │   │   ├── PerformanceMetrics.tsx
│   │   │   └── AssetPerformanceModal.tsx
│   │   ├── Fees/
│   │   │   ├── FeesOverview.tsx
│   │   │   └── FeesChart.tsx
│   │   └── Common/
│   │       ├── LoadingSpinner.tsx
│   │       ├── ErrorMessage.tsx
│   │       └── Pagination.tsx
│   ├── pages/
│   │   ├── Dashboard.tsx
│   │   ├── AccountDetails.tsx
│   │   ├── Transactions.tsx
│   │   ├── Performance.tsx
│   │   └── Fees.tsx
│   ├── services/
│   │   ├── api.ts              # Client API HTTP
│   │   ├── accounts.ts
│   │   ├── transactions.ts
│   │   ├── performance.ts
│   │   └── fees.ts
│   ├── hooks/
│   │   ├── useAccounts.ts
│   │   ├── useTransactions.ts
│   │   ├── usePerformance.ts
│   │   └── useFees.ts
│   ├── types/
│   │   ├── account.ts
│   │   ├── transaction.ts
│   │   ├── performance.ts
│   │   └── asset.ts
│   ├── styles/
│   │   ├── theme.ts            # Thème sombre + bleu
│   │   └── global.css
│   ├── App.tsx
│   └── main.tsx
├── package.json
└── vite.config.ts
```

#### 2. Stack Technologique Frontend

- **Framework** : React 18 avec TypeScript
- **Build Tool** : Vite (rapide, moderne)
- **Routing** : React Router v6
- **State Management** : React Query (TanStack Query) pour le cache et la synchronisation des données
- **Graphiques** : Recharts (bibliothèque de graphiques React native)
- **Styling** : Tailwind CSS avec thème personnalisé
- **HTTP Client** : Axios
- **Formulaires** : React Hook Form
- **Validation** : Zod

#### 3. Thème Visuel

**Palette de Couleurs**
```typescript
const theme = {
  colors: {
    background: {
      primary: '#0a0a0a',      // Noir profond
      secondary: '#1a1a1a',    // Gris très foncé
      tertiary: '#2a2a2a',     // Gris anthracite
    },
    text: {
      primary: '#ffffff',      // Blanc
      secondary: '#b0b0b0',    // Gris clair
      muted: '#6b6b6b',        // Gris moyen
    },
    accent: {
      primary: '#3b82f6',      // Bleu principal
      hover: '#2563eb',        // Bleu hover
      light: '#60a5fa',        // Bleu clair
    },
    success: '#10b981',        // Vert pour gains
    error: '#ef4444',          // Rouge pour pertes
    warning: '#f59e0b',        // Orange pour avertissements
  },
  borderRadius: {
    sm: '0.375rem',
    md: '0.5rem',
    lg: '0.75rem',
  },
  shadows: {
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.5)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.5)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.5)',
  },
}
```

#### 4. Composants Clés

**PerformanceChart.tsx**
```typescript
interface PerformanceChartProps {
  data: PerformancePoint[];
  period: '1m' | '3m' | '1y' | 'all';
  onPeriodChange: (period: string) => void;
}

// Utilise Recharts LineChart avec :
// - Axe X : dates
// - Axe Y : valeur du portefeuille
// - Tooltip personnalisé avec date et valeur
// - Gradient bleu pour la ligne
// - Zone remplie sous la courbe
```

**TransactionTable.tsx**
```typescript
interface TransactionTableProps {
  transactions: Transaction[];
  filters: TransactionFilters;
  onFilterChange: (filters: TransactionFilters) => void;
  onSort: (column: string, direction: 'asc' | 'desc') => void;
}

// Tableau avec :
// - Colonnes : Date, Actif, Type, Montant, Frais
// - Tri cliquable sur chaque colonne
// - Filtres : date range, type, actif
// - Pagination
// - Click sur actif → ouvre AssetPerformanceModal
```

**AssetPerformanceModal.tsx**
```typescript
interface AssetPerformanceModalProps {
  isin: string;
  isOpen: boolean;
  onClose: () => void;
}

// Modal affichant :
// - Nom de l'actif et ISIN
// - Prix actuel
// - Graphique de performance historique
// - Métriques : gain/perte, % de performance
```

## Modèles de Données

### Schéma PostgreSQL

```sql
-- Table des comptes financiers
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL, -- 'traderepublic', 'binance', 'boursedirect'
    credentials TEXT NOT NULL,      -- Chiffré avec AES-256
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_sync TIMESTAMP
);

-- Table des actifs
CREATE TABLE assets (
    isin VARCHAR(12) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(20),
    type VARCHAR(20) NOT NULL,      -- 'stock', 'etf', 'crypto'
    currency VARCHAR(3) NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table des prix des actifs
CREATE TABLE asset_prices (
    id BIGSERIAL PRIMARY KEY,
    isin VARCHAR(12) REFERENCES assets(isin),
    price DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    UNIQUE(isin, timestamp)
);

CREATE INDEX idx_asset_prices_isin_timestamp ON asset_prices(isin, timestamp DESC);

-- Table des transactions (une par compte)
-- Exemple pour Trade Republic :
CREATE TABLE transactions_traderepublic (
    id VARCHAR(255) PRIMARY KEY,
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    title VARCHAR(255),
    subtitle VARCHAR(255),
    isin VARCHAR(12) REFERENCES assets(isin),
    quantity DECIMAL(20, 8),
    amount_value DECIMAL(20, 8),
    amount_currency VARCHAR(3),
    fees DECIMAL(20, 8),
    transaction_type VARCHAR(50),   -- 'buy', 'sell', 'dividend', 'fee'
    status VARCHAR(50),
    metadata JSONB                  -- Autres champs spécifiques
);

CREATE INDEX idx_transactions_tr_account ON transactions_traderepublic(account_id);
CREATE INDEX idx_transactions_tr_timestamp ON transactions_traderepublic(timestamp DESC);
CREATE INDEX idx_transactions_tr_isin ON transactions_traderepublic(isin);

-- Même structure pour Binance et Bourse Direct
CREATE TABLE transactions_binance (...);
CREATE TABLE transactions_boursedirect (...);
```

### Relations

```mermaid
erDiagram
    ACCOUNTS ||--o{ TRANSACTIONS : has
    ASSETS ||--o{ TRANSACTIONS : involves
    ASSETS ||--o{ ASSET_PRICES : has
    
    ACCOUNTS {
        uuid id PK
        string name
        string platform
        text credentials
        timestamp last_sync
    }
    
    ASSETS {
        string isin PK
        string name
        string symbol
        string type
        string currency
    }
    
    ASSET_PRICES {
        bigint id PK
        string isin FK
        decimal price
        timestamp timestamp
    }
    
    TRANSACTIONS {
        string id PK
        uuid account_id FK
        timestamp timestamp
        string isin FK
        decimal quantity
        decimal amount
        decimal fees
        string type
    }
```

## Gestion des Erreurs

### Backend

**Codes d'Erreur HTTP**
- `400 Bad Request` : Données invalides (validation échouée)
- `404 Not Found` : Ressource non trouvée
- `500 Internal Server Error` : Erreur serveur
- `503 Service Unavailable` : API externe indisponible

**Format de Réponse d'Erreur**
```json
{
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Les identifiants fournis sont invalides",
    "details": {
      "field": "pin",
      "reason": "PIN must be 4 digits"
    }
  }
}
```

**Gestion des Erreurs de Scraping**
```go
type ScraperError struct {
    Platform string
    Type     string // "auth", "network", "parsing"
    Message  string
    Retry    bool
}

// Stratégie de retry avec backoff exponentiel
// Logging détaillé pour debugging
// Notification à l'utilisateur via l'API
```

**Gestion des Erreurs d'API Financière**
```go
// Fallback : utiliser le dernier prix connu
// Cache des prix pour résilience
// Retry avec timeout
// Logging des échecs pour monitoring
```

### Frontend

**Gestion des Erreurs Réseau**
```typescript
// Utilisation de React Query pour :
// - Retry automatique (3 tentatives)
// - Cache des données
// - Affichage d'états de chargement
// - Messages d'erreur utilisateur-friendly

const { data, error, isLoading } = useQuery({
  queryKey: ['transactions', accountId],
  queryFn: () => fetchTransactions(accountId),
  retry: 3,
  staleTime: 5 * 60 * 1000, // 5 minutes
});
```

**Composant ErrorMessage**
```typescript
// Affiche les erreurs de manière non intrusive
// Toast notifications pour les erreurs temporaires
// Messages d'erreur détaillés pour les erreurs critiques
// Bouton "Réessayer" pour les erreurs récupérables
```

## Stratégie de Test

### Tests Backend (Go)

**Tests Unitaires**
- Tests des handlers API avec mocks
- Tests des services (PriceService, PerformanceService)
- Tests des fonctions de calcul de performance
- Tests de validation des données

**Tests d'Intégration**
- Tests avec base de données PostgreSQL de test
- Tests des scrapers avec données mockées
- Tests des endpoints API end-to-end

**Outils**
- `testing` package standard Go
- `testify` pour les assertions
- `sqlmock` pour mocker PostgreSQL
- `httptest` pour tester les handlers

### Tests Frontend (React)

**Tests Unitaires**
- Tests des composants avec React Testing Library
- Tests des hooks personnalisés
- Tests des fonctions utilitaires

**Tests d'Intégration**
- Tests des flows utilisateur complets
- Tests des interactions avec l'API (mockée)

**Outils**
- Vitest (rapide, compatible Vite)
- React Testing Library
- MSW (Mock Service Worker) pour mocker l'API

### Tests End-to-End

**Scénarios**
1. Ajouter un compte → Synchroniser → Voir les transactions
2. Consulter la performance globale
3. Filtrer les transactions par date
4. Cliquer sur un actif → Voir sa performance
5. Importer un CSV

**Outil** : Playwright

## Sécurité

### Chiffrement des Identifiants

```go
// Utilisation de AES-256-GCM pour chiffrer les credentials
type EncryptionService struct {
    key []byte // Clé de 32 bytes (256 bits)
}

func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
    // 1. Générer un nonce aléatoire
    // 2. Chiffrer avec AES-GCM
    // 3. Encoder en base64
    // 4. Retourner : nonce + ciphertext
}

func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
    // 1. Décoder base64
    // 2. Extraire nonce et ciphertext
    // 3. Déchiffrer avec AES-GCM
    // 4. Retourner plaintext
}
```

**Gestion de la Clé**
- Clé stockée dans une variable d'environnement
- Rotation de clé possible via migration
- Jamais loggée ou exposée

### CORS

```go
// Configuration CORS pour le frontend
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}
```

### Validation des Entrées

```go
// Validation stricte de toutes les entrées utilisateur
// Utilisation de bibliothèques de validation (validator)
// Sanitization des données avant stockage
// Protection contre les injections SQL (utilisation de prepared statements)
```

## Déploiement et Packaging

### Architecture de Déploiement

L'application sera déployée sur une VM provisionnée via Terraform. Le déploiement utilise Docker pour packager l'application et GitHub Actions pour automatiser les releases.

```mermaid
graph LR
    subgraph "GitHub"
        Code[Code Repository]
        Actions[GitHub Actions]
        Releases[GitHub Releases]
    end
    
    subgraph "Build Pipeline"
        Build[Build Backend + Frontend]
        Docker[Create Docker Images]
        Package[Create Release Package]
    end
    
    subgraph "VM (Terraform)"
        Deploy[Deploy Script]
        App[Valhafin App]
        PG[PostgreSQL]
    end
    
    Code --> Actions
    Actions --> Build
    Build --> Docker
    Docker --> Package
    Package --> Releases
    Releases --> Deploy
    Deploy --> App
    App --> PG
```

### Structure de Packaging

**1. Dockerfile Multi-Stage**

```dockerfile
# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o valhafin .

# Stage 3: Final Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=backend-builder /app/valhafin .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./valhafin"]
```

**2. Docker Compose pour Déploiement**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: valhafin-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-valhafin}
      POSTGRES_USER: ${POSTGRES_USER:-valhafin}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    ports:
      - "5432:5432"
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U valhafin"]
      interval: 10s
      timeout: 5s
      retries: 5

  valhafin:
    image: ghcr.io/${GITHUB_REPOSITORY}/valhafin:${VERSION:-latest}
    container_name: valhafin-app
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}
      PORT: 8080
      YAHOO_FINANCE_API_KEY: ${YAHOO_FINANCE_API_KEY}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
    driver: local

networks:
  default:
    name: valhafin-network
```

### GitHub Actions Workflows

**1. CI/CD Pipeline (.github/workflows/ci-cd.yml)**

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [ main, develop ]
    tags: [ 'v*' ]
  pull_request:
    branches: [ main, develop ]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  test-backend:
    name: Test Backend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload coverage
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html

  test-frontend:
    name: Test Frontend
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json
      
      - name: Install dependencies
        working-directory: ./frontend
        run: npm ci
      
      - name: Run tests
        working-directory: ./frontend
        run: npm test
      
      - name: Run linter
        working-directory: ./frontend
        run: npm run lint

  build-and-push:
    name: Build and Push Docker Image
    runs-on: ubuntu-latest
    needs: [test-backend, test-frontend]
    if: github.event_name == 'push' && (github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/tags/v'))
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      
      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=sha
      
      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  create-release:
    name: Create Release
    runs-on: ubuntu-latest
    needs: build-and-push
    if: startsWith(github.ref, 'refs/tags/v')
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      
      - name: Create deployment package
        run: |
          mkdir -p release
          cp docker-compose.yml release/
          cp .env.example release/.env
          cp deploy.sh release/
          chmod +x release/deploy.sh
          tar -czf valhafin-${{ github.ref_name }}.tar.gz -C release .
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: valhafin-${{ github.ref_name }}.tar.gz
          generate_release_notes: true
          draft: false
          prerelease: false
```

**2. Automated Dependency Updates (.github/workflows/dependabot.yml)**

```yaml
version: 2
updates:
  # Go dependencies
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10

  # Frontend dependencies
  - package-ecosystem: "npm"
    directory: "/frontend"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10

  # Docker dependencies
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5

  # GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 5
```

### Script de Déploiement

**deploy.sh**

```bash
#!/bin/bash
set -e

# Couleurs pour les logs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Valhafin Deployment Script${NC}"

# Vérifier que .env existe
if [ ! -f .env ]; then
    echo -e "${RED}❌ .env file not found!${NC}"
    echo "Please create .env file from .env.example"
    exit 1
fi

# Charger les variables d'environnement
source .env

# Vérifier les variables requises
required_vars=("POSTGRES_PASSWORD" "ENCRYPTION_KEY")
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo -e "${RED}❌ $var is not set in .env${NC}"
        exit 1
    fi
done

# Arrêter les conteneurs existants
echo -e "${YELLOW}⏹️  Stopping existing containers...${NC}"
docker-compose down

# Télécharger la dernière image
echo -e "${YELLOW}📥 Pulling latest image...${NC}"
docker-compose pull

# Démarrer les services
echo -e "${YELLOW}🔄 Starting services...${NC}"
docker-compose up -d

# Attendre que les services soient prêts
echo -e "${YELLOW}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Vérifier la santé des services
if docker-compose ps | grep -q "unhealthy"; then
    echo -e "${RED}❌ Some services are unhealthy${NC}"
    docker-compose ps
    docker-compose logs
    exit 1
fi

echo -e "${GREEN}✅ Deployment successful!${NC}"
echo -e "${GREEN}🌐 Application is running at http://localhost:8080${NC}"

# Afficher les logs
echo -e "${YELLOW}📋 Showing logs (Ctrl+C to exit)...${NC}"
docker-compose logs -f
```

**.env.example**

```bash
# PostgreSQL Configuration
POSTGRES_DB=valhafin
POSTGRES_USER=valhafin
POSTGRES_PASSWORD=changeme_secure_password

# Application Configuration
ENCRYPTION_KEY=changeme_32_byte_hex_key_here
VERSION=latest

# Optional: Yahoo Finance API Key
YAHOO_FINANCE_API_KEY=

# GitHub Container Registry (for private repos)
# GITHUB_TOKEN=ghp_your_token_here
```

### Terraform Configuration (Exemple)

**terraform/main.tf**

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Security Group
resource "aws_security_group" "valhafin" {
  name        = "valhafin-sg"
  description = "Security group for Valhafin application"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.admin_ip]
  }

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# EC2 Instance
resource "aws_instance" "valhafin" {
  ami           = var.ami_id
  instance_type = var.instance_type
  key_name      = var.key_name

  vpc_security_group_ids = [aws_security_group.valhafin.id]

  user_data = <<-EOF
              #!/bin/bash
              apt-get update
              apt-get install -y docker.io docker-compose
              systemctl start docker
              systemctl enable docker
              
              # Télécharger et déployer Valhafin
              cd /opt
              wget https://github.com/${var.github_repo}/releases/latest/download/valhafin-latest.tar.gz
              tar -xzf valhafin-latest.tar.gz
              cd valhafin
              
              # Configurer les variables d'environnement
              cat > .env <<EOL
              POSTGRES_PASSWORD=${var.postgres_password}
              ENCRYPTION_KEY=${var.encryption_key}
              VERSION=latest
              EOL
              
              # Démarrer l'application
              ./deploy.sh
              EOF

  tags = {
    Name = "valhafin-app"
  }
}

# Elastic IP
resource "aws_eip" "valhafin" {
  instance = aws_instance.valhafin.id
  domain   = "vpc"
}

output "public_ip" {
  value = aws_eip.valhafin.public_ip
}
```

**terraform/variables.tf**

```hcl
variable "aws_region" {
  description = "AWS region"
  default     = "eu-west-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  default     = "t3.small"
}

variable "ami_id" {
  description = "AMI ID for Ubuntu 22.04"
  default     = "ami-0c55b159cbfafe1f0"
}

variable "key_name" {
  description = "SSH key name"
  type        = string
}

variable "admin_ip" {
  description = "Admin IP for SSH access"
  type        = string
}

variable "github_repo" {
  description = "GitHub repository (owner/repo)"
  type        = string
}

variable "postgres_password" {
  description = "PostgreSQL password"
  type        = string
  sensitive   = true
}

variable "encryption_key" {
  description = "Encryption key for credentials"
  type        = string
  sensitive   = true
}
```

### Processus de Release

**1. Versioning Sémantique**
- Format: `vX.Y.Z` (ex: v1.0.0, v1.1.0, v2.0.0)
- Major (X): Changements incompatibles
- Minor (Y): Nouvelles fonctionnalités compatibles
- Patch (Z): Corrections de bugs

**2. Créer une Release**

```bash
# 1. Créer un tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. GitHub Actions build automatiquement l'image Docker
# 3. GitHub Actions crée la release avec le package de déploiement
```

**3. Déployer sur la VM**

```bash
# Sur la VM
cd /opt/valhafin
wget https://github.com/user/valhafin/releases/download/v1.0.0/valhafin-v1.0.0.tar.gz
tar -xzf valhafin-v1.0.0.tar.gz
./deploy.sh
```

### Monitoring et Logs

**1. Health Check Endpoint**

```go
// api/handlers.go
func HealthCheckHandler(c *gin.Context) {
    // Vérifier la connexion à la base de données
    if err := db.Ping(); err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "database": "down",
        })
        return
    }
    
    c.JSON(200, gin.H{
        "status": "healthy",
        "version": version,
        "uptime": time.Since(startTime).String(),
    })
}
```

**2. Logs Structurés**

```go
// Utiliser zerolog pour des logs structurés
log.Info().
    Str("endpoint", "/api/accounts").
    Str("method", "POST").
    Int("status", 201).
    Dur("duration", duration).
    Msg("Account created")
```

**3. Backup Automatique**

```bash
# Script de backup (à ajouter au cron)
#!/bin/bash
BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
docker exec valhafin-postgres pg_dump -U valhafin valhafin > $BACKUP_DIR/valhafin_$DATE.sql
gzip $BACKUP_DIR/valhafin_$DATE.sql

# Garder seulement les 7 derniers backups
find $BACKUP_DIR -name "valhafin_*.sql.gz" -mtime +7 -delete
```

### Configuration Locale

**Backend**
```bash
# Variables d'environnement
DATABASE_URL=postgresql://user:password@localhost:5432/valhafin
ENCRYPTION_KEY=<32-byte-hex-key>
YAHOO_FINANCE_API_KEY=<optional>
PORT=8080
```

**Frontend**
```bash
# .env
VITE_API_URL=http://localhost:8080
```


## Propriétés de Correction

*Une propriété est une caractéristique ou un comportement qui doit être vrai pour toutes les exécutions valides d'un système - essentiellement, une déclaration formelle sur ce que le système doit faire. Les propriétés servent de pont entre les spécifications lisibles par l'homme et les garanties de correction vérifiables par machine.*

### Propriété 1: Création de compte avec chiffrement

*Pour toute* plateforme supportée (Trade Republic, Binance, Bourse Direct) et tout ensemble d'identifiants valides, lorsqu'un compte est créé, le système doit stocker les identifiants de manière chiffrée dans la base de données et aucun identifiant ne doit être stocké en clair.

**Valide: Exigences 1.1, 1.2, 1.3, 1.5**

### Propriété 2: Rejet des identifiants invalides

*Pour tout* ensemble d'identifiants invalides (format incorrect, credentials manquants, authentification échouée), le système doit rejeter la création du compte et retourner un message d'erreur explicite.

**Valide: Exigences 1.4**

### Propriété 3: Suppression en cascade

*Pour tout* compte supprimé, toutes les données associées (identifiants, transactions, table dédiée) doivent être supprimées de la base de données et aucune donnée orpheline ne doit subsister.

**Valide: Exigences 1.6, 8.7**

### Propriété 4: Synchronisation complète initiale

*Pour tout* compte nouvellement connecté, la première synchronisation doit récupérer l'historique complet des transactions depuis la plateforme et les stocker dans la table dédiée avec tous les champs requis (date, actif, montant, frais, type, métadonnées).

**Valide: Exigences 2.1, 2.2, 2.3**

### Propriété 5: Synchronisation incrémentale

*Pour tout* compte déjà synchronisé, une nouvelle synchronisation doit récupérer uniquement les transactions postérieures à la dernière synchronisation et ne pas créer de doublons.

**Valide: Exigences 2.4**

### Propriété 6: Gestion d'erreur de synchronisation

*Pour toute* erreur survenant pendant une synchronisation (erreur réseau, authentification échouée, parsing échoué), le système doit logger l'erreur avec des détails suffisants et retourner un message d'erreur structuré à l'utilisateur.

**Valide: Exigences 2.5**

### Propriété 7: Filtrage des transactions

*Pour tout* filtre appliqué (date, type d'opération, actif), le système doit retourner uniquement les transactions correspondant exactement aux critères du filtre et aucune transaction ne doit être retournée si elle ne correspond pas.

**Valide: Exigences 3.2, 3.3, 3.4**

### Propriété 8: Tri des transactions

*Pour tout* critère de tri (date, montant) et ordre (croissant, décroissant), le système doit retourner les transactions dans l'ordre spécifié et l'ordre doit être cohérent avec le critère choisi.

**Valide: Exigences 3.5, 3.6**

### Propriété 9: Pagination des transactions

*Pour tout* ensemble de transactions dont la taille dépasse 50, le système doit paginer les résultats et chaque page doit contenir au maximum 50 transactions sans duplication entre les pages.

**Valide: Exigences 3.7**

### Propriété 10: Calcul de performance avec prix actuels

*Pour tout* portefeuille, la performance calculée doit utiliser les prix actuels des actifs (récupérés via ISIN depuis l'API financière) et inclure tous les frais de transaction dans le calcul, et la formule doit être : performance % = ((valeur_actuelle - investissement_total - frais_totaux) / investissement_total) × 100.

**Valide: Exigences 4.4, 4.6, 5.7**

### Propriété 11: Agrégation de performance globale

*Pour tout* ensemble de comptes, la performance globale doit être la somme des performances de chaque compte et doit refléter la valeur totale de tous les actifs détenus.

**Valide: Exigences 4.2**

### Propriété 12: Filtrage par période

*Pour toute* période sélectionnée (1 mois, 3 mois, 1 an, tout), le système doit retourner uniquement les données (transactions, performances, frais) dans la période spécifiée.

**Valide: Exigences 4.3, 5.5**

### Propriété 13: Identification par ISIN

*Pour tout* actif dans le système, l'identification et la récupération des prix doivent utiliser l'ISIN comme clé unique et aucun actif ne doit être identifié par un autre moyen.

**Valide: Exigences 4.9, 10.1**

### Propriété 14: Récupération et stockage des prix

*Pour tout* actif identifié par ISIN, le système doit récupérer le prix actuel depuis l'API financière externe, le stocker dans la base de données avec un timestamp, et mettre à jour périodiquement (quotidiennement ou horaire).

**Valide: Exigences 10.2, 10.3, 10.4**

### Propriété 15: Fallback sur dernier prix connu

*Pour tout* actif dont le prix ne peut pas être récupéré depuis l'API financière, le système doit utiliser le dernier prix connu stocké en base de données et afficher un avertissement indiquant que le prix n'est pas à jour.

**Valide: Exigences 10.5**

### Propriété 16: Calcul de valeur actuelle

*Pour tout* actif détenu dans le portefeuille, la valeur actuelle doit être calculée en multipliant la quantité détenue par le prix actuel de l'actif, et la somme de toutes les valeurs actuelles doit égaler la valeur totale du portefeuille.

**Valide: Exigences 10.7**

### Propriété 17: Agrégation des frais

*Pour tout* compte ou ensemble de comptes, le système doit calculer correctement le total des frais, les frais moyens par transaction, et la répartition par type d'opération, et ces métriques doivent être cohérentes avec les transactions stockées.

**Valide: Exigences 5.1, 5.2, 5.3, 5.4**

### Propriété 18: Validation des entrées API

*Pour toute* requête API avec des données invalides (format incorrect, champs manquants, valeurs hors limites), le système doit rejeter la requête avec un code HTTP 400 et un message d'erreur structuré en JSON décrivant l'erreur.

**Valide: Exigences 7.3, 7.4**

### Propriété 19: Schéma de table de transactions

*Pour toute* table de transactions créée (par compte), le schéma doit inclure les colonnes obligatoires : id, date, actif, montant, frais, type_opération, métadonnées, et des index doivent exister sur date, actif, et type_opération.

**Valide: Exigences 8.2, 8.3**

### Propriété 20: Parsing et validation CSV

*Pour tout* fichier CSV importé, le système doit valider la présence des colonnes requises (date, actif, montant, frais) avant le parsing, et si le fichier est invalide, rejeter l'import avec un rapport d'erreurs détaillé listant toutes les erreurs trouvées.

**Valide: Exigences 9.1, 9.2, 9.3**

### Propriété 21: Import CSV avec déduplication

*Pour tout* fichier CSV valide importé, le système doit insérer les transactions dans la table du compte correspondant, détecter et ignorer les transactions en double (basé sur id ou combinaison date+actif+montant), et retourner un résumé avec le nombre de transactions importées, ignorées, et erreurs.

**Valide: Exigences 9.4, 9.5, 9.6**

### Propriété 22: Round-trip chiffrement/déchiffrement

*Pour tout* identifiant ou clé API chiffré et stocké, le déchiffrement doit retourner exactement la valeur originale, et aucune perte de données ne doit survenir lors du round-trip chiffrement → stockage → déchiffrement.

**Valide: Exigences 1.5**

### Propriété 23: Logging des requêtes et erreurs

*Pour toute* requête API reçue et toute erreur survenue, le système doit créer une entrée de log avec timestamp, endpoint, paramètres (sanitisés), et détails de l'erreur le cas échéant.

**Valide: Exigences 7.5**

### Propriété 24: Packaging Docker

*Pour toute* version de l'application, le build Docker doit produire une image fonctionnelle contenant le backend Go et les assets frontend, et l'image doit démarrer correctement avec les variables d'environnement requises.

**Valide: Exigences 11.1, 11.2**

### Propriété 25: CI/CD automatisé

*Pour tout* push sur la branche main ou création de tag de version, GitHub Actions doit automatiquement exécuter les tests, builder l'image Docker, et créer une release si c'est un tag.

**Valide: Exigences 11.3, 11.4**

### Propriété 26: Health check

*Pour toute* requête au endpoint /health, le système doit retourner un statut 200 avec des informations sur l'état de l'application (version, uptime, état de la base de données) si tout fonctionne, ou 503 si un service critique est indisponible.

**Valide: Exigences 11.6**

### Propriété 27: Versioning sémantique

*Pour toute* release créée, le numéro de version doit suivre le format sémantique vX.Y.Z où X est le major, Y le minor, et Z le patch, et les tags Git doivent correspondre aux versions des releases.

**Valide: Exigences 11.8**
