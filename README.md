# Valhafin 🔥⚔️

**Your Financial Valhalla**

*Where wealth warriors ascend*

Application web de gestion de portefeuille financier qui permet de connecter des comptes sur différentes plateformes d'investissement (Trade Republic, Binance, Bourse Direct), de télécharger automatiquement l'historique des transactions, et de visualiser les performances financières à travers des graphiques et des métriques détaillées.

Named after Valhalla, the hall of slain heroes in Norse mythology - your ultimate destination for financial glory.

## Architecture

Valhafin est composé de deux parties principales :

- **Backend Go** : API RESTful qui gère les scrapers, la base de données PostgreSQL, et la récupération des prix des actifs
- **Frontend React** : Interface utilisateur moderne avec thème sombre, construite avec React, TypeScript et Tailwind CSS

## Fonctionnalités

- 🔐 Connexion sécurisée aux comptes financiers (Trade Republic, Binance, Bourse Direct)
- 📊 Synchronisation automatique des transactions
- 📈 Visualisation des performances avec graphiques interactifs
- 💰 Métriques détaillées sur les frais
- 🎨 Interface moderne avec thème sombre et touches de bleu
- 📱 Design responsive (desktop, tablette, mobile)
- 🔄 Mise à jour automatique des prix des actifs
- 📥 Import de données CSV

## Démarrage Rapide

### Prérequis

- Go 1.21+
- Node.js 20+
- PostgreSQL 15+ (ou Docker)
- Make (optionnel, mais recommandé)

### Installation Rapide

```bash
# 1. Cloner le repo
git clone https://github.com/your-org/valhafin.git
cd valhafin

# 2. Installer les dépendances
make setup

# 3. Copier et configurer .env
cp .env.example .env
# Éditer .env avec vos valeurs (voir ci-dessous)

# 4. Démarrer PostgreSQL
make dev-db

# 5. Démarrer le backend
make dev-backend

# 6. Dans un autre terminal, démarrer le frontend
make dev-frontend
```

### Configuration (.env)

Le backend charge automatiquement le fichier `.env` au démarrage. Créez-le à partir de `.env.example`:

```bash
cp .env.example .env
```

Générer une clé de chiffrement sécurisée (32 bytes en hexadécimal):

```bash
# Avec OpenSSL
openssl rand -hex 32
```

Éditer `.env` avec vos configurations:

```env
DATABASE_URL=postgresql://valhafin:valhafin_dev_password@localhost:5432/valhafin_dev?sslmode=disable
PORT=8080
ENCRYPTION_KEY=your_generated_32_byte_hex_key_here
```

**Important:** Le fichier `.env` est ignoré par git et ne doit JAMAIS être commité.

### Démarrage Manuel (sans Make)

#### 1. Base de données

```bash
docker-compose -f docker-compose.dev.yml up -d
```

#### 2. Backend

```bash
# Le .env est chargé automatiquement
go run main.go
```

Le serveur API sera accessible sur http://localhost:8080

#### 3. Frontend

```bash
cd frontend
npm install
npm run dev
```

Le frontend sera accessible sur http://localhost:5173

### Vérifier que tout fonctionne

```bash
# Health check
curl http://localhost:8080/health

# Tester tous les endpoints
make test-api
```

## Structure du Projet

```
valhafin/
├── main.go                    # Point d'entrée du serveur API
├── internal/                  # Code privé de l'application
│   ├── api/                   # HTTP handlers, routes, middleware, validation
│   ├── domain/
│   │   └── models/            # Modèles métier (Account, Asset, Transaction, etc.)
│   ├── repository/
│   │   └── database/          # Couche d'accès aux données PostgreSQL
│   ├── service/
│   │   ├── encryption/        # Service de chiffrement AES-256-GCM
│   │   └── scraper/           # Scrapers pour chaque plateforme
│   │       ├── traderepublic/
│   │       ├── binance/
│   │       └── boursedirect/
│   ├── config/                # Configuration de l'application
│   └── utils/                 # Fonctions utilitaires
└── frontend/                  # Application React
    ├── src/
    │   ├── components/        # Composants React
    │   ├── pages/             # Pages
    │   ├── services/          # Services API
    │   ├── hooks/             # Hooks personnalisés
    │   └── types/             # Types TypeScript
    └── package.json
```

**Note**: Le dossier `internal/` suit la convention Go pour le code privé qui ne peut pas être importé par d'autres projets.

## Développement

### Commandes Make

```bash
# Démarrer PostgreSQL
make dev-db

# Démarrer le backend (charge automatiquement .env)
make dev-backend

# Démarrer le frontend
make dev-frontend

# Tester l'API
make test-api

# Lancer les tests Go
make test

# Arrêter PostgreSQL
make dev-db-stop

# Nettoyer les artifacts de build
make clean
```

### Backend

```bash
# Lancer les tests
go test ./...

# Build
go build -o valhafin

# Lancer avec logs
go run main.go
```

### Frontend

```bash
cd frontend

# Lancer les tests
npm test

# Linting
npm run lint

# Build de production
npm run build
```

## API Endpoints

Le backend expose une API RESTful complète:

**Comptes:**
- `POST /api/accounts` - Créer un compte
- `GET /api/accounts` - Lister les comptes
- `GET /api/accounts/:id` - Détails d'un compte
- `DELETE /api/accounts/:id` - Supprimer un compte
- `POST /api/accounts/:id/sync` - Synchroniser un compte

**Transactions:**
- `GET /api/accounts/:id/transactions` - Transactions d'un compte
- `GET /api/transactions` - Toutes les transactions
- `POST /api/transactions/import` - Importer depuis CSV

**Performance:**
- `GET /api/accounts/:id/performance` - Performance d'un compte
- `GET /api/performance` - Performance globale
- `GET /api/assets/:isin/performance` - Performance d'un actif

**Frais:**
- `GET /api/accounts/:id/fees` - Frais d'un compte
- `GET /api/fees` - Frais globaux

**Prix:**
- `GET /api/assets/:isin/price` - Prix actuel d'un actif
- `GET /api/assets/:isin/history` - Historique des prix

**Health:**
- `GET /health` - État de l'application

## Plateformes Supportées

- ✅ Trade Republic (scraper fonctionnel)
- 🚧 Binance (en développement)
- 🚧 Bourse Direct (en développement)

## Documentation

### Guides de Démarrage
- **[Guide Simplifié](docs/SIMPLE_STARTUP_GUIDE.md)** - Démarrage rapide en 3 commandes
- **[Guide Complet](docs/BACKEND_STARTUP_GUIDE.md)** - Guide détaillé du backend
- **[FAQ](docs/FAQ_BACKEND_STARTUP.md)** - Questions fréquentes

### Déploiement
- **[Production](docs/PRODUCTION_DEPLOYMENT.md)** - Guide de déploiement en production
- **[Docker & CI/CD](docs/PRODUCTION_DEPLOYMENT.md#méthodes-de-déploiement)** - Déploiement avec Docker, Kubernetes, etc.

### Architecture
- **[Spécifications](.kiro/specs/portfolio-web-app/)** - Exigences et design complet
- **[Documentation Frontend](frontend/README.md)** - Guide du frontend React
- **[Index Documentation](docs/README.md)** - Toute la documentation

### Résumés des Tâches
- Consultez le dossier [docs/](docs/) pour les résumés détaillés de chaque tâche implémentée

## License

MIT
