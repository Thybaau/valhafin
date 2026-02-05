# Guide de Démarrage Simplifié

## TL;DR

```bash
# 1. Démarrer PostgreSQL
make dev-db

# 2. Démarrer le backend (charge automatiquement .env)
make dev-backend

# 3. Tester
curl http://localhost:8080/health
```

C'est tout! 🎉

## Première Installation

```bash
# 1. Cloner et installer
git clone https://github.com/your-org/valhafin.git
cd valhafin
make setup

# 2. Configurer .env
cp .env.example .env
# Éditer .env avec vos valeurs

# 3. Démarrer
make dev-db
make dev-backend
```

## Configuration (.env)

Le fichier `.env` doit être à la racine du projet:

```bash
# Base de données
DATABASE_URL=postgresql://valhafin:valhafin@localhost:5432/valhafin_dev?sslmode=disable

# Port du serveur
PORT=8080

# Clé de chiffrement (générer avec: openssl rand -hex 32)
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

**Important:**
- ✅ Le fichier `.env` est ignoré par git
- ✅ Utilisez `.env.example` comme template
- ✅ Ne commitez JAMAIS le fichier `.env`

## Comment ça marche?

Le backend charge automatiquement le fichier `.env` au démarrage:

```go
// Dans main.go
func main() {
    // Charge .env si il existe
    _ = godotenv.Load()
    
    // Le reste du code...
}
```

**En développement:** Le `.env` est chargé automatiquement  
**En production:** Les variables sont gérées par le système (Docker, K8s, etc.)

## Commandes Utiles

```bash
# Démarrer
make dev-db              # PostgreSQL
make dev-backend         # Backend
make dev-frontend        # Frontend

# Tester
make test-api            # Tester tous les endpoints
make test                # Tests Go

# Arrêter
make dev-db-stop         # PostgreSQL
# Backend: Ctrl+C
```

## Vérifier que ça fonctionne

```bash
curl http://localhost:8080/health
```

**Réponse attendue:**
```json
{
  "status": "healthy",
  "database": "up",
  "version": "dev",
  "uptime": "2m30s"
}
```

## Dépannage

### Erreur: "database URL is empty"
```bash
# Vérifier que .env existe
cat .env

# Copier depuis l'exemple
cp .env.example .env
```

### Erreur: "bind: address already in use"
```bash
# Arrêter le processus sur le port 8080
kill $(lsof -ti:8080)
```

### Erreur: "Failed to connect to database"
```bash
# Démarrer PostgreSQL
make dev-db
```

## Pour Aller Plus Loin

- **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** - Déploiement en production
- **[CHECKPOINT_15_SUMMARY.md](CHECKPOINT_15_SUMMARY.md)** - État actuel du backend
- **[README.md](README.md)** - Index de toute la documentation
