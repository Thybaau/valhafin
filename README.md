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

### 1. Configuration de la base de données

Démarrer PostgreSQL avec Docker Compose :

```bash
docker-compose -f docker-compose.dev.yml up -d
```

### 2. Configuration du backend

Créer un fichier `.env` à partir de `.env.example` :

```bash
cp .env.example .env
```

Générer une clé de chiffrement sécurisée (32 bytes en hexadécimal) :

```bash
# Avec OpenSSL
openssl rand -hex 32

# Ou avec Go
go run -c 'package main; import ("crypto/rand"; "encoding/hex"; "fmt"); func main() { key := make([]byte, 32); rand.Read(key); fmt.Println(hex.EncodeToString(key)) }'
```

Éditer `.env` avec vos configurations :

```env
DATABASE_URL=postgresql://valhafin:valhafin_dev_password@localhost:5432/valhafin_dev?sslmode=disable
PORT=8080
ENCRYPTION_KEY=your_generated_32_byte_hex_key_here
```

Installer les dépendances Go :

```bash
go mod download
```

Démarrer le serveur API :

```bash
go run main.go
```

Le serveur API sera accessible sur http://localhost:8080

**Endpoints disponibles :**
- `GET /health` - Health check
- `POST /api/accounts` - Créer un compte
- `GET /api/accounts` - Lister les comptes
- `GET /api/accounts/:id` - Détails d'un compte
- `DELETE /api/accounts/:id` - Supprimer un compte

### 3. Configuration du frontend

Installer les dépendances :

```bash
cd frontend
npm install
```

Démarrer le serveur de développement :

```bash
npm run dev
```

Le frontend sera accessible sur http://localhost:5173

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

### Backend

```bash
# Lancer les tests
go test ./...

# Build
go build -o valhafin
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

## Plateformes Supportées

- ✅ Trade Republic (scraper fonctionnel)
- 🚧 Binance (en développement)
- 🚧 Bourse Direct (en développement)

## Documentation

Pour plus de détails sur l'architecture et le design, consultez :

- [Spécifications](.kiro/specs/portfolio-web-app/)
- [Documentation Frontend](frontend/README.md)

## License

MIT
