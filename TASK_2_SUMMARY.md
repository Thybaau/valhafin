# Tâche 2 : Modèles de données et migrations de base de données

## ✅ Complété

Cette tâche a mis en place les modèles de données Go, le système de migrations PostgreSQL, et la couche d'accès aux données pour l'application Valhafin.

## Ce qui a été fait

### 1. Modèles Go (Subtask 2.1)

#### ✅ models/account.go
- Modèle `Account` représentant un compte financier sur une plateforme
- Champs :
  - `ID` : UUID unique
  - `Name` : Nom du compte
  - `Platform` : Plateforme (traderepublic, binance, boursedirect)
  - `Credentials` : Identifiants chiffrés (non exposés en JSON)
  - `CreatedAt`, `UpdatedAt`, `LastSync` : Timestamps
- Méthode `Validate()` :
  - Vérifie que le nom, la plateforme et les credentials sont présents
  - Valide que la plateforme est l'une des trois supportées
  - Retourne des erreurs descriptives

#### ✅ models/asset.go
- Modèle `Asset` représentant un actif financier (action, ETF, crypto)
- Champs :
  - `ISIN` : Code d'identification international (clé primaire)
  - `Name` : Nom de l'actif
  - `Symbol` : Symbole boursier
  - `Type` : Type d'actif (stock, etf, crypto)
  - `Currency` : Devise (format ISO 4217)
  - `LastUpdated` : Date de dernière mise à jour
- Méthode `Validate()` :
  - Vérifie le format ISIN (2 lettres + 10 alphanumériques)
  - Valide le type d'actif
  - Vérifie le format de la devise (3 lettres majuscules)

#### ✅ models/asset_price.go
- Modèle `AssetPrice` représentant le prix d'un actif à un moment donné
- Champs :
  - `ID` : Identifiant auto-incrémenté
  - `ISIN` : Référence à l'actif
  - `Price` : Prix (DECIMAL 20,8)
  - `Currency` : Devise
  - `Timestamp` : Date et heure du prix
- Méthode `Validate()` :
  - Vérifie que l'ISIN est présent
  - Valide que le prix est positif
  - Vérifie que le timestamp est défini

#### ✅ models/transaction.go (étendu)
- Extension du modèle `Transaction` existant avec nouveaux champs :
  - `AccountID` : Référence au compte (UUID)
  - `ISIN` : Référence à l'actif
  - `Quantity` : Quantité d'actifs
  - `TransactionType` : Type de transaction (buy, sell, dividend, fee)
  - `Metadata` : Données JSON pour informations spécifiques à la plateforme
- Ajout des tags `db` pour tous les champs existants
- Méthode `Validate()` :
  - Vérifie l'ID, l'AccountID et le timestamp
  - Valide le format RFC3339 du timestamp
  - Vérifie la présence de la devise

#### ✅ models/models_test.go
- Tests unitaires complets pour tous les modèles
- Tests de validation pour :
  - Comptes valides et invalides (nom, plateforme, credentials)
  - Actifs valides et invalides (ISIN, type, devise)
  - Prix valides et invalides (ISIN, prix, timestamp)
  - Transactions valides et invalides (ID, AccountID, timestamp)
- **Résultat** : Tous les tests passent ✅

### 2. Système de migrations (Subtask 2.2)

#### ✅ database/db.go
- Structure `DB` encapsulant la connexion sqlx
- Structure `Config` pour la configuration de connexion
- Fonction `Connect()` :
  - Établit la connexion PostgreSQL avec DSN
  - Teste la connexion avec Ping
  - Retourne une instance `*DB`
- Fonction `Close()` pour fermer proprement la connexion

#### ✅ database/migrations.go
- Système de migrations complet avec 7 migrations :

**Migration 1 : Table accounts**
```sql
- Colonnes : id (UUID), name, platform, credentials (chiffré), timestamps
- Index sur platform
- Clé primaire UUID avec gen_random_uuid()
```

**Migration 2 : Table assets**
```sql
- Colonnes : isin (PK), name, symbol, type, currency, last_updated
- Index sur type et symbol
- Contraintes sur les types d'actifs
```

**Migration 3 : Table asset_prices**
```sql
- Colonnes : id (BIGSERIAL), isin (FK), price, currency, timestamp
- Contrainte UNIQUE sur (isin, timestamp)
- Index sur (isin, timestamp DESC) pour requêtes rapides
- Cascade DELETE sur suppression d'actif
```

