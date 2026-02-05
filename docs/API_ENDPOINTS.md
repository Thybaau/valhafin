# API Endpoints Documentation

## Base URL
```
http://localhost:8080/api
```

## Table of Contents
- [Health Check](#health-check)
- [Accounts](#accounts)
- [Transactions](#transactions)
- [Performance](#performance)
- [Fees](#fees)
- [Assets](#assets)

---

## Health Check

### GET `/health`
**Description:** Vérifie l'état de santé du backend et de la base de données

**Utilisé par:** Monitoring, healthcheck

**Réponse:**
```json
{
  "status": "healthy",
  "database": "connected",
  "version": "1.0.0",
  "uptime": "2h30m15s"
}
```

---

## Accounts

### GET `/api/accounts`
**Description:** Récupère la liste de tous les comptes

**Utilisé par:** Page Accounts, Dashboard

**Réponse:**
```json
[
  {
    "id": "uuid",
    "name": "Trade Republic",
    "platform": "traderepublic",
    "created_at": "2024-01-01T00:00:00Z",
    "last_synced": "2024-01-15T10:30:00Z"
  }
]
```

---

### GET `/api/accounts/{id}`
**Description:** Récupère les détails d'un compte spécifique

**Utilisé par:** Page Account Details

**Paramètres:**
- `id` (path): ID du compte

**Réponse:**
```json
{
  "id": "uuid",
  "name": "Trade Republic",
  "platform": "traderepublic",
  "created_at": "2024-01-01T00:00:00Z",
  "last_synced": "2024-01-15T10:30:00Z"
}
```

---

### POST `/api/accounts`
**Description:** Crée un nouveau compte avec credentials chiffrés

**Utilisé par:** Modal "Add Account"

**Body:**
```json
{
  "name": "Mon Trade Republic",
  "platform": "traderepublic",
  "credentials": {
    "phone_number": "+33612345678",
    "pin": "1234"
  }
}
```

**Réponse:**
```json
{
  "id": "uuid",
  "name": "Mon Trade Republic",
  "platform": "traderepublic",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### DELETE `/api/accounts/{id}`
**Description:** Supprime un compte et toutes ses données associées (cascade)

**Utilisé par:** Page Accounts, bouton "Delete"

**Paramètres:**
- `id` (path): ID du compte

**Réponse:**
```json
{
  "message": "Account deleted successfully"
}
```

---

### POST `/api/accounts/{id}/sync`
**Description:** Synchronise un compte (Binance, Bourse Direct - sans 2FA)

**Utilisé par:** Page Accounts, bouton "Sync"

**Paramètres:**
- `id` (path): ID du compte

**Réponse:**
```json
{
  "success": true,
  "transactions_added": 42,
  "message": "Synchronization completed"
}
```

---

### POST `/api/accounts/{id}/sync/init`
**Description:** Initie la synchronisation Trade Republic (déclenche 2FA)

**Utilisé par:** Page Accounts, bouton "Sync" pour Trade Republic

**Paramètres:**
- `id` (path): ID du compte

**Réponse:**
```json
{
  "requires_two_factor": true,
  "process_id": "process-uuid",
  "message": "2FA code sent to your phone"
}
```

---

### POST `/api/accounts/{id}/sync/complete`
**Description:** Complète la synchronisation Trade Republic avec le code 2FA

**Utilisé par:** Modal 2FA

**Paramètres:**
- `id` (path): ID du compte

**Body:**
```json
{
  "process_id": "process-uuid",
  "code": "123456"
}
```

**Réponse:**
```json
{
  "success": true,
  "transactions_added": 42,
  "message": "Synchronization completed"
}
```

---

## Transactions

### GET `/api/accounts/{id}/transactions`
**Description:** Récupère les transactions d'un compte avec filtres et pagination

**Utilisé par:** Page Account Details

**Paramètres:**
- `id` (path): ID du compte
- `start_date` (query, optional): Date de début (YYYY-MM-DD)
- `end_date` (query, optional): Date de fin (YYYY-MM-DD)
- `asset` (query, optional): Filtrer par ISIN
- `type` (query, optional): Filtrer par type (buy, sell, dividend, etc.)
- `page` (query, optional): Numéro de page (défaut: 1)
- `limit` (query, optional): Nombre par page (défaut: 50)
- `sort_by` (query, optional): Champ de tri (date, amount, type)
- `sort_order` (query, optional): Ordre (asc, desc)

**Réponse:**
```json
{
  "transactions": [...],
  "total": 150,
  "page": 1,
  "limit": 50,
  "total_pages": 3
}
```

---

### GET `/api/transactions`
**Description:** Récupère toutes les transactions (tous comptes) avec filtres et pagination

**Utilisé par:** Page Transactions

**Paramètres:** Mêmes que `/api/accounts/{id}/transactions`

**Réponse:**
```json
{
  "transactions": [
    {
      "id": "uuid",
      "account_id": "account-uuid",
      "date": "2024-01-15T10:30:00Z",
      "type": "buy",
      "asset_name": "Physical Gold USD (Acc)",
      "isin": "IE00B4ND3602",
      "quantity": 0.5,
      "amount_value": 38.85,
      "amount_currency": "EUR",
      "fees_value": 0,
      "fees_currency": "EUR"
    }
  ],
  "total": 150,
  "page": 1,
  "limit": 50,
  "total_pages": 3
}
```

---

### PUT `/api/transactions/{id}`
**Description:** Met à jour une transaction existante

**Utilisé par:** Modal "Edit Transaction"

**Paramètres:**
- `id` (path): ID de la transaction

**Body:**
```json
{
  "type": "buy",
  "quantity": 1.0,
  "amount_value": 77.71
}
```

**Réponse:**
```json
{
  "id": "uuid",
  "type": "buy",
  "quantity": 1.0,
  "amount_value": 77.71,
  "updated_at": "2024-01-15T10:30:00Z"
}
```

---

### POST `/api/transactions/import`
**Description:** Importe des transactions depuis un fichier CSV

**Utilisé par:** Modal "Import CSV"

**Body:** multipart/form-data
- `file`: Fichier CSV
- `account_id`: ID du compte

**Réponse:**
```json
{
  "imported": 42,
  "ignored": 3,
  "errors": [
    {
      "line": 5,
      "message": "Invalid date format"
    }
  ]
}
```

---

## Performance

### GET `/api/performance`
**Description:** Récupère les métriques de performance globales (tous comptes)

**Utilisé par:** Page Performance, Dashboard

**Paramètres:**
- `period` (query, optional): Période (1m, 3m, 6m, 1y, all)

**Réponse:**
```json
{
  "total_invested": 4738.45,
  "total_value": 5124.32,
  "total_gain": 385.87,
  "total_gain_percent": 8.14,
  "cash_balance": 1024.15,
  "time_series": [
    {
      "date": "2024-01-01",
      "value": 4500.00
    }
  ]
}
```

---

### GET `/api/accounts/{id}/performance`
**Description:** Récupère les métriques de performance d'un compte spécifique

**Utilisé par:** Page Account Details

**Paramètres:**
- `id` (path): ID du compte
- `period` (query, optional): Période (1m, 3m, 6m, 1y, all)

**Réponse:** Même format que `/api/performance`

---

### GET `/api/assets/{isin}/performance`
**Description:** Récupère les métriques de performance d'un actif spécifique

**Utilisé par:** Modal "Asset Performance"

**Paramètres:**
- `isin` (path): ISIN de l'actif
- `period` (query, optional): Période (1m, 3m, 6m, 1y, all)

**Réponse:**
```json
{
  "isin": "IE00B4ND3602",
  "name": "Physical Gold USD (Acc)",
  "quantity": 0.5,
  "average_buy_price": 77.70,
  "current_price": 77.71,
  "total_invested": 38.85,
  "current_value": 38.86,
  "gain": 0.01,
  "gain_percent": 0.03
}
```

---

## Fees

### GET `/api/fees`
**Description:** Récupère les métriques de frais globales (tous comptes)

**Utilisé par:** Page Fees

**Paramètres:**
- `start_date` (query, optional): Date de début (YYYY-MM-DD)
- `end_date` (query, optional): Date de fin (YYYY-MM-DD)
- `period` (query, optional): Période (1m, 3m, 1y, all)

**Réponse:**
```json
{
  "total_fees": 12.50,
  "average_fees": 0.25,
  "fees_by_type": {
    "buy": 5.00,
    "sell": 7.50,
    "transfer": 0.00,
    "other": 0.00
  },
  "time_series": [
    {
      "date": "2024-01-01",
      "fees": 1.00
    }
  ]
}
```

---

### GET `/api/accounts/{id}/fees`
**Description:** Récupère les métriques de frais d'un compte spécifique

**Utilisé par:** Page Account Details

**Paramètres:**
- `id` (path): ID du compte
- Mêmes query params que `/api/fees`

**Réponse:** Même format que `/api/fees`

---

## Assets

### GET `/api/assets`
**Description:** Récupère tous les actifs avec positions et valeurs actuelles

**Utilisé par:** Page Assets

**Réponse:**
```json
[
  {
    "isin": "IE00B4ND3602",
    "name": "Physical Gold USD (Acc)",
    "symbol": "IGLN",
    "type": "etf",
    "currency": "EUR",
    "quantity": 0.5,
    "average_buy_price": 77.70,
    "current_price": 77.71,
    "current_value": 38.86,
    "gain": 0.01,
    "gain_percent": 0.03,
    "purchases": [
      {
        "date": "2024-01-15T10:30:00Z",
        "quantity": 0.5,
        "price": 77.70
      }
    ]
  }
]
```

---

### GET `/api/assets/{isin}/price`
**Description:** Récupère le prix actuel d'un actif

**Utilisé par:** Page Assets (refresh price)

**Paramètres:**
- `isin` (path): ISIN de l'actif

**Réponse:**
```json
{
  "isin": "IE00B4ND3602",
  "price": 77.71,
  "currency": "EUR",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

### GET `/api/assets/{isin}/history`
**Description:** Récupère l'historique des prix d'un actif

**Utilisé par:** Page Assets (graphique)

**Paramètres:**
- `isin` (path): ISIN de l'actif
- `start_date` (query, optional): Date de début (YYYY-MM-DD)
- `end_date` (query, optional): Date de fin (YYYY-MM-DD)

**Réponse:**
```json
[
  {
    "isin": "IE00B4ND3602",
    "price": 77.50,
    "currency": "EUR",
    "timestamp": "2024-01-01T00:00:00Z"
  },
  {
    "isin": "IE00B4ND3602",
    "price": 77.71,
    "currency": "EUR",
    "timestamp": "2024-01-15T00:00:00Z"
  }
]
```

---

### POST `/api/assets/{isin}/price/update`
**Description:** Force la mise à jour du prix d'un actif (admin/debug)

**Utilisé par:** Debug, admin tools

**Paramètres:**
- `isin` (path): ISIN de l'actif

**Réponse:**
```json
{
  "isin": "IE00B4ND3602",
  "price": 77.71,
  "currency": "EUR",
  "timestamp": "2024-01-15T10:30:00Z",
  "message": "Price updated successfully"
}
```

---

## Résumé

**Total: 21 endpoints**

- ✅ **19 utilisés par le frontend**
- ✅ **2 utilisés pour monitoring/debug** (`/health`, `/assets/{isin}/price/update`)

**Répartition:**
- Health: 1 endpoint
- Accounts: 7 endpoints
- Transactions: 4 endpoints
- Performance: 3 endpoints
- Fees: 2 endpoints
- Assets: 4 endpoints
