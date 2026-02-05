# Tâche 8 : Planificateur de tâches (Scheduler)

## Vue d'ensemble

Implémentation d'un système de planification de tâches périodiques pour automatiser la mise à jour des prix des actifs et la synchronisation des comptes.

## Statut : ✅ Complété

## Composants implémentés

### 1. Service Scheduler (`internal/service/scheduler/scheduler.go`)

**Fonctionnalités principales :**
- Gestion de tâches périodiques avec intervalles configurables
- Exécution concurrente de multiples tâches via goroutines
- Arrêt gracieux avec attente de complétion des tâches en cours
- Gestion d'erreurs robuste (les tâches continuent même en cas d'échec)

**Structure :**
```go
type Scheduler struct {
    tasks        []Task
    ctx          context.Context
    cancel       context.CancelFunc
    wg           sync.WaitGroup
    priceService price.Service
    syncService  SyncService
}
```

**Tâches par défaut :**
1. **update_prices** : Mise à jour des prix des actifs toutes les heures
2. **sync_accounts** : Synchronisation de tous les comptes toutes les 24 heures

**Méthodes clés :**
- `Start()` : Démarre toutes les tâches planifiées
- `Stop()` : Arrête gracieusement toutes les tâches
- `AddTask()` : Ajoute une nouvelle tâche personnalisée
- `runTask()` : Exécute une tâche à intervalle régulier

### 2. Intégration dans l'application

**Modifications apportées :**

**`internal/api/routes.go` :**
- Ajout d'une structure `Services` pour exposer les services nécessaires
- Retour du router ET des services pour utilisation par le scheduler

**`main.go` :**
- Import du package scheduler
- Initialisation du scheduler avec les services price et sync
- Démarrage du scheduler au lancement de l'application
- Gestion des signaux d'interruption (SIGINT, SIGTERM)
- Arrêt gracieux du scheduler avant fermeture de l'application

**Flux de démarrage :**
```
1. Connexion à la base de données
2. Initialisation des services (encryption, sync, price)
3. Configuration des routes API
4. Création et démarrage du scheduler
5. Démarrage du serveur HTTP
6. Attente du signal d'arrêt
7. Arrêt du scheduler
8. Fermeture de la base de données
```

### 3. Tests unitaires (`internal/service/scheduler/scheduler_test.go`)

**Tests implémentés :**

1. **TestSchedulerTaskExecution**
   - Vérifie que les tâches s'exécutent au bon intervalle
   - Valide le nombre d'exécutions sur une période donnée

2. **TestSchedulerDefaultTasks**
   - Confirme que les tâches par défaut sont ajoutées
   - Vérifie leur exécution au démarrage

3. **TestSchedulerTaskInterval**
   - Valide la précision des intervalles d'exécution
   - Vérifie l'exécution immédiate au démarrage
   - Contrôle les intervalles entre exécutions successives

4. **TestSchedulerErrorHandling**
   - Teste la résilience face aux erreurs
   - Vérifie que les tâches continuent malgré les échecs

5. **TestSchedulerGracefulShutdown**
   - Valide l'arrêt gracieux du scheduler
   - Vérifie que les tâches en cours se terminent avant l'arrêt

6. **TestSchedulerMultipleTasks**
   - Teste l'exécution concurrente de plusieurs tâches
   - Vérifie que chaque tâche respecte son propre intervalle

**Mocks utilisés :**
- `mockPriceService` : Simule le service de prix
- `mockSyncService` : Simule le service de synchronisation

**Résultats des tests :**
```
✅ TestSchedulerTaskExecution (0.35s)
✅ TestSchedulerDefaultTasks (0.10s)
✅ TestSchedulerTaskInterval (0.50s)
✅ TestSchedulerErrorHandling (0.10s)
✅ TestSchedulerGracefulShutdown (0.20s)
✅ TestSchedulerMultipleTasks (0.25s)
```

## Exigences satisfaites

### Exigence 2.6
> THE Système SHALL permettre la synchronisation automatique périodique des comptes connectés

