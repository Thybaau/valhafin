# Valhafin Frontend

Interface web moderne pour l'application de gestion de portefeuille financier Valhafin.

## Stack Technologique

- **React 18** avec TypeScript
- **Vite** - Build tool rapide et moderne
- **Tailwind CSS** - Framework CSS avec thème sombre personnalisé
- **React Router** - Navigation
- **TanStack Query (React Query)** - Gestion des données et cache
- **Axios** - Client HTTP
- **Recharts** - Bibliothèque de graphiques (à installer)

## Thème Visuel

L'application utilise un thème sombre avec des touches de bleu :

- **Fond** : Noir profond (#0a0a0a), gris foncé (#1a1a1a), gris anthracite (#2a2a2a)
- **Texte** : Blanc (#ffffff), gris clair (#b0b0b0), gris moyen (#6b6b6b)
- **Accent** : Bleu (#3b82f6, #2563eb, #60a5fa)
- **Succès** : Vert (#10b981)
- **Erreur** : Rouge (#ef4444)
- **Avertissement** : Orange (#f59e0b)

## Développement

### Installation

```bash
npm install
```

### Démarrage du serveur de développement

```bash
npm run dev
```

L'application sera accessible sur http://localhost:5173

### Build de production

```bash
npm run build
```

### Tests

```bash
npm test
```

### Linting

```bash
npm run lint
```

## Structure du Projet

```
src/
├── components/       # Composants React
│   ├── common/      # Composants réutilisables
│   ├── Layout/      # Layout et navigation
│   ├── Accounts/    # Gestion des comptes
│   ├── Transactions/# Liste des transactions
│   ├── Performance/ # Graphiques de performance
│   └── Fees/        # Métriques de frais
├── pages/           # Pages de l'application
├── services/        # Services API
├── hooks/           # Hooks personnalisés
├── types/           # Types TypeScript
└── styles/          # Styles globaux
```

## Configuration

Le frontend communique avec le backend via l'API REST sur http://localhost:8080/api

Pour changer l'URL de l'API, créez un fichier `.env.local` :

```
VITE_API_URL=http://votre-api-url/api
```