**Migrations 4, 5, 6 : Tables de transactions par plateforme**
```sql
- transactions_traderepublic
- transactions_binance
- transactions_boursedirect

Chaque table contient :
- Tous les champs du modèle Transaction
- Référence FK vers accounts (CASCADE DELETE)
- Référence FK vers assets
- Index sur account_id, timestamp, isin, transaction_type
- Support JSONB pour metadata
```

**Migration 7 : Table schema_migrations**
```sql
- Suivi des migrations appliquées
- Colonnes : version (PK), name, applied_at
```

- Fonction `RunMigrations()` :
  - Crée la table de suivi des migrations
  - Récupère la version actuelle
  - Exécute les migrations en attente
  - Enregistre chaque migration appliquée
  - Logs détaillés pour chaque étape

- Fonction `RollbackMigration()` :
  - Annule la dernière migration
  - Exécute le script Down
  - Supprime l'enregistrement de migration

### 3. Couche d'accès aux données (Subtask 2.3)

#### ✅ database/accounts.go
Fonctions CRUD complètes pour les comptes :

- `CreateAccount()` :
  - Génère un UUID si non fourni
  - Définit les timestamps
  - Valide le compte
  - Insère dans la base de données

- `GetAccountByID()` : Récupère un compte par son ID
- `GetAllAccounts()` : Liste tous les comptes (triés par date de création)
- `GetAccountsByPlatform()` : Filtre les comptes par plateforme
- `UpdateAccount()` : Met à jour un compte existant
- `UpdateAccountLastSync()` : Met à jour uniquement le timestamp de synchronisation
- `DeleteAccount()` : Supprime un compte (cascade sur les transactions)

#### ✅ database/transactions.go
Gestion complète des transactions avec support multi-plateforme :

- `CreateTransaction()` :
  - Valide la transaction
  - Insère dans la table spécifique à la plateforme
  - Gère les conflits avec ON CONFLICT DO NOTHING

- `CreateTransactionsBatch()` :
  - Import en masse avec transaction SQL
  - Prepared statement pour performance
  - Gestion des doublons automatique

- `GetTransactionsByAccount()` :
  - Récupère les transactions d'un compte
  - Support des filtres : date, ISIN, type
  - Pagination avec LIMIT et OFFSET

- `GetAllTransactions()` :
  - Récupère toutes les transactions d'une plateforme
  - Mêmes filtres et pagination

- `GetTransactionByID()` : Récupère une transaction spécifique
- `DeleteTransaction()` : Supprime une transaction
- `CountTransactions()` : Compte les transactions avec filtres
- `ImportTransactionsFromJSON()` : Import depuis JSON

- Structure `TransactionFilter` :
  - AccountID, StartDate, EndDate
  - ISIN, TransactionType
  - Page, Limit (pagination)

- Fonction `getTransactionTableName()` :
  - Route vers la bonne table selon la plateforme
  - Fallback sur traderepublic

#### ✅ database/prices.go
Gestion des actifs et de leurs prix :

**Gestion des actifs :**
- `CreateAsset()` : Crée ou met à jour un actif (UPSERT)
- `GetAssetByISIN()` : Récupère un actif par ISIN
- `GetAllAssets()` : Liste tous les actifs
- `GetAssetsByType()` : Filtre par type (stock, etf, crypto)
- `UpdateAsset()` : Met à jour un actif
- `DeleteAsset()` : Supprime un actif

**Gestion des prix :**
- `CreateAssetPrice()` : Enregistre un prix (UPSERT sur conflit)
- `CreateAssetPricesBatch()` : Import en masse de prix
- `GetLatestAssetPrice()` : Prix le plus récent d'un actif
- `GetAssetPriceHistory()` : Historique des prix sur une période
- `GetAssetPriceAt()` : Prix à une date spécifique (ou avant)
- `GetAllLatestPrices()` : Derniers prix de tous les actifs
- `DeleteOldPrices()` : Nettoyage des prix anciens

### 4. Dépendances ajoutées

- ✅ `github.com/lib/pq` v1.11.1 : Driver PostgreSQL
- ✅ `github.com/jmoiron/sqlx` v1.4.0 : Extension de database/sql
- ✅ `github.com/google/uuid` v1.6.0 : Génération d'UUID

## Structure des fichiers créés

