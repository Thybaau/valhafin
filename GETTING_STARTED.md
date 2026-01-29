# Getting Started with Valhafin

Ce guide vous aidera à configurer et démarrer l'application Valhafin pour le développement.

## Prérequis

Avant de commencer, assurez-vous d'avoir installé :

- **Go 1.21+** : [Installation Go](https://golang.org/doc/install)
- **Node.js 20+** : [Installation Node.js](https://nodejs.org/)
- **Docker** : [Installation Docker](https://docs.docker.com/get-docker/) (pour PostgreSQL)
- **Make** : Généralement préinstallé sur macOS/Linux

## Installation Rapide

### 1. Cloner le projet

```bash
git clone <repository-url>
cd valhafin
```

### 2. Configuration initiale

Exécutez la commande de setup qui installe toutes les dépendances et démarre PostgreSQL :

```bash
make setup
```

Cette commande va :
- Installer les dépendances Go
- Installer les dépendances npm du frontend
- Démarrer PostgreSQL avec Docker Compose

### 3. Configuration de l'environnement

Créez un fichier `.env` à partir de l'exemple :

```bash
cp .env.example .env
```

Éditez `.env` et configurez les variables nécessaires :

```env
# Base de données
DATABASE_URL=postgresql://valhafin:valhafin_dev_password@localhost:5432/valhafin_dev?sslmode=disable

# Serveur
PORT=8080

# Clé de chiffrement (générez une clé aléatoire de 64 caractères hexadécimaux)
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

# API Yahoo Finance (optionnel)
YAHOO_FINANCE_API_KEY=

# Identifiants Trade Republic (pour le scraping)
TR_PHONE_NUMBER=+33XXXXXXXXX
TR_PIN=XXXX
```

### 4. Démarrer l'application

#### Option A : Démarrage manuel (recommandé pour le développement)

Dans un terminal, démarrez le backend :

```bash
make dev-backend
```

Dans un autre terminal, démarrez le frontend :

```bash
make dev-frontend
```

#### Option B : Utiliser le Makefile

```bash
# Démarrer PostgreSQL
make dev-db

# Dans un autre terminal
make dev-backend

# Dans un troisième terminal
make dev-frontend
```

### 5. Accéder à l'application

- **Frontend** : http://localhost:5173
- **Backend API** : http://localhost:8080
- **Health Check** : http://localhost:8080/health

## Structure du Projet

```
valhafin/
├── api/                       # API REST (handlers, middleware, routes)
├── config/                    # Configuration (support .env et config.yaml)
├── database/                  # Couche d'accès aux données (à créer)
├── models/                    # Modèles de données
├── scrapers/                  # Scrapers pour chaque plateforme
├── services/                  # Services métier (à créer)
├── utils/                     # Utilitaires
├── frontend/                  # Application React
│   ├── src/
│   │   ├── components/       # Composants React
│   │   ├── services/         # Services API
│   │   ├── types/            # Types TypeScript
│   │   └── App.tsx           # Composant principal
│   └── package.json
├── docker-compose.dev.yml     # PostgreSQL pour développement
├── .env.example               # Exemple de configuration
└── Makefile                   # Commandes de développement
```

## Commandes Utiles

### Backend

```bash
# Installer les dépendances
go mod download

# Lancer les tests
go test ./...

# Build
go build -o valhafin

# Lancer l'application
./valhafin
```

### Frontend

```bash
cd frontend

# Installer les dépendances
npm install

# Démarrer le serveur de développement
npm run dev

# Lancer les tests
npm test

# Linting
npm run lint

# Build de production
npm run build
```

### Base de données

```bash
# Démarrer PostgreSQL
make dev-db

# Arrêter PostgreSQL
make dev-db-stop

# Se connecter à PostgreSQL
docker exec -it valhafin-postgres-dev psql -U valhafin -d valhafin_dev
```

## Développement

### Workflow de développement

1. **Backend** : Modifiez les fichiers Go, le serveur redémarre automatiquement avec `go run`
2. **Frontend** : Modifiez les fichiers React, Vite recharge automatiquement le navigateur
3. **Base de données** : Les migrations seront créées dans les prochaines tâches

### Ajouter une nouvelle route API

1. Définir le handler dans `api/handlers.go`
2. Ajouter la route dans `api/routes.go`
3. Créer le service correspondant dans `services/`
4. Créer les fonctions de base de données dans `database/`

### Ajouter un nouveau composant frontend

1. Créer le composant dans `frontend/src/components/`
2. Créer les types dans `frontend/src/types/`
3. Créer le service API dans `frontend/src/services/`
4. Créer le hook personnalisé dans `frontend/src/hooks/`

## Prochaines Étapes

Maintenant que l'infrastructure de base est configurée, vous pouvez :

1. **Tâche 2** : Créer les modèles de données et migrations de base de données
2. **Tâche 3** : Implémenter le service de chiffrement
3. **Tâche 4** : Développer l'API REST pour la gestion des comptes

Consultez le fichier `.kiro/specs/portfolio-web-app/tasks.md` pour la liste complète des tâches.

## Dépannage

### PostgreSQL ne démarre pas

```bash
# Vérifier les logs
docker-compose -f docker-compose.dev.yml logs postgres

# Redémarrer
docker-compose -f docker-compose.dev.yml restart postgres
```

### Le frontend ne se connecte pas au backend

Vérifiez que :
- Le backend est démarré sur le port 8080
- Le CORS est configuré correctement dans `api/middleware.go`
- L'URL de l'API est correcte dans `frontend/src/services/api.ts`

### Erreur de dépendances Go

```bash
go mod tidy
go mod download
```

### Erreur de dépendances npm

```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
```

## Support

Pour toute question ou problème, consultez :
- [Documentation du projet](.kiro/specs/portfolio-web-app/)
- [Issues GitHub](lien-vers-issues)
