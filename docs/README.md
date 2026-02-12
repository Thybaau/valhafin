# Documentation Valhafin

Documentation complète du projet Valhafin - Application web de gestion de portefeuille financier.

## 📚 Guides Principaux

| Guide | Description |
|-------|-------------|
| **[SIMPLE_STARTUP_GUIDE.md](SIMPLE_STARTUP_GUIDE.md)** | 🚀 Démarrage rapide en 3 étapes |
| **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)** | 👨‍💻 Architecture, conventions, tests |
| **[API_ENDPOINTS.md](API_ENDPOINTS.md)** | 📡 Documentation des 21 endpoints REST |
| **[PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md)** | 🏭 Déploiement Docker, CI/CD, releases |

## ⚡ Démarrage Rapide

```bash
# 1. Installation
make setup
cp .env.example .env
# Éditer .env et ajouter ENCRYPTION_KEY (openssl rand -hex 32)

# 2. Lancement
make dev-db        # Terminal 1: PostgreSQL
make dev-backend   # Terminal 2: Backend
make dev-frontend  # Terminal 3: Frontend

# 3. Vérification
curl http://localhost:8080/health
```

**URLs:**
- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

## 🎯 État du Projet

**Version**: v1.0.0 - Production Ready ✅

| Composant | Statut | Tests |
|-----------|--------|-------|
| Backend API | ✅ Complet | 21/21 endpoints |
| Frontend React | ✅ Complet | Interface responsive |
| Docker & CI/CD | ✅ Complet | GitHub Actions |
| Documentation | ✅ Complète | 4 guides principaux |

## 🔧 Commandes Principales

```bash
# Développement
make dev-db / dev-backend / dev-frontend
make test / test-api
cd frontend && npm test

# Build & Déploiement
make build / build-all
cd frontend && npm run build
docker-compose up -d

# Nettoyage
make clean
```

## 📁 Structure

```
docs/
├── README.md                    # Index (ce fichier)
├── SIMPLE_STARTUP_GUIDE.md     # Démarrage rapide
├── DEVELOPER_GUIDE.md          # Guide développeur complet
├── API_ENDPOINTS.md            # Documentation API REST
└── PRODUCTION_DEPLOYMENT.md    # Déploiement production
```

## 🔗 Liens Utiles

- [Spécifications complètes](../.kiro/specs/portfolio-web-app/)
- [README principal](../README.md)
- [Code source](../)

---

**Dernière mise à jour:** 2026-02-10
