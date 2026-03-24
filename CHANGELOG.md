# Changelog

All notable changes to the Valhafin project are documented in this file.

## [v1.0.5] - 2026-03-24

### Added
- AWS WAF token retrieval via headless browser (`go-rod/rod`) to bypass Trade Republic's new anti-bot protection
- Device info generation (SHA-512 device ID, base64-encoded) matching Trade Republic's expected format
- New `waf.go` file with `fetchWAFToken()` and `generateDeviceInfo()`
- `InitWAF()` method on Trade Republic scraper (lazy-loaded automatically on first authentication)
- `setTRHeaders()` helper applying all required headers to every TR API request

### Changed
- Trade Republic authentication requests now include full browser-like headers: `Accept`, `Accept-Language`, `Cache-Control`, `Pragma`, `x-aws-waf-token`, `x-tr-app-version`, `x-tr-device-info`, `x-tr-platform`
- `Scraper` struct now stores `wafToken` and `deviceInfo` fields

### Dependencies
- Added `github.com/go-rod/rod` for headless browser automation

### Fixed
- Trade Republic sync now surfaces real API error responses (HTTP status + body) instead of generic messages
- Sync error details displayed in frontend account card instead of generic "Erreur lors de la synchronisation"
- Replace `as any` with explicit type in AccountCard sync error handling (eslint `no-explicit-any`)

## [v1.0.4] - 2026-03-16

### Added
- Swagger/OpenAPI documentation for all 25 API endpoints
- Swagger UI accessible at `/swagger/index.html`
- `make swagger` target to regenerate documentation

### Changed
- Refactored `handlers.go` (~2200 lines) into domain-specific files: `handler.go`, `handler_health.go`, `handler_accounts.go`, `handler_sync.go`, `handler_transactions.go`, `handler_performance.go`, `handler_fees.go`, `handler_assets.go`

## [v1.0.3] - 2026-03-07

### Added
- Application title and SVG icon

### Fixed
- Nginx SPA routing to avoid directory listing on `/assets` route

## [v1.0.2] - 2026-03-07

### Changed
- Frontend API URL now uses relative URLs (`/api` in prod, `localhost:8080/api` in dev via env files)

### Fixed
- Frontend linting

### Updated
- codecov/codecov-action v4 → v5
- actions/upload-artifact v4 → v6
- actions/checkout v4 → v6
- actions/cache v4 → v5
- actions/download-artifact v4 → v7
- lucide-react (frontend)
- github.com/lib/pq v1.11.1 → v1.11.2
- @types/node (frontend)

## [v1.0.1] - 2026-02-12

### Fixed
- Docker image tag
- Image path in `docker-compose.yml`
- Docker Compose pull permissions
- Repo name and Docker Compose command in workflow
- `schema_migrations` table created last instead of first
- Missing dev dependencies in frontend Dockerfile

### Added
- Backend build and start job in tests workflow
- Backend container error check step in Docker workflow
- `ENCRYPTION_KEY` environment variable in Docker build workflow
- Pull policy in Docker workflow

## [v1.0.0] - 2026-02-12

### Added
- Go backend with REST API (29 endpoints)
- React 19 + TypeScript + Tailwind CSS frontend
- PostgreSQL database with automatic migrations
- AES-256-GCM credential encryption
- Trade Republic scraper with WebSocket authentication and 2FA
- Yahoo Finance price service with cache and automatic symbol resolution
- Scheduler for periodic price updates
- Performance calculation per account, global and per asset
- Fee metrics per account and global
- CSV transaction import with deduplication
- Dashboard tab with charts and latest buys/sells
- Performance tab with evolution charts
- Assets tab with price history and diagrams (1M, 5Y, MAX)
- Fees tab with evolution chart
- Yahoo Finance symbol search modal
- Animations and responsive design
- Dockerfiles and Docker Compose for production
- CI/CD workflows: tests, security, Docker build, release
- Dependabot for dependency tracking
- API endpoints documentation
- README badges

### Fixed
- Performance calculation in diagrams
- Realized and unrealized gains calculation
- Asset quantities in transactions tab
- Cache issue when refreshing prices
- Buy point positioning in diagrams
- Asset price verification in list
- Yahoo Finance price fetching issue
- CORS issue when adding an account
- Frontend linting