```
valhafin/
├── models/
│   ├── account.go           # Modèle Account avec validation
│   ├── asset.go             # Modèle Asset avec validation ISIN
│   ├── asset_price.go       # Modèle AssetPrice
│   ├── transaction.go       # Modèle Transaction étendu
│   └── models_test.go       # Tests unitaires (4 suites)
├── database/
│   ├── db.go                # Connexion PostgreSQL
│   ├── migrations.go        # Système de migrations (7 migrations)
│   ├── accounts.go          # CRUD comptes (8 fonctions)
│   ├── transactions.go      # CRUD transactions (10 fonctions)
│   └── prices.go            # CRUD actifs et prix (15 fonctions)
└── go.mod                   # Dépendances mises à jour
```

## Tests effectués

### Tests unitaires
```bash
$ go test ./models -v
=== RUN   TestAccountValidation
--- PASS: TestAccountValidation (0.00s)
=== RUN   TestAssetValidation
--- PASS: TestAssetValidation (0.00s)
=== RUN   TestAssetPriceValidation
--- PASS: TestAssetPriceValidation (0.00s)
=== RUN   TestTransactionValidation
--- PASS: TestTransactionValidation (0.00s)
PASS
ok      valhafin/models 0.539s
```

### Compilation
```bash
$ go build -o /dev/null .
✅ Build successful
```

## Schéma de base de données

```
accounts (UUID, name, platform, credentials, timestamps)
    ↓ (1:N, CASCADE DELETE)
transactions_traderepublic (id, account_id, timestamp, ...)
transactions_binance (id, account_id, timestamp, ...)
transactions_boursedirect (id, account_id, timestamp, ...)
    ↓ (N:1)
assets (isin, name, symbol, type, currency)
    ↓ (1:N, CASCADE DELETE)
asset_prices (id, isin, price, timestamp)
```

## Fonctionnalités clés

### Validation robuste
- Tous les modèles ont des méthodes de validation
- Formats stricts (ISIN, devise, plateforme)
- Messages d'erreur descriptifs

### Gestion multi-plateforme
- Tables de transactions séparées par plateforme
- Routing automatique vers la bonne table
- Support de Trade Republic, Binance, Bourse Direct

### Performance
- Index sur toutes les colonnes fréquemment requêtées
- Batch inserts pour les imports en masse
- Prepared statements pour les opérations répétitives
- Contraintes UNIQUE pour éviter les doublons

### Sécurité
- Credentials jamais exposés en JSON (tag `json:"-"`)
- Foreign keys avec CASCADE DELETE
- Validation avant insertion
- Transactions SQL pour les opérations en masse

## Prochaines étapes

La tâche 2 est terminée. Vous pouvez maintenant passer à la **Tâche 3 : Service de chiffrement et sécurité**.

Pour tester les migrations :

```bash
# Démarrer PostgreSQL
make dev-db

# Dans votre code Go, utiliser :
db, err := database.Connect(database.Config{
    Host:     "localhost",
    Port:     5432,
    User:     "valhafin",
    Password: "valhafin_dev_password",
    DBName:   "valhafin_dev",
    SSLMode:  "disable",
})

err = db.RunMigrations()
```

## Exigences satisfaites

- ✅ **Exigence 8.2** : Tables dédiées par compte financier (par plateforme)
- ✅ **Exigence 8.3** : Colonnes complètes pour les transactions (id, date, actif, montant, frais, type, métadonnées)
- ✅ **Exigence 8.4** : Index sur colonnes fréquemment utilisées (date, actif, type_opération)
- ✅ **Exigence 8.5** : Système de migrations pour gérer l'évolution du schéma

## Notes techniques

### Choix de conception

1. **Tables séparées par plateforme** : Permet une flexibilité pour des champs spécifiques à chaque plateforme tout en gardant une structure commune.

2. **UPSERT pour assets et prices** : Évite les erreurs de duplication et permet des mises à jour automatiques.

3. **UUID pour accounts** : Génération côté base de données avec `gen_random_uuid()` pour garantir l'unicité.

4. **JSONB pour metadata** : Flexibilité pour stocker des données spécifiques à chaque plateforme sans modifier le schéma.

5. **Validation stricte** : Tous les modèles valident leurs données avant insertion pour garantir l'intégrité.

### Améliorations futures possibles

- Ajouter des tests d'intégration avec une base de données de test
- Implémenter un pool de connexions configurable
- Ajouter des métriques de performance pour les requêtes
- Créer des vues SQL pour simplifier les requêtes complexes
- Ajouter un système de soft delete pour les transactions
