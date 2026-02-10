# Valhafin 🔥⚔️

**Your Financial Valhalla** - *Where wealth warriors ascend*

Application web de gestion de portefeuille financier qui connecte vos comptes d'investissement (Trade Republic, Binance, Bourse Direct), synchronise automatiquement vos transactions, et visualise vos performances financières avec des graphiques interactifs.

## ⚡ Fonctionnalités

- 🔐 **Sécurité** - Chiffrement AES-256-GCM des credentials
- 📊 **Synchronisation** - Import automatique des transactions
- 📈 **Performance** - Graphiques interactifs d'évolution du portefeuille
- 💰 **Analyse** - Métriques détaillées sur les frais et gains/pertes
- 🔄 **Temps réel** - Mise à jour automatique des prix via Yahoo Finance

## 🏦 Plateformes Supportées

- ✅ Trade Republic (scraper fonctionnel)
- 🚧 Binance (en développement)
- 🚧 Bourse Direct (en développement)

## 📡 API REST

29 endpoints disponibles - [Documentation complète](docs/API_ENDPOINTS.md)

### Endpoints par catégorie

#### 🏦 Gestion des Comptes
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/accounts` | GET | Lister tous les comptes |
| `/api/accounts` | POST | Créer un nouveau compte |
| `/api/accounts/:id` | GET | Détails d'un compte |
| `/api/accounts/:id` | DELETE | Supprimer un compte |
| `/api/accounts/:id/sync` | POST | Synchroniser un compte (Binance, Bourse Direct) |
| `/api/accounts/:id/sync/init` | POST | Initier sync Trade Republic (2FA) |
| `/api/accounts/:id/sync/complete` | POST | Compléter sync Trade Republic avec code 2FA |

#### 💸 Transactions
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/accounts/:id/transactions` | GET | Transactions d'un compte (filtres, pagination) |
| `/api/transactions` | GET | Toutes les transactions (tous comptes) |
| `/api/transactions/:id` | PUT | Modifier une transaction |
| `/api/transactions/import` | POST | Importer transactions depuis CSV |

#### 📈 Performance
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/performance` | GET | Performance globale (tous comptes) |
| `/api/accounts/:id/performance` | GET | Performance d'un compte |
| `/api/assets/:isin/performance` | GET | Performance d'un actif spécifique |

#### 💰 Frais
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/fees` | GET | Métriques de frais globales |
| `/api/accounts/:id/fees` | GET | Métriques de frais par compte |

#### 📊 Actifs & Prix
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/assets` | GET | Liste des actifs avec positions |
| `/api/assets/:isin/price` | GET | Prix actuel d'un actif |
| `/api/assets/:isin/history` | GET | Historique des prix |
| `/api/assets/:isin/price/update` | POST | Forcer mise à jour du prix (admin) |
| `/api/assets/:isin/price/refresh` | POST | Rafraîchir le prix d'un actif |
| `/api/assets/:isin/symbol` | PUT | Mettre à jour le symbole d'un actif |
| `/api/assets/symbols/resolve` | POST | Résoudre tous les symboles manquants |

#### 🔍 Recherche de Symboles
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/api/symbols/search` | GET | Rechercher un symbole boursier |

#### 🏥 Monitoring
| Endpoint | Méthode | Description |
|----------|---------|-------------|
| `/health` | GET | État de santé de l'application |

## 🏗️ Architecture

**Backend Go** - API RESTful avec PostgreSQL
- Scrapers pour Trade Republic, Binance, Bourse Direct
- Service de chiffrement et gestion sécurisée des credentials
- Calcul de performance et analyse des frais
- Scheduler pour mises à jour automatiques

**Frontend React** - Interface utilisateur moderne
- React 19 + TypeScript + Tailwind CSS
- TanStack Query pour la gestion d'état
- Recharts pour les graphiques
- Design responsive mobile-first

