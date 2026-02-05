# Technology Stack

## Backend

- **Language**: Go 1.21+
- **Web Framework**: Gorilla Mux (routing), standard library HTTP server
- **Database**: PostgreSQL 15+ with sqlx for queries
- **Configuration**: Viper + godotenv for environment management
- **Testing**: Standard Go testing + gopter (property-based testing)
- **Security**: AES-256-GCM encryption for credentials

### Key Backend Libraries

- `github.com/gorilla/mux` - HTTP routing
- `github.com/gorilla/websocket` - WebSocket support (Trade Republic scraper)
- `github.com/jmoiron/sqlx` - SQL extensions
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/spf13/viper` - Configuration management
- `github.com/joho/godotenv` - .env file loading
- `github.com/leanovate/gopter` - Property-based testing
- `github.com/google/uuid` - UUID generation

## Frontend

- **Framework**: React 19.2 with TypeScript
- **Build Tool**: Vite 7.2
- **Styling**: Tailwind CSS 4.1
- **State Management**: TanStack Query (React Query) 5.90
- **Routing**: React Router DOM 7.13
- **Charts**: Recharts 3.7
- **Icons**: Lucide React 0.563
- **HTTP Client**: Axios 1.13

## Infrastructure

- **Database**: PostgreSQL 15 (Docker for development)
- **Container Orchestration**: Docker Compose for local development
- **Environment Management**: .env files (never committed)

## Common Commands

### Setup & Installation

```bash
# First-time setup (installs dependencies, starts DB)
make setup

# Install dependencies only
make install
```

### Development

```bash
# Start PostgreSQL
make dev-db

# Start backend (loads .env automatically)
make dev-backend

# Start frontend (in separate terminal)
make dev-frontend

# Stop PostgreSQL
make dev-db-stop
```

### Testing

```bash
# Run Go tests (includes property-based tests)
make test

# Test API endpoints
make test-api

# Run frontend tests
cd frontend && npm test

# Lint frontend
cd frontend && npm run lint
```

### Building

```bash
# Build backend binary
make build

# Build frontend for production
cd frontend && npm run build

# Build for multiple platforms
make build-all
```

### Cleanup

```bash
# Remove build artifacts
make clean
```

## Configuration

Backend configuration is loaded from `.env` file in the project root. Copy `.env.example` to `.env` and configure:

- `DATABASE_URL` - PostgreSQL connection string
- `PORT` - API server port (default: 8080)
- `ENCRYPTION_KEY` - 32-byte hex key for credential encryption (generate with `openssl rand -hex 32`)

Frontend configuration is in `frontend/.env` (API base URL).

## API Endpoints

Backend exposes RESTful API on `http://localhost:8080/api` with endpoints for:
- Accounts (`/api/accounts`)
- Transactions (`/api/transactions`)
- Performance (`/api/performance`)
- Fees (`/api/fees`)
- Assets (`/api/assets`)
- Health check (`/health`)

See `docs/API_ENDPOINTS.md` for complete API documentation.
