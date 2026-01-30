# Plan d'Implémentation: Application Web de Gestion de Portefeuille Financier

## Vue d'Ensemble

Ce plan d'implémentation transforme le design en étapes concrètes de développement. L'approche est incrémentale : chaque tâche construit sur les précédentes et se termine par l'intégration du code. Les tâches marquées d'un `*` sont optionnelles et peuvent être sautées pour un MVP plus rapide.

## Tâches

- [x] 1. Configuration initiale du projet et infrastructure de base
  - Créer la structure de dossiers pour le backend Go et le frontend React
  - Configurer PostgreSQL avec Docker Compose pour le développement local
  - Créer le fichier de configuration Go pour charger les variables d'environnement
  - Initialiser le projet frontend React avec Vite et TypeScript
  - Configurer Tailwind CSS avec le thème sombre personnalisé
  - _Exigences: 7.2, 8.1, 6.1, 6.2_

- [x] 2. Modèles de données et migrations de base de données
  - [x] 2.1 Créer les modèles Go pour Account, Asset, AssetPrice, Transaction
    - Définir les structs avec les tags JSON et DB appropriés
    - Implémenter les méthodes de validation pour chaque modèle
    - _Exigences: 8.3_
  
  - [x] 2.2 Implémenter le système de migrations de base de données
    - Créer les migrations pour les tables accounts, assets, asset_prices
    - Créer les migrations pour les tables de transactions (par plateforme)
    - Ajouter les index sur les colonnes fréquemment utilisées
    - _Exigences: 8.2, 8.3, 8.4, 8.5_
  
  - [x] 2.3 Créer la couche d'accès aux données (database layer)
    - Implémenter les fonctions CRUD pour les comptes
    - Implémenter les fonctions CRUD pour les transactions
    - Implémenter les fonctions CRUD pour les prix des actifs
    - _Exigences: 8.3_

- [x] 3. Service de chiffrement et sécurité
  - [x] 3.1 Implémenter le service de chiffrement AES-256-GCM
    - Créer les fonctions Encrypt et Decrypt
    - Implémenter la gestion sécurisée de la clé de chiffrement
    - _Exigences: 1.5_
  
  - [x] 3.2 Écrire les tests de propriété pour le chiffrement
    - **Propriété 22: Round-trip chiffrement/déchiffrement**
    - **Valide: Exigences 1.5**

- [x] 4. API REST - Gestion des comptes
  - [x] 4.1 Implémenter les endpoints de gestion des comptes
    - POST /api/accounts - Créer un compte avec chiffrement des credentials
    - GET /api/accounts - Lister tous les comptes
    - GET /api/accounts/:id - Détails d'un compte
    - DELETE /api/accounts/:id - Supprimer un compte avec cascade
    - _Exigences: 1.1, 1.2, 1.3, 1.5, 1.6_
  
  - [x] 4.2 Implémenter la validation des entrées pour les comptes
    - Valider le format des identifiants selon la plateforme
    - Valider les champs requis
    - _Exigences: 7.3_
  
  - [x] 4.3 Écrire les tests de propriété pour la gestion des comptes
    - **Propriété 1: Création de compte avec chiffrement**
    - **Propriété 2: Rejet des identifiants invalides**
    - **Propriété 3: Suppression en cascade**
    - **Valide: Exigences 1.1-1.6**

- [x] 5. Checkpoint - Vérifier que les tests passent et que l'API des comptes fonctionne
  - Tester manuellement la création, lecture, et suppression de comptes
  - Vérifier que les credentials sont bien chiffrés en base de données
  - Demander à l'utilisateur si des questions se posent

- [x] 6. Intégration des scrapers existants
  - [x] 6.1 Adapter les scrapers existants pour l'architecture API
    - Créer une interface Scraper commune pour Trade Republic, Binance, Bourse Direct
    - Adapter le scraper Trade Republic existant
    - Préparer les stubs pour Binance et Bourse Direct
    - _Exigences: 2.1, 7.2_
  
  - [x] 6.2 Implémenter l'endpoint de synchronisation
    - POST /api/accounts/:id/sync - Déclencher une synchronisation
    - Implémenter la logique de synchronisation complète (première fois)
    - Implémenter la logique de synchronisation incrémentale
    - Gérer les erreurs de scraping avec logging détaillé
    - _Exigences: 2.1, 2.2, 2.3, 2.4, 2.5_
  
  - [x] 6.3 Écrire les tests de propriété pour la synchronisation
    - **Propriété 4: Synchronisation complète initiale**
    - **Propriété 5: Synchronisation incrémentale**
    - **Propriété 6: Gestion d'erreur de synchronisation**
    - **Valide: Exigences 2.1-2.5**

