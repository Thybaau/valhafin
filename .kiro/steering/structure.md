# Project Structure

## Root Layout

```
valhafin/
├── main.go                    # Application entry point
├── go.mod, go.sum            # Go dependencies
├── Makefile                  # Build and development commands
├── .env.example              # Environment template (copy to .env)
├── docker-compose.dev.yml    # PostgreSQL for development
├── internal/                 # Private application code (Go convention)
├── frontend/                 # React application
├── cmd/                      # Additional CLI tools
└── docs/                     # Documentation
```

## Backend Structure (`internal/`)

Following Go's `internal/` convention - code here cannot be imported by external projects.

```
internal/
├── api/                      # HTTP layer
│   ├── handlers.go          # Request handlers
│   ├── routes.go            # Route definitions
│   ├── middleware.go        # CORS, logging, recovery
│   └── validation.go        # Request validation
├── domain/
│   └── models/              # Business entities
│       ├── account.go
│       ├── asset.go
│       ├── transaction.go
│       └── asset_price.go
├── repository/
│   └── database/            # Data access layer
│       ├── db.go           # Connection management
│       ├── migrations.go   # Schema migrations
│       ├── accounts.go     # Account queries
│       ├── transactions.go # Transaction queries
│       └── prices.go       # Price queries
├── service/                 # Business logic
│   ├── encryption/         # AES-256-GCM credential encryption
│   ├── scraper/            # Platform-specific scrapers
│   │   ├── interface.go
│   │   ├── traderepublic/
│   │   ├── binance/
│   │   └── boursedirect/
│   ├── sync/               # Account synchronization
│   ├── price/              # Yahoo Finance integration
│   ├── performance/        # Performance calculations
│   ├── fees/               # Fee analysis
│   └── scheduler/          # Background job scheduling
├── config/                  # Configuration loading
└── utils/                   # Shared utilities
```

## Frontend Structure

```
frontend/
├── src/
│   ├── main.tsx            # Application entry point
│   ├── App.tsx             # Root component with routing
│   ├── components/         # Reusable UI components
│   │   ├── Accounts/
│   │   ├── Transactions/
│   │   ├── Performance/
│   │   ├── Fees/
│   │   ├── Assets/
│   │   ├── Layout/
│   │   └── common/         # Shared components (Loading, Error, etc.)
│   ├── pages/              # Route pages
│   │   ├── Dashboard.tsx
│   │   ├── Accounts.tsx
│   │   ├── Transactions.tsx
│   │   ├── Performance.tsx
│   │   ├── Fees.tsx
│   │   └── Assets.tsx
│   ├── services/           # API client functions
│   │   ├── api.ts         # Axios instance
│   │   ├── accounts.ts
│   │   ├── transactions.ts
│   │   ├── performance.ts
│   │   ├── fees.ts
│   │   └── assets.ts
│   ├── hooks/              # Custom React hooks
│   │   ├── useAccounts.ts
│   │   ├── useTransactions.ts
│   │   ├── usePerformance.ts
│   │   └── useFees.ts
│   └── types/              # TypeScript type definitions
│       └── index.ts
├── public/                 # Static assets
├── package.json
├── vite.config.ts
├── tailwind.config.js
└── tsconfig.json
```

## Architecture Patterns

### Backend

- **Layered Architecture**: API → Service → Repository → Database
- **Dependency Injection**: Services passed to handlers via constructor
- **Interface-based Design**: Scrapers and price services use interfaces
- **Repository Pattern**: Database access abstracted behind repository layer
- **Middleware Chain**: CORS → Recovery → Logging

### Frontend

- **Component-based**: Reusable components organized by feature
- **Custom Hooks**: Data fetching logic encapsulated in hooks
- **Service Layer**: API calls isolated in service modules
- **React Query**: Server state management with caching
- **Type Safety**: Full TypeScript coverage

## Testing Structure

- Backend tests: `*_test.go` files alongside source
- Property-based tests: Using gopter library (see `performance_test.go`)
- Frontend tests: `*.test.tsx` files (when present)

## Configuration Files

- `.env` - Backend environment variables (not committed)
- `frontend/.env` - Frontend environment variables (not committed)
- `.env.example` - Template for environment setup
- `docker-compose.dev.yml` - Development database configuration

## Documentation

- `docs/` - Comprehensive documentation
- `.kiro/specs/` - Feature specifications and design documents
- `README.md` - Quick start guide
