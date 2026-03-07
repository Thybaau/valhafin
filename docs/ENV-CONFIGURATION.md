# Configuration des variables d'environnement

## Comment Vite gère les fichiers .env

Vite charge les fichiers d'environnement dans cet ordre (priorité décroissante) :

```
1. .env.[mode].local      (ex: .env.production.local) - Overrides locaux, NON commités
2. .env.[mode]            (ex: .env.production)       - Config spécifique au mode
3. .env.local             - Overrides locaux, NON commités
4. .env                   - Configuration par défaut
```

### Modes Vite

- **`development`** : Utilisé par `npm run dev`
- **`production`** : Utilisé par `npm run build`

## Fichiers du projet

### `.env` (commité)
```bash
VITE_API_URL=/api
```
Configuration par défaut, utilisée comme fallback.

### `.env.development` (commité)
```bash
VITE_API_URL=http://localhost:8080/api
```
Utilisé en développement local (`npm run dev`).
Permet de se connecter au backend local.

### `.env.production` (commité)
```bash
VITE_API_URL=/api
```
Utilisé lors du build production (`npm run build`).
URLs relatives pour fonctionner avec l'Ingress Kubernetes.

### `.env.example` (commité)
Documentation de la configuration disponible.

### `.env.local` (NON commité)
Overrides personnels, ignoré par git.
Utile pour tester des configurations locales sans modifier les fichiers commités.

## Utilisation lors du build Docker

### Processus de build

```dockerfile
# 1. Copie du code source (incluant les fichiers .env)
COPY . ./

# 2. Build de l'application
RUN npm run build
```

### Ce qui se passe

1. **`npm run build`** exécute Vite en mode `production`
2. Vite charge les fichiers dans cet ordre :
   - `.env.production.local` (si existe, non commité)
   - **`.env.production`** ✅ (utilisé)
   - `.env.local` (si existe, non commité)
   - `.env` (fallback)
3. Vite remplace `import.meta.env.VITE_API_URL` par la valeur `/api`
4. Le code JavaScript généré contient la valeur hardcodée `/api`
5. Les fichiers sont copiés dans l'image nginx

### Résultat

Le JavaScript final dans l'image Docker contient :
```javascript
const API_BASE_URL = "/api" || '/api'
```

**Important** : Les variables d'environnement sont injectées au **build time**, pas au runtime !

## Scénarios d'utilisation

### 1. Développement local

```bash
cd frontend
npm run dev
```

**Fichier utilisé** : `.env.development`
**URL API** : `http://localhost:8080/api`
**Comportement** : Connexion directe au backend local

### 2. Build local pour tester

```bash
cd frontend
npm run build
npm run preview
```

**Fichier utilisé** : `.env.production`
**URL API** : `/api`
**Comportement** : URLs relatives (nécessite un proxy ou Ingress)

### 3. Build Docker (CI/CD)

```bash
docker build -t valhafin-frontend .
```

**Fichier utilisé** : `.env.production`
**URL API** : `/api`
**Comportement** : URLs relatives pour Kubernetes

### 4. Override local temporaire

Créer `.env.local` :
```bash
VITE_API_URL=http://192.168.1.100:8080/api
```

Ce fichier n'est pas commité et a la priorité sur les autres.

## Build Docker avec variable personnalisée

Si tu veux builder avec une URL différente :

### Option 1 : Build argument (nécessite modification du Dockerfile)

```dockerfile
# Dans le Dockerfile
ARG VITE_API_URL=/api
ENV VITE_API_URL=$VITE_API_URL

# Build
docker build --build-arg VITE_API_URL=http://custom.api/api -t valhafin-frontend .
```

### Option 2 : Fichier .env.production.local (recommandé)

```bash
# Créer un fichier temporaire
echo "VITE_API_URL=http://custom.api/api" > .env.production.local

# Builder
docker build -t valhafin-frontend .

# Nettoyer
rm .env.production.local
```

### Option 3 : Utiliser notre configuration actuelle (recommandé)

Ne rien changer ! L'URL relative `/api` fonctionne avec n'importe quel domaine grâce à l'Ingress.

## Vérification

### Vérifier quelle variable est utilisée

```bash
# En développement
npm run dev
# Regarde les logs de la console : [API] GET /...

# En production (après build)
npm run build
grep -r "VITE_API_URL" dist/  # Ne devrait rien trouver (remplacé par la valeur)
```

### Vérifier dans l'image Docker

```bash
# Builder l'image
docker build -t valhafin-frontend .

# Extraire et inspecter le JavaScript
docker run --rm valhafin-frontend cat /usr/share/nginx/html/assets/index-*.js | grep -o "http://[^\"']*api"
```

## Bonnes pratiques

✅ **À faire**
- Commiter `.env`, `.env.development`, `.env.production`, `.env.example`
- Utiliser `.env.local` pour les overrides personnels
- Documenter les variables dans `.env.example`
- Utiliser des URLs relatives en production

❌ **À éviter**
- Commiter `.env.local` ou `.env.*.local`
- Mettre des secrets dans les fichiers `.env` (ils sont dans le code JavaScript final !)
- Essayer d'injecter des variables au runtime (Vite ne le supporte pas)
- Hardcoder des URLs dans le code

## Résumé

| Fichier | Commité | Utilisé par | Priorité | Usage |
|---------|---------|-------------|----------|-------|
| `.env` | ✅ | Tous | 4 (plus basse) | Fallback par défaut |
| `.env.development` | ✅ | `npm run dev` | 2 | Dev local |
| `.env.production` | ✅ | `npm run build` | 2 | Build Docker/CI |
| `.env.local` | ❌ | Tous | 3 | Overrides perso |
| `.env.development.local` | ❌ | `npm run dev` | 1 (plus haute) | Overrides dev perso |
| `.env.production.local` | ❌ | `npm run build` | 1 (plus haute) | Overrides build perso |
| `.env.example` | ✅ | Documentation | - | Template |

**Notre configuration actuelle est optimale pour Kubernetes !** 🎉