- [x] 7. Service de récupération des prix (PriceService)
  - [x] 7.1 Implémenter le PriceService avec Yahoo Finance
    - Créer l'interface PriceService
    - Implémenter YahooFinancePriceService avec cache
    - Implémenter la conversion ISIN → symbole Yahoo Finance
    - Gérer les erreurs avec fallback sur dernier prix connu
    - _Exigences: 10.1, 10.2, 10.3, 10.5_
  
  - [x] 7.2 Implémenter les endpoints de prix
    - GET /api/assets/:isin/price - Prix actuel d'un actif
    - GET /api/assets/:isin/history - Historique des prix
    - _Exigences: 10.3, 10.6_
  
  - [x] 7.3 Écrire les tests de propriété pour le service de prix
    - **Propriété 13: Identification par ISIN**
    - **Propriété 14: Récupération et stockage des prix**
    - **Propriété 15: Fallback sur dernier prix connu**
    - **Valide: Exigences 10.1-10.6**

- [x] 8. Planificateur de tâches (Scheduler)
  - [x] 8.1 Implémenter le Scheduler pour les tâches périodiques
    - Créer le scheduler avec support de tâches récurrentes
    - Ajouter la tâche de mise à jour des prix (horaire)
    - Ajouter la tâche de synchronisation automatique (quotidienne)
    - _Exigences: 2.6, 10.4_
  
  - [x] 8.2 Écrire les tests unitaires pour le scheduler
    - Tester que les tâches sont bien déclenchées aux intervalles corrects
    - _Exigences: 2.6, 10.4_

- [x] 9. Checkpoint - Vérifier la synchronisation et les prix
  - Tester la synchronisation complète d'un compte Trade Republic
  - Vérifier que les prix sont récupérés et stockés
  - Vérifier que le scheduler fonctionne
  - Demander à l'utilisateur si des questions se posent

- [x] 10. Service de calcul de performance
  - [x] 10.1 Implémenter le PerformanceService
    - Créer l'interface PerformanceService
    - Implémenter le calcul de performance par compte
    - Implémenter le calcul de performance globale
    - Implémenter le calcul de performance par actif
    - Inclure les frais dans tous les calculs
    - _Exigences: 4.4, 4.6, 5.7, 10.7_
  
  - [x] 10.2 Implémenter les endpoints de performance
    - GET /api/accounts/:id/performance - Performance d'un compte
    - GET /api/performance - Performance globale
    - GET /api/assets/:isin/performance - Performance d'un actif
    - Supporter le filtrage par période (1m, 3m, 1y, all)
    - _Exigences: 4.1, 4.2, 4.3, 4.8_
  
  - [x] 10.3 Écrire les tests de propriété pour le calcul de performance
    - **Propriété 10: Calcul de performance avec prix actuels**
    - **Propriété 11: Agrégation de performance globale**
    - **Propriété 16: Calcul de valeur actuelle**
    - **Valide: Exigences 4.4, 4.6, 10.7**

- [x] 11. API REST - Transactions et filtres
  - [x] 11.1 Implémenter les endpoints de transactions
    - GET /api/accounts/:id/transactions - Lister les transactions d'un compte
    - GET /api/transactions - Lister toutes les transactions
    - Implémenter les filtres (date, type, actif)
    - Implémenter le tri (date, montant)
    - Implémenter la pagination (50 par page)
    - _Exigences: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7_
  
  - [x] 11.2 Écrire les tests de propriété pour les transactions
    - **Propriété 7: Filtrage des transactions**
    - **Propriété 8: Tri des transactions**
    - **Propriété 9: Pagination des transactions**
    - **Valide: Exigences 3.2-3.7**

- [x] 12. API REST - Métriques de frais
  - [x] 12.1 Implémenter les endpoints de métriques de frais
    - GET /api/accounts/:id/fees - Métriques de frais par compte
    - GET /api/fees - Métriques de frais globales
    - Calculer le total des frais, frais moyens, répartition par type
    - Supporter le filtrage par période
    - Générer les données pour le graphique d'évolution des frais
    - _Exigences: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
  
  - [x] 12.2 Écrire les tests de propriété pour les métriques de frais
    - **Propriété 17: Agrégation des frais**
    - **Valide: Exigences 5.1-5.6**

- [x] 13. Import CSV
  - [x] 13.1 Implémenter l'endpoint d'import CSV
    - POST /api/transactions/import - Importer des transactions depuis CSV
    - Valider le format du CSV (colonnes requises)
    - Parser le CSV et extraire les transactions
    - Détecter et ignorer les doublons
    - Retourner un résumé détaillé (importées, ignorées, erreurs)
    - _Exigences: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_
  
  - [x] 13.2 Écrire les tests de propriété pour l'import CSV
    - **Propriété 20: Parsing et validation CSV**
    - **Propriété 21: Import CSV avec déduplication**
    - **Valide: Exigences 9.1-9.6**