**Base de Données PostgreSQL** - 7 tables principales
- `accounts` - Comptes financiers connectés
- `assets` - Catalogue des actifs (actions, ETF, crypto)
- `asset_prices` - Historique des prix
- `transactions_*` - Transactions par plateforme (Trade Republic, Binance, Bourse Direct)
- [Documentation complète du schéma](docs/DATABASE_SCHEMA.md)

## 🚀 Démarrage Rapide

### Installation via Release

Déploiement rapide avec Docker Compose à partir d'une release GitHub:

```bash
# 1. Télécharger la dernière release
wget https://github.com/your-org/valhafin/releases/latest/download/valhafin-latest.tar.gz
tar -xzf valhafin-latest.tar.gz
cd valhafin

# 2. Configurer l'environnement
cp .env.example .env
# Générer les secrets
openssl rand -hex 32  # Copier dans ENCRYPTION_KEY
openssl rand -base64 32  # Copier dans POSTGRES_PASSWORD
# Éditer .env avec vos valeurs

# 3. Déployer avec Docker Compose
chmod +x deploy.sh
./deploy.sh
```

**Accès:**
- Frontend: http://localhost:80
- Backend API: http://localhost:8080

### Vérification

```bash
curl http://localhost:8080/health
# Réponse: {"status":"healthy","database":"connected"}
```

## 📚 Documentation

- **[Guide de Démarrage](docs/SIMPLE_STARTUP_GUIDE.md)** - Installation et configuration
- **[Guide Développeur](docs/DEVELOPER_GUIDE.md)** - Architecture, conventions, tests
- **[API Reference](docs/API_ENDPOINTS.md)** - Documentation des 29 endpoints
- **[Schéma Base de Données](docs/DATABASE_SCHEMA.md)** - Tables PostgreSQL et relations
- **[Déploiement Production](docs/PRODUCTION_DEPLOYMENT.md)** - Docker, CI/CD, releases

## 📁 Structure du Projet

```
valhafin/
├── main.go                    # Point d'entrée
├── internal/                  # Backend Go
│   ├── api/                   # Handlers HTTP, routes, middleware
│   ├── domain/models/         # Modèles métier
│   ├── repository/database/   # Accès PostgreSQL
│   └── service/               # Logique métier
│       ├── encryption/        # Chiffrement AES-256-GCM
│       ├── scraper/           # Trade Republic, Binance, Bourse Direct
│       ├── price/             # Yahoo Finance
│       ├── performance/       # Calculs de performance
│       └── scheduler/         # Tâches automatiques
└── frontend/                  # Frontend React
    └── src/
        ├── components/        # Composants UI
        ├── pages/             # Pages de l'app
        ├── services/          # Client API
        └── hooks/             # React Query hooks
```

## 🛠️ Développement

### Prérequis

- Go 1.21+
- Node.js 20+
- Docker & Docker Compose
- Make (recommandé)

### Installation

Installation complète avec code source pour développer:

```bash
# 1. Cloner et installer
git clone https://github.com/your-org/valhafin.git
cd valhafin
make setup

# 2. Configurer l'environnement
cp .env.example .env
openssl rand -hex 32  # Copier dans ENCRYPTION_KEY
# Éditer .env avec la clé générée

# 3. Démarrer l'application
make dev-db        # Terminal 1: PostgreSQL
make dev-backend   # Terminal 2: Backend (http://localhost:8080)
make dev-frontend  # Terminal 3: Frontend (http://localhost:5173)
```

### Commandes principales

```bash
# Développement
make dev-db          # Démarrer PostgreSQL
make dev-backend     # Démarrer le backend
make dev-frontend    # Démarrer le frontend

# Tests
make test            # Tests Go
make test-api        # Tests API endpoints
cd frontend && npm test  # Tests React

# Build
make build           # Compiler le backend
cd frontend && npm run build  # Compiler le frontend

# Nettoyage
make clean           # Supprimer les artifacts
```

## 📄 License

MIT
