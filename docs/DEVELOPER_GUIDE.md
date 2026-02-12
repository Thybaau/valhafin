# Guide du Développeur - Valhafin

Ce guide fournit toutes les informations nécessaires pour contribuer au développement de Valhafin.

## Table des Matières

- [Architecture](#architecture)
- [Modèles de Données](#modèles-de-données)
- [Développement Local](#développement-local)
- [Structure du Code](#structure-du-code)
- [Conventions de Code](#conventions-de-code)
- [Tests](#tests)
- [Contribution](#contribution)

---

## Architecture

### Vue d'Ensemble

Valhafin suit une architecture en couches (layered architecture) avec séparation claire des responsabilités:

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (React + TS)                     │
│  Components → Pages → Hooks → Services → API Client         │
└─────────────────────────────────────────────────────────────┘
                              ↓ HTTP/REST
┌─────────────────────────────────────────────────────────────┐
│                      Backend (Go)                            │
│  API Handlers → Services → Repository → Database            │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                       │
└─────────────────────────────────────────────────────────────┘
```

### Couches Backend

#### 1. API Layer (`internal/api/`)

Responsabilités:
- Gestion des requêtes HTTP
- Validation des entrées
- Sérialisation/désérialisation JSON
- Gestion des erreurs HTTP
- Middleware (CORS, logging, recovery)

Fichiers principaux:
- `handlers.go` - Handlers HTTP pour chaque endpoint
- `routes.go` - Définition des routes et mapping vers les handlers
- `middleware.go` - Middleware chain (CORS, logging, recovery)
- `validation.go` - Validation des requêtes

#### 2. Service Layer (`internal/service/`)

Responsabilités:
- Logique métier
- Orchestration des opérations
- Calculs de performance et frais
- Intégration avec APIs externes

Modules:
- `encryption/` - Chiffrement AES-256-GCM des credentials
- `scraper/` - Scrapers pour chaque plateforme (Trade Republic, Binance, Bourse Direct)
- `sync/` - Synchronisation des comptes
- `price/` - Récupération des prix via Yahoo Finance
- `performance/` - Calculs de performance
- `fees/` - Analyse des frais
- `scheduler/` - Tâches planifiées (mise à jour des prix, sync auto)

#### 3. Repository Layer (`internal/repository/database/`)

Responsabilités:
- Accès aux données
- Requêtes SQL
- Gestion des transactions DB
- Migrations

Fichiers:
- `db.go` - Connexion et configuration PostgreSQL
- `migrations.go` - Migrations de schéma
- `accounts.go` - CRUD pour les comptes
- `transactions.go` - CRUD pour les transactions
- `prices.go` - CRUD pour les prix

#### 4. Domain Layer (`internal/domain/models/`)

Responsabilités:
- Définition des entités métier
- Validation des modèles
- Logique métier simple

Modèles:
- `account.go` - Compte financier
- `asset.go` - Actif (action, ETF, crypto)
- `transaction.go` - Transaction financière
- `asset_price.go` - Prix d'un actif

### Flux de Données

#### Exemple: Création d'un compte

```
1. Frontend: POST /api/accounts
   ↓
2. API Handler: handlers.CreateAccount()
   - Valide les données (validation.go)
   - Parse le JSON
   ↓
3. Service: encryption.Encrypt(credentials)
   - Chiffre les credentials avec AES-256-GCM
   ↓
4. Repository: database.CreateAccount()
   - INSERT dans PostgreSQL
   ↓
5. Response: JSON avec le compte créé
```

#### Exemple: Synchronisation d'un compte

```
1. Frontend: POST /api/accounts/:id/sync
   ↓
2. API Handler: handlers.SyncAccount()
   ↓
3. Service: sync.SyncAccount()
   - Récupère les credentials chiffrés
   - Déchiffre avec encryption.Decrypt()
   - Appelle le scraper approprié
   ↓
4. Scraper: scraper.FetchTransactions()
   - Se connecte à la plateforme
   - Récupère les transactions
   ↓
5. Repository: database.BulkInsertTransactions()
   - INSERT des nouvelles transactions
   ↓
6. Response: JSON avec le nombre de transactions ajoutées
```

---

## Modèles de Données

### Schéma PostgreSQL

#### Table: `accounts`

Stocke les comptes financiers connectés.

```sql
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL,  -- 'traderepublic', 'binance', 'boursedirect'
    credentials TEXT NOT NULL,       -- Chiffré avec AES-256-GCM
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_sync TIMESTAMP
);
```

**Champs:**
- `id` - UUID unique
- `name` - Nom donné par l'utilisateur
- `platform` - Plateforme (traderepublic, binance, boursedirect)
- `credentials` - Credentials chiffrés (JSON chiffré)
- `last_sync` - Date de dernière synchronisation

#### Table: `assets`

Stocke les actifs financiers (actions, ETF, crypto).

```sql
CREATE TABLE assets (
    isin VARCHAR(12) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    symbol VARCHAR(20),
    type VARCHAR(20) NOT NULL,       -- 'stock', 'etf', 'crypto'
    currency VARCHAR(3) NOT NULL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Champs:**
- `isin` - Code ISIN (clé primaire)
- `name` - Nom de l'actif
- `symbol` - Symbole boursier
- `type` - Type (stock, etf, crypto)
- `currency` - Devise (EUR, USD, etc.)

#### Table: `asset_prices`

Stocke l'historique des prix des actifs.

```sql
CREATE TABLE asset_prices (
    id BIGSERIAL PRIMARY KEY,
    isin VARCHAR(12) REFERENCES assets(isin),
    price DECIMAL(20, 8) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    UNIQUE(isin, timestamp)
);

CREATE INDEX idx_asset_prices_isin_timestamp ON asset_prices(isin, timestamp DESC);
```

**Champs:**
- `isin` - Référence vers l'actif
- `price` - Prix de l'actif
- `timestamp` - Date et heure du prix

#### Tables: `transactions_*`

Une table par plateforme pour stocker les transactions.

```sql
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
    transaction_type VARCHAR(50),    -- 'buy', 'sell', 'dividend', 'fee'
    status VARCHAR(50),
    metadata JSONB
);

CREATE INDEX idx_transactions_tr_account ON transactions_traderepublic(account_id);
CREATE INDEX idx_transactions_tr_timestamp ON transactions_traderepublic(timestamp DESC);
CREATE INDEX idx_transactions_tr_isin ON transactions_traderepublic(isin);
```

**Champs:**
- `id` - ID unique de la transaction
- `account_id` - Référence vers le compte (CASCADE DELETE)
- `timestamp` - Date et heure de la transaction
- `isin` - Référence vers l'actif
- `quantity` - Quantité achetée/vendue
- `amount_value` - Montant de la transaction
- `fees` - Frais de transaction
- `transaction_type` - Type (buy, sell, dividend, fee)
- `metadata` - Données supplémentaires spécifiques à la plateforme

### Modèles Go

#### Account

```go
type Account struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Platform    string    `json:"platform" db:"platform"`
    Credentials string    `json:"-" db:"credentials"`  // Jamais exposé en JSON
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    LastSync    *time.Time `json:"last_sync" db:"last_sync"`
}

// Validate vérifie que les champs requis sont présents
func (a *Account) Validate() error {
    if a.Name == "" {
        return errors.New("name is required")
    }
    if !isValidPlatform(a.Platform) {
        return errors.New("invalid platform")
    }
    return nil
}
```

#### Asset

```go
type Asset struct {
    ISIN        string    `json:"isin" db:"isin"`
    Name        string    `json:"name" db:"name"`
    Symbol      string    `json:"symbol" db:"symbol"`
    Type        string    `json:"type" db:"type"`
    Currency    string    `json:"currency" db:"currency"`
    LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}
```

#### Transaction

```go
type Transaction struct {
    ID              string    `json:"id" db:"id"`
    AccountID       string    `json:"account_id" db:"account_id"`
    Timestamp       time.Time `json:"timestamp" db:"timestamp"`
    Title           string    `json:"title" db:"title"`
    Subtitle        string    `json:"subtitle" db:"subtitle"`
    ISIN            string    `json:"isin" db:"isin"`
    Quantity        float64   `json:"quantity" db:"quantity"`
    AmountValue     float64   `json:"amount_value" db:"amount_value"`
    AmountCurrency  string    `json:"amount_currency" db:"amount_currency"`
    Fees            float64   `json:"fees" db:"fees"`
    TransactionType string    `json:"transaction_type" db:"transaction_type"`
    Status          string    `json:"status" db:"status"`
    Metadata        string    `json:"metadata" db:"metadata"`  // JSON string
}
```

---

## Développement Local

### Prérequis

- Go 1.23+
- Node.js 20+
- PostgreSQL 15+ (ou Docker)
- Make (optionnel)

### Configuration Initiale

```bash
# 1. Cloner le repo
git clone https://github.com/your-org/valhafin.git
cd valhafin

# 2. Installer les dépendances
make setup
# Ou manuellement:
# go mod download
# cd frontend && npm install

# 3. Configurer .env
cp .env.example .env
# Éditer .env avec vos valeurs

# 4. Générer une clé de chiffrement
openssl rand -hex 32
# Copier dans .env: ENCRYPTION_KEY=...
```

### Démarrage

#### Avec Make (recommandé)

```bash
# Terminal 1: PostgreSQL
make dev-db

# Terminal 2: Backend
make dev-backend

# Terminal 3: Frontend
make dev-frontend
```

#### Sans Make

```bash
# Terminal 1: PostgreSQL
docker-compose -f docker-compose.dev.yml up

# Terminal 2: Backend
go run main.go

# Terminal 3: Frontend
cd frontend
npm run dev
```

### URLs de Développement

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080
- Health Check: http://localhost:8080/health
- PostgreSQL: localhost:5432

### Hot Reload

- **Backend**: Utiliser `air` pour le hot reload
  ```bash
  go install github.com/cosmtrek/air@latest
  air
  ```

- **Frontend**: Vite fournit le hot reload par défaut

---

## Structure du Code

### Backend (`internal/`)

```
internal/
├── api/
│   ├── handlers.go          # HTTP handlers
│   ├── routes.go            # Route definitions
│   ├── middleware.go        # CORS, logging, recovery
│   └── validation.go        # Request validation
├── domain/
│   └── models/
│       ├── account.go       # Account model
│       ├── asset.go         # Asset model
│       ├── transaction.go   # Transaction model
│       └── asset_price.go   # AssetPrice model
├── repository/
│   └── database/
│       ├── db.go           # DB connection
│       ├── migrations.go   # Schema migrations
│       ├── accounts.go     # Account queries
│       ├── transactions.go # Transaction queries
│       └── prices.go       # Price queries
├── service/
│   ├── encryption/
│   │   ├── encryption.go   # AES-256-GCM encryption
│   │   └── encryption_test.go
│   ├── scraper/
│   │   ├── interface.go    # Scraper interface
│   │   ├── traderepublic/
│   │   │   ├── scraper.go
│   │   │   └── scraper_test.go
│   │   ├── binance/
│   │   └── boursedirect/
│   ├── sync/
│   │   ├── sync.go         # Account synchronization
│   │   └── sync_test.go
│   ├── price/
│   │   ├── price_service.go # Yahoo Finance integration
│   │   └── price_service_test.go
│   ├── performance/
│   │   ├── performance.go  # Performance calculations
│   │   └── performance_test.go
│   ├── fees/
│   │   ├── fees.go         # Fee analysis
│   │   └── fees_test.go
│   └── scheduler/
│       ├── scheduler.go    # Background jobs
│       └── scheduler_test.go
├── config/
│   └── config.go           # Configuration loading
└── utils/
    └── utils.go            # Utility functions
```

### Frontend (`frontend/src/`)

```
src/
├── main.tsx                # Entry point
├── App.tsx                 # Root component with routing
├── components/
│   ├── Accounts/
│   │   ├── AccountList.tsx
│   │   ├── AccountCard.tsx
│   │   └── AddAccountModal.tsx
│   ├── Transactions/
│   │   ├── TransactionTable.tsx
│   │   ├── TransactionFilters.tsx
│   │   └── ImportCSVModal.tsx
│   ├── Performance/
│   │   ├── PerformanceChart.tsx
│   │   └── PerformanceMetrics.tsx
│   ├── Fees/
│   │   ├── FeesOverview.tsx
│   │   └── FeesChart.tsx
│   ├── Assets/
│   │   ├── AssetDetailModal.tsx
│   │   └── SymbolSearchModal.tsx
│   ├── Layout/
│   │   ├── Sidebar.tsx
│   │   ├── Header.tsx
│   │   └── Layout.tsx
│   └── common/
│       ├── LoadingSpinner.tsx
│       ├── ErrorMessage.tsx
│       └── Pagination.tsx
├── pages/
│   ├── Dashboard.tsx
│   ├── Accounts.tsx
│   ├── Transactions.tsx
│   ├── Performance.tsx
│   ├── Fees.tsx
│   └── Assets.tsx
├── services/
│   ├── api.ts              # Axios instance
│   ├── accounts.ts         # Account API calls
│   ├── transactions.ts     # Transaction API calls
│   ├── performance.ts      # Performance API calls
│   ├── fees.ts             # Fees API calls
│   └── assets.ts           # Assets API calls
├── hooks/
│   ├── useAccounts.ts      # React Query hook for accounts
│   ├── useTransactions.ts  # React Query hook for transactions
│   ├── usePerformance.ts   # React Query hook for performance
│   └── useFees.ts          # React Query hook for fees
└── types/
    └── index.ts            # TypeScript type definitions
```

---

## Conventions de Code

### Backend (Go)

#### Naming

- **Packages**: lowercase, single word (e.g., `encryption`, `scraper`)
- **Files**: lowercase with underscores (e.g., `price_service.go`)
- **Types**: PascalCase (e.g., `Account`, `PriceService`)
- **Functions**: PascalCase for exported, camelCase for private
- **Variables**: camelCase
- **Constants**: PascalCase or UPPER_SNAKE_CASE

#### Structure

```go
// 1. Package declaration
package service

// 2. Imports (grouped: stdlib, external, internal)
import (
    "context"
    "time"
    
    "github.com/jmoiron/sqlx"
    
    "valhafin/internal/domain/models"
)

// 3. Constants
const (
    DefaultTimeout = 30 * time.Second
)

// 4. Types
type PriceService struct {
    db     *sqlx.DB
    client *http.Client
}

// 5. Constructor
func NewPriceService(db *sqlx.DB) *PriceService {
    return &PriceService{
        db:     db,
        client: &http.Client{Timeout: DefaultTimeout},
    }
}

// 6. Methods
func (s *PriceService) GetPrice(ctx context.Context, isin string) (*models.AssetPrice, error) {
    // Implementation
}
```

#### Error Handling

```go
// Retourner des erreurs avec contexte
if err != nil {
    return nil, fmt.Errorf("failed to fetch price for %s: %w", isin, err)
}

// Utiliser des erreurs personnalisées pour les cas métier
var ErrAccountNotFound = errors.New("account not found")

// Logger les erreurs avant de les retourner
if err != nil {
    log.Error().Err(err).Str("isin", isin).Msg("failed to fetch price")
    return nil, err
}
```

#### Interfaces

```go
// Définir des interfaces pour l'injection de dépendances
type Scraper interface {
    FetchTransactions(ctx context.Context, credentials string) ([]models.Transaction, error)
}

// Implémenter l'interface
type TradeRepublicScraper struct {
    // fields
}

func (s *TradeRepublicScraper) FetchTransactions(ctx context.Context, credentials string) ([]models.Transaction, error) {
    // Implementation
}
```

### Frontend (TypeScript/React)

#### Naming

- **Components**: PascalCase (e.g., `AccountList.tsx`)
- **Hooks**: camelCase with `use` prefix (e.g., `useAccounts.ts`)
- **Services**: camelCase (e.g., `accountService.ts`)
- **Types**: PascalCase (e.g., `Account`, `Transaction`)
- **Variables**: camelCase
- **Constants**: UPPER_SNAKE_CASE

#### Component Structure

```typescript
// 1. Imports
import { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AccountCard } from './AccountCard';
import { useAccounts } from '../../hooks/useAccounts';

// 2. Types
interface AccountListProps {
  onAccountClick: (id: string) => void;
}

// 3. Component
export function AccountList({ onAccountClick }: AccountListProps) {
  // Hooks
  const { data: accounts, isLoading, error } = useAccounts();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  
  // Effects
  useEffect(() => {
    // Side effects
  }, []);
  
  // Event handlers
  const handleClick = (id: string) => {
    setSelectedId(id);
    onAccountClick(id);
  };
  
  // Render
  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;
  
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {accounts?.map(account => (
        <AccountCard
          key={account.id}
          account={account}
          onClick={handleClick}
          isSelected={selectedId === account.id}
        />
      ))}
    </div>
  );
}
```

#### Hooks

```typescript
// Custom hook avec React Query
export function useAccounts() {
  return useQuery({
    queryKey: ['accounts'],
    queryFn: accountService.getAll,
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 3,
  });
}

// Hook avec mutation
export function useCreateAccount() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: accountService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['accounts'] });
    },
  });
}
```

#### Services

```typescript
// Service API
import axios from 'axios';
import { Account, CreateAccountRequest } from '../types';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
});

export const accountService = {
  getAll: async (): Promise<Account[]> => {
    const { data } = await api.get('/api/accounts');
    return data;
  },
  
  getById: async (id: string): Promise<Account> => {
    const { data } = await api.get(`/api/accounts/${id}`);
    return data;
  },
  
  create: async (request: CreateAccountRequest): Promise<Account> => {
    const { data } = await api.post('/api/accounts', request);
    return data;
  },
  
  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/accounts/${id}`);
  },
};
```

---

## Tests

### Backend Tests

#### Tests Unitaires

```go
// account_test.go
package models

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestAccount_Validate(t *testing.T) {
    tests := []struct {
        name    string
        account Account
        wantErr bool
    }{
        {
            name: "valid account",
            account: Account{
                Name:     "Test Account",
                Platform: "traderepublic",
            },
            wantErr: false,
        },
        {
            name: "missing name",
            account: Account{
                Platform: "traderepublic",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.account.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

#### Tests de Propriété (Property-Based Testing)

```go
// encryption_test.go
package encryption

import (
    "testing"
    "github.com/leanovate/gopter"
    "github.com/leanovate/gopter/gen"
    "github.com/leanovate/gopter/prop"
)

func TestEncryption_RoundTrip(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("encrypt then decrypt returns original", prop.ForAll(
        func(plaintext string) bool {
            service := NewEncryptionService(testKey)
            
            encrypted, err := service.Encrypt(plaintext)
            if err != nil {
                return false
            }
            
            decrypted, err := service.Decrypt(encrypted)
            if err != nil {
                return false
            }
            
            return plaintext == decrypted
        },
        gen.AnyString(),
    ))
    
    properties.TestingRun(t)
}
```

#### Lancer les Tests

```bash
# Tous les tests
go test ./...

# Avec coverage
go test -cover ./...

# Avec coverage détaillé
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Tests d'un package spécifique
go test ./internal/service/encryption/

# Tests avec verbose
go test -v ./...
```

### Frontend Tests

#### Tests de Composants

```typescript
// AccountCard.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { AccountCard } from './AccountCard';

describe('AccountCard', () => {
  const mockAccount = {
    id: '123',
    name: 'Test Account',
    platform: 'traderepublic',
    created_at: '2024-01-01T00:00:00Z',
    last_sync: '2024-01-15T10:30:00Z',
  };
  
  it('renders account name', () => {
    render(<AccountCard account={mockAccount} onClick={() => {}} />);
    expect(screen.getByText('Test Account')).toBeInTheDocument();
  });
  
  it('calls onClick when clicked', () => {
    const handleClick = jest.fn();
    render(<AccountCard account={mockAccount} onClick={handleClick} />);
    
    fireEvent.click(screen.getByRole('button'));
    expect(handleClick).toHaveBeenCalledWith('123');
  });
});
```

#### Tests de Hooks

```typescript
// useAccounts.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAccounts } from './useAccounts';

describe('useAccounts', () => {
  it('fetches accounts', async () => {
    const queryClient = new QueryClient();
    const wrapper = ({ children }) => (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
    
    const { result } = renderHook(() => useAccounts(), { wrapper });
    
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(2);
  });
});
```

#### Lancer les Tests

```bash
cd frontend

# Tous les tests
npm test

# Avec coverage
npm test -- --coverage

# Mode watch
npm test -- --watch

# Tests d'un fichier spécifique
npm test AccountCard.test.tsx
```

---

## Contribution

### Workflow Git

```bash
# 1. Créer une branche feature
git checkout -b feature/add-binance-scraper

# 2. Faire les modifications
# ... éditer les fichiers ...

# 3. Commit avec message conventionnel
git add .
git commit -m "feat(scraper): add Binance scraper implementation"

# 4. Push
git push origin feature/add-binance-scraper

# 5. Créer une Pull Request sur GitHub
```

### Conventional Commits

Format: `<type>(<scope>): <description>`

Types:
- `feat`: Nouvelle fonctionnalité
- `fix`: Correction de bug
- `docs`: Documentation
- `style`: Formatage (pas de changement de code)
- `refactor`: Refactoring
- `test`: Ajout de tests
- `chore`: Tâches de maintenance

Exemples:
```
feat(api): add endpoint for asset price history
fix(scraper): handle Trade Republic 2FA timeout
docs(readme): update deployment instructions
refactor(performance): optimize calculation algorithm
test(encryption): add property-based tests
```

### Pull Request

1. **Titre**: Utiliser conventional commits
2. **Description**: Expliquer le problème et la solution
3. **Tests**: Ajouter des tests pour les nouvelles fonctionnalités
4. **Documentation**: Mettre à jour la documentation si nécessaire
5. **Review**: Attendre l'approbation d'au moins un reviewer

### Code Review Checklist

- [ ] Le code suit les conventions de style
- [ ] Les tests passent
- [ ] La couverture de tests est maintenue ou améliorée
- [ ] La documentation est à jour
- [ ] Pas de secrets ou credentials en dur
- [ ] Les erreurs sont gérées correctement
- [ ] Le code est performant
- [ ] Pas de code dupliqué

---

## Ressources

### Documentation Externe

- [Go Documentation](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [TypeScript Documentation](https://www.typescriptlang.org/docs/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [TanStack Query](https://tanstack.com/query/latest)
- [Tailwind CSS](https://tailwindcss.com/docs)

### Documentation Interne

- [API Endpoints](./API_ENDPOINTS.md)
- [Production Deployment](./PRODUCTION_DEPLOYMENT.md)
- [Simple Startup Guide](./SIMPLE_STARTUP_GUIDE.md)
- [Specifications](./.kiro/specs/portfolio-web-app/)

---

## Support

Pour toute question ou problème:

1. Consulter la documentation
2. Chercher dans les issues GitHub
3. Créer une nouvelle issue avec le template approprié
4. Contacter l'équipe sur le canal de discussion

---

**Dernière mise à jour**: 2024-01-15