**Implémentation :**
- Tâche `sync_accounts` exécutée toutes les 24 heures
- Appel à `syncService.SyncAllAccounts()`
- Logging des résultats (succès/échecs)

### Exigence 10.4
> THE Système SHALL mettre à jour les prix des actifs de manière périodique (quotidienne ou horaire)

**Implémentation :**
- Tâche `update_prices` exécutée toutes les heures
- Appel à `priceService.UpdateAllPrices()`
- Gestion des erreurs avec logging

## Architecture technique

### Concurrence et synchronisation
- Utilisation de goroutines pour l'exécution parallèle des tâches
- `sync.WaitGroup` pour attendre la complétion de toutes les tâches
- `context.Context` pour la propagation de l'annulation
- `sync.Mutex` dans les mocks pour la sécurité des accès concurrents

### Gestion du cycle de vie
```
Start() → addDefaultTasks() → runTask() (goroutines) → Stop()
   ↓                              ↓                        ↓
Ajoute tâches            Exécution périodique      Annulation context
                         avec ticker                + Wait sur WaitGroup
```

### Logging
- Logs structurés avec emojis pour meilleure lisibilité
- Niveaux : INFO (✅), ERROR (❌), GENERAL (📅, 💰, 🔄)
- Informations détaillées sur chaque exécution de tâche

## Utilisation

### Démarrage automatique
Le scheduler démarre automatiquement avec l'application :
```bash
./valhafin
```

Logs attendus :
```
📅 Scheduler starting...
📅 Task 'update_prices' scheduled to run every 1h0m0s
📅 Task 'sync_accounts' scheduled to run every 24h0m0s
📅 Scheduler started with 2 tasks
💰 Updating asset prices...
💰 Asset prices updated successfully
✅ Task 'update_prices' completed successfully
```

### Arrêt gracieux
L'application gère les signaux d'interruption :
```bash
# Ctrl+C ou kill
🛑 Shutdown signal received
📅 Scheduler stopping...
📅 Task 'update_prices' stopped
📅 Task 'sync_accounts' stopped
📅 Scheduler stopped
👋 Server stopped gracefully
```

### Ajout de tâches personnalisées
```go
scheduler.AddTask("custom_task", 30*time.Minute, func() error {
    // Logique de la tâche
    return nil
})
```

## Points techniques notables

1. **Interface SyncService** : Création d'une interface pour permettre le mocking dans les tests
2. **Alias d'import** : Utilisation de `syncsvc` pour éviter le conflit avec le package `sync` standard
3. **Exécution immédiate** : Les tâches s'exécutent immédiatement au démarrage, puis à intervalle régulier
4. **Tolérance aux pannes** : Les erreurs sont loggées mais n'arrêtent pas le scheduler

## Améliorations futures possibles

1. **Configuration dynamique** : Permettre de modifier les intervalles via configuration
2. **Métriques** : Exposer des métriques Prometheus sur l'exécution des tâches
3. **Retry avec backoff** : Implémenter une stratégie de retry pour les tâches échouées
4. **Persistance** : Sauvegarder l'état des tâches pour reprendre après redémarrage
5. **API de gestion** : Endpoints pour activer/désactiver des tâches à chaud

## Fichiers modifiés/créés

### Nouveaux fichiers
- `internal/service/scheduler/scheduler.go` (140 lignes)
- `internal/service/scheduler/scheduler_test.go` (340 lignes)
- `docs/TASK_8_SUMMARY.md` (ce document)

### Fichiers modifiés
- `internal/api/routes.go` : Ajout de la structure Services et modification du retour
- `main.go` : Intégration du scheduler avec gestion du cycle de vie

## Validation

✅ Compilation réussie : `go build -o valhafin .`
✅ Tous les tests passent : `go test ./internal/service/scheduler/ -v`
✅ Tests d'intégration OK : `go test ./...`
✅ Exigences 2.6 et 10.4 satisfaites

---

**Date de complétion :** 29 janvier 2026
**Développeur :** Kiro AI Assistant
**Durée estimée :** ~2 heures