- [x] 14. Middleware et gestion d'erreurs backend
  - [x] 14.1 Implémenter les middlewares
    - Middleware CORS pour le frontend
    - Middleware de logging des requêtes
    - Middleware de gestion d'erreurs globale
    - _Exigences: 7.4, 7.5, 7.6_
  
  - [x] 14.2 Implémenter le endpoint de health check
    - GET /health - Vérifier l'état de l'application et de la base de données
    - _Exigences: 11.6_
  
  - [x] 14.3 Écrire les tests de propriété pour la validation et les erreurs
    - **Propriété 18: Validation des entrées API**
    - **Propriété 23: Logging des requêtes et erreurs**
    - **Propriété 26: Health check**
    - **Valide: Exigences 7.3-7.5, 11.6**

- [x] 15. Checkpoint - Backend complet
  - Tester tous les endpoints API avec Postman ou curl
  - Vérifier que tous les calculs de performance sont corrects
  - Vérifier que les logs sont créés correctement
  - Demander à l'utilisateur si des questions se posent

- [x] 16. Frontend - Configuration et structure de base
  - [x] 16.1 Configurer le projet React avec TypeScript
    - Initialiser le projet avec Vite
    - Configurer Tailwind CSS avec le thème sombre personnalisé
    - Configurer React Router pour la navigation
    - Configurer React Query (TanStack Query) pour la gestion des données
    - Configurer Axios pour les appels API
    - _Exigences: 6.1, 6.2, 6.3_
  
  - [x] 16.2 Créer la structure de composants de base
    - Créer le Layout avec Sidebar et Header
    - Créer les composants communs (LoadingSpinner, ErrorMessage, Pagination)
    - Créer les pages principales (Dashboard, Transactions, Performance, Fees)
    - _Exigences: 6.3, 6.6, 6.7_

- [x] 17. Frontend - Service API et hooks
  - [x] 17.1 Créer le client API HTTP
    - Configurer Axios avec l'URL de base et les intercepteurs
    - Créer les fonctions API pour les comptes
    - Créer les fonctions API pour les transactions
    - Créer les fonctions API pour la performance
    - Créer les fonctions API pour les frais
    - _Exigences: 7.1_
  
  - [x] 17.2 Créer les hooks personnalisés avec React Query
    - useAccounts, useTransactions, usePerformance, useFees
    - Configurer le cache et les stratégies de refetch
    - _Exigences: 7.1_

- [x] 18. Frontend - Gestion des comptes
  - [x] 18.1 Implémenter la page de gestion des comptes
    - Créer AccountList pour afficher tous les comptes
    - Créer AccountCard pour afficher un compte
    - Créer AddAccountModal pour ajouter un nouveau compte
    - Implémenter la synchronisation manuelle d'un compte
    - Implémenter la suppression d'un compte
    - _Exigences: 1.1, 1.2, 1.3, 1.4, 1.6, 2.1_
  
  - [x] 18.2 Écrire les tests E2E pour la gestion des comptes
    - Tester l'ajout d'un compte
    - Tester la synchronisation
    - Tester la suppression
    - _Exigences: 1.1-1.6_

- [x] 19. Frontend - Liste des transactions
  - [x] 19.1 Implémenter la page des transactions
    - Créer TransactionTable avec colonnes (Date, Actif, Type, Montant, Frais)
    - Créer TransactionFilters pour les filtres (date, type, actif)
    - Implémenter le tri cliquable sur les colonnes
    - Implémenter la pagination
    - Implémenter le clic sur un actif pour ouvrir AssetPerformanceModal
    - _Exigences: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 4.8_
  
  - [x] 19.2 Créer ImportCSVModal pour l'import de transactions
    - Implémenter l'upload de fichier CSV
    - Afficher le résumé d'import
    - Gérer les erreurs d'import
    - _Exigences: 9.1, 9.2, 9.3, 9.6_

- [ ] 20. Frontend - Graphiques de performance
  - [ ] 20.1 Implémenter PerformanceChart avec Recharts
    - Créer le graphique de ligne pour l'évolution de la valeur
    - Implémenter le sélecteur de période (1m, 3m, 1y, all)
    - Implémenter le tooltip personnalisé avec date et valeur
    - Appliquer le style avec gradient bleu
    - _Exigences: 4.1, 4.2, 4.3, 4.7_
  
  - [ ] 20.2 Créer PerformanceMetrics pour afficher les métriques
    - Afficher la valeur totale, investissement, gains/pertes
    - Afficher la performance en pourcentage
    - Utiliser des couleurs (vert pour gains, rouge pour pertes)
    - _Exigences: 4.4, 4.5, 4.6_
  
  - [ ] 20.3 Créer AssetPerformanceModal
    - Afficher le nom de l'actif, ISIN, et prix actuel
    - Afficher le graphique de performance de l'actif
    - Afficher les métriques de l'actif
    - _Exigences: 4.8, 4.9_

- [ ] 21. Frontend - Métriques de frais
  - [ ] 21.1 Implémenter la page des frais
    - Créer FeesOverview pour afficher les métriques (total, moyenne, répartition)
    - Créer FeesChart pour le graphique d'évolution des frais
    - Implémenter le sélecteur de période
    - _Exigences: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

- [ ] 22. Frontend - Dashboard
  - [ ] 22.1 Créer la page Dashboard
    - Afficher un résumé de la performance globale
    - Afficher les comptes avec leurs valeurs actuelles
    - Afficher les dernières transactions
    - Afficher un graphique de performance globale
    - _Exigences: 4.2, 4.4_

- [ ] 23. Frontend - Responsive et animations
  - [ ] 23.1 Rendre l'application responsive
    - Adapter le layout pour mobile et tablette
    - Adapter les tableaux pour les petits écrans
    - Adapter les graphiques pour les petits écrans
    - _Exigences: 6.3_
  
  - [ ] 23.2 Ajouter des transitions et animations
    - Ajouter des transitions entre les pages
    - Ajouter des animations subtiles sur les interactions
    - _Exigences: 6.4_

- [ ] 24. Checkpoint - Frontend complet
  - Tester l'application complète dans le navigateur
  - Vérifier que toutes les fonctionnalités fonctionnent
  - Vérifier le responsive sur différentes tailles d'écran
  - Demander à l'utilisateur si des questions se posent

- [ ] 25. Docker et packaging
  - [ ] 25.1 Créer le Dockerfile multi-stage
    - Stage 1: Build du frontend React
    - Stage 2: Build du backend Go
    - Stage 3: Image finale optimisée
    - _Exigences: 11.1_
  
  - [ ] 25.2 Créer le docker-compose.yml pour production
    - Service PostgreSQL avec volumes persistants
    - Service Valhafin avec health checks
    - Configuration des réseaux et volumes
    - _Exigences: 11.2_
  
  - [ ] 25.3 Créer le script de déploiement
    - Créer deploy.sh avec vérifications et logs
    - Créer .env.example avec toutes les variables
    - _Exigences: 11.5_
  
  - [ ] 25.4 Écrire les tests pour le packaging Docker
    - **Propriété 24: Packaging Docker**
    - **Valide: Exigences 11.1, 11.2**

- [ ] 26. GitHub Actions et CI/CD
  - [ ] 26.1 Créer le workflow CI/CD
    - Job test-backend: tests Go avec coverage
    - Job test-frontend: tests React et linting
    - Job build-and-push: build et push de l'image Docker
    - Job create-release: création de release avec package
    - _Exigences: 11.3, 11.4_
  
  - [ ] 26.2 Configurer Dependabot
    - Créer .github/dependabot.yml pour Go, npm, Docker, GitHub Actions
    - _Exigences: 11.3_
  
  - [ ] 26.3 Écrire les tests pour le CI/CD
    - **Propriété 25: CI/CD automatisé**
    - **Propriété 27: Versioning sémantique**
    - **Valide: Exigences 11.3, 11.4, 11.8**

- [ ] 27. Terraform (optionnel)
  - [ ] 27.1 Créer la configuration Terraform
    - Créer main.tf pour provisionner une VM
    - Créer variables.tf pour les paramètres
    - Configurer le user_data pour installer Docker et déployer l'app
    - _Exigences: 11.7_

- [ ] 28. Documentation et finalisation
  - [ ] 28.1 Créer la documentation utilisateur
    - Mettre à jour README.md avec instructions de déploiement
    - Documenter les variables d'environnement
    - Documenter le processus de release
    - _Exigences: 11.5, 11.8_
  
  - [ ] 28.2 Créer la documentation développeur
    - Documenter l'architecture de l'API
    - Documenter les modèles de données
    - Documenter les processus de développement local

- [ ] 29. Checkpoint final - Application complète
  - Déployer l'application avec Docker Compose
  - Tester le déploiement complet
  - Créer une première release v1.0.0
  - Vérifier que tout fonctionne en production
  - Demander à l'utilisateur si des questions se posent

## Notes

- Toutes les tâches sont obligatoires pour garantir une couverture de tests complète
- Chaque tâche référence les exigences spécifiques pour la traçabilité
- Les checkpoints permettent une validation incrémentale
- Les tests de propriété valident les propriétés universelles de correction
- Les tests unitaires valident des exemples spécifiques et des cas limites
