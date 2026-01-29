# Document des Exigences

## Introduction

Valhafin est une application web personnelle de gestion de portefeuille financier qui permet de connecter des comptes sur différentes plateformes d'investissement (Trade Republic, Binance, Bourse Direct), de télécharger automatiquement l'historique des transactions, et de visualiser les performances financières à travers des graphiques et des métriques détaillées. L'application réutilise un backend Go existant avec des scrapers déjà développés et ajoute une interface web moderne avec un thème sombre.

## Glossaire

- **Système**: L'application web Valhafin dans son ensemble (backend Go + frontend web)
- **Compte_Financier**: Compte sur une plateforme d'investissement (Trade Republic, Binance, Bourse Direct)
- **Transaction**: Opération financière (achat, vente, dividende, frais) sur un compte financier
- **Scraper**: Module backend qui se connecte à une plateforme et récupère les données
- **Base_Données**: Base de données SQL (PostgreSQL ou SQLite) stockant les transactions
- **Historique**: Ensemble des transactions d'un compte financier sur une période donnée
- **Performance**: Évolution de la valeur d'un portefeuille dans le temps
- **Métriques**: Indicateurs calculés (frais totaux, gains, pertes, etc.)
- **ISIN**: Code d'identification international des valeurs mobilières (International Securities Identification Number)
- **Prix_Actuel**: Valeur de marché actuelle d'un actif financier
- **API_Financière**: Service externe fournissant les données de prix des actifs (ex: Yahoo Finance, Alpha Vantage)
- **Docker**: Plateforme de conteneurisation pour packager l'application
- **GitHub_Actions**: Service d'intégration continue et déploiement continu (CI/CD)
- **Release**: Version packageée et publiée de l'application
- **VM**: Machine virtuelle hébergeant l'application
- **Health_Check**: Endpoint API vérifiant l'état de santé de l'application

## Exigences

### Exigence 1: Connexion aux Comptes Financiers

**User Story:** En tant qu'utilisateur, je veux connecter mes comptes Trade Republic, Binance et Bourse Direct, afin de centraliser la gestion de mes investissements.

#### Critères d'Acceptation

1. WHEN l'utilisateur ajoute un compte Trade Republic avec des identifiants valides, THEN LE Système SHALL établir la connexion et stocker les informations de connexion de manière sécurisée
2. WHEN l'utilisateur ajoute un compte Binance avec des clés API valides, THEN LE Système SHALL établir la connexion et stocker les clés de manière sécurisée
3. WHEN l'utilisateur ajoute un compte Bourse Direct avec des identifiants valides, THEN LE Système SHALL établir la connexion et stocker les informations de connexion de manière sécurisée
4. WHEN l'utilisateur tente de connecter un compte avec des identifiants invalides, THEN LE Système SHALL rejeter la connexion et afficher un message d'erreur explicite
5. THE Système SHALL chiffrer les identifiants et clés API avant de les stocker dans la Base_Données
6. WHEN l'utilisateur supprime un compte connecté, THEN LE Système SHALL supprimer toutes les données associées (identifiants et historique des transactions)

### Exigence 2: Téléchargement et Stockage des Transactions

**User Story:** En tant qu'utilisateur, je veux télécharger automatiquement l'historique de mes transactions depuis mes comptes connectés, afin de disposer d'une vue centralisée de toutes mes opérations financières.

#### Critères d'Acceptation

1. WHEN l'utilisateur déclenche une synchronisation pour un compte, THEN LE Système SHALL utiliser le Scraper approprié pour récupérer l'historique complet des transactions
2. WHEN des transactions sont récupérées, THEN LE Système SHALL les stocker dans une table SQL dédiée au compte financier correspondant
3. FOR ALL transactions stockées, THE Système SHALL enregistrer la date, l'actif, le montant, les frais, le type d'opération et toutes les métadonnées disponibles
4. WHEN une synchronisation est déclenchée pour un compte déjà synchronisé, THEN LE Système SHALL télécharger uniquement les nouvelles transactions depuis la dernière synchronisation
5. IF une erreur survient pendant la synchronisation, THEN LE Système SHALL enregistrer l'erreur dans les logs et afficher un message d'erreur
6. THE Système SHALL permettre la synchronisation automatique périodique des comptes connectés

### Exigence 3: Affichage des Transactions

**User Story:** En tant qu'utilisateur, je veux consulter la liste de mes transactions par compte avec des filtres et des options de tri, afin d'analyser facilement mes opérations financières.

#### Critères d'Acceptation

1. WHEN l'utilisateur accède à la vue des transactions d'un compte, THEN LE Système SHALL afficher toutes les transactions avec la date, l'actif, le montant, les frais et le type d'opération
2. WHEN l'utilisateur applique un filtre par date, THEN LE Système SHALL afficher uniquement les transactions dans la période sélectionnée
3. WHEN l'utilisateur applique un filtre par type d'opération, THEN LE Système SHALL afficher uniquement les transactions du type sélectionné
4. WHEN l'utilisateur applique un filtre par actif, THEN LE Système SHALL afficher uniquement les transactions concernant l'actif sélectionné
5. WHEN l'utilisateur trie les transactions par date, THEN LE Système SHALL réorganiser la liste selon l'ordre chronologique choisi (croissant ou décroissant)
6. WHEN l'utilisateur trie les transactions par montant, THEN LE Système SHALL réorganiser la liste selon l'ordre de montant choisi (croissant ou décroissant)
7. THE Système SHALL paginer les résultats lorsque le nombre de transactions dépasse 50 par page

### Exigence 4: Visualisation des Performances

**User Story:** En tant qu'utilisateur, je veux visualiser les performances de mes comptes à travers des graphiques, afin de suivre l'évolution de mes investissements dans le temps.

#### Critères d'Acceptation

1. WHEN l'utilisateur accède à la vue de performance d'un compte, THEN LE Système SHALL afficher un graphique montrant l'évolution de la valeur du portefeuille dans le temps
2. WHEN l'utilisateur accède à la vue de performance globale, THEN LE Système SHALL afficher un graphique montrant l'évolution de la valeur totale de tous les comptes confondus
3. WHEN l'utilisateur sélectionne une période (1 mois, 3 mois, 1 an, tout), THEN LE Système SHALL mettre à jour le graphique pour afficher uniquement les données de la période sélectionnée
4. THE Système SHALL calculer la performance en pourcentage en utilisant la valeur actuelle de chaque actif basée sur son ISIN
5. THE Système SHALL récupérer les prix actuels des actifs depuis des sources de données financières externes
6. THE Système SHALL calculer les gains et pertes réalisés et non réalisés en incluant tous les frais de transaction
7. WHEN l'utilisateur survole un point du graphique, THEN LE Système SHALL afficher une infobulle avec la valeur exacte et la date
8. WHEN l'utilisateur clique sur un actif spécifique, THEN LE Système SHALL afficher un graphique détaillé de l'évolution de la performance de cet actif dans le temps
9. FOR ALL actifs affichés, THE Système SHALL identifier l'actif par son ISIN pour récupérer les données de prix historiques et actuelles

### Exigence 5: Métriques sur les Frais

**User Story:** En tant qu'utilisateur, je veux consulter des métriques détaillées sur les frais payés, afin d'optimiser mes coûts de transaction.

#### Critères d'Acceptation

1. WHEN l'utilisateur accède à la vue des métriques de frais, THEN LE Système SHALL afficher le total des frais payés par compte
2. WHEN l'utilisateur accède à la vue des métriques de frais, THEN LE Système SHALL afficher le total des frais payés tous comptes confondus
3. THE Système SHALL calculer et afficher les frais moyens par transaction
4. THE Système SHALL afficher la répartition des frais par type d'opération (achat, vente, transfert)
5. WHEN l'utilisateur sélectionne une période, THEN LE Système SHALL mettre à jour les métriques pour afficher uniquement les frais de la période sélectionnée
6. THE Système SHALL afficher un graphique montrant l'évolution des frais dans le temps
7. THE Système SHALL inclure les frais dans le calcul de la performance globale du portefeuille

### Exigence 6: Interface Utilisateur Moderne

**User Story:** En tant qu'utilisateur, je veux une interface moderne avec un thème sombre et des touches de bleu, afin de bénéficier d'une expérience visuelle agréable et professionnelle.

#### Critères d'Acceptation

1. THE Système SHALL utiliser un thème sombre avec des couleurs noir et gris anthracite comme couleurs principales
2. THE Système SHALL utiliser des touches de bleu pour les éléments interactifs et les accents visuels
3. THE Système SHALL être responsive et s'adapter aux différentes tailles d'écran (desktop, tablette, mobile)
4. WHEN l'utilisateur navigue entre les pages, THEN LE Système SHALL fournir des transitions fluides et des animations subtiles
5. THE Système SHALL utiliser une typographie moderne et lisible
6. THE Système SHALL afficher des indicateurs de chargement pendant les opérations asynchrones
7. WHEN une erreur survient, THEN LE Système SHALL afficher un message d'erreur clair et non intrusif

### Exigence 7: Architecture Backend et API

**User Story:** En tant que développeur, je veux une architecture backend robuste avec des API RESTful, afin de garantir la maintenabilité et l'évolutivité du système.

#### Critères d'Acceptation

1. THE Système SHALL exposer une API RESTful pour toutes les opérations (gestion des comptes, récupération des transactions, métriques)
2. THE Système SHALL utiliser le backend Go existant et les scrapers déjà développés
3. THE Système SHALL valider toutes les entrées utilisateur avant traitement
4. WHEN une requête API échoue, THEN LE Système SHALL retourner un code HTTP approprié et un message d'erreur structuré en JSON
5. THE Système SHALL logger toutes les requêtes API et les erreurs pour faciliter le débogage
6. THE Système SHALL gérer les erreurs de connexion aux plateformes externes de manière gracieuse

### Exigence 8: Gestion de la Base de Données

**User Story:** En tant que développeur, je veux une structure de base de données SQL bien organisée, afin de stocker efficacement les transactions et les données des comptes.

#### Critères d'Acceptation

1. THE Système SHALL utiliser PostgreSQL comme système de gestion de base de données
2. THE Système SHALL créer une table dédiée pour chaque compte financier connecté
3. FOR ALL tables de transactions, THE Système SHALL inclure les colonnes: id, date, actif, montant, frais, type_opération, métadonnées
4. THE Système SHALL créer des index sur les colonnes fréquemment utilisées pour les requêtes (date, actif, type_opération)
5. THE Système SHALL implémenter des migrations de base de données pour gérer l'évolution du schéma
6. THE Système SHALL effectuer des sauvegardes régulières de la base de données
7. WHEN l'utilisateur supprime un compte, THEN LE Système SHALL supprimer la table associée et toutes les données liées

### Exigence 9: Import de Données CSV

**User Story:** En tant qu'utilisateur, je veux importer des données depuis des fichiers CSV, afin d'ajouter manuellement des transactions ou de migrer des données historiques.

#### Critères d'Acceptation

1. WHEN l'utilisateur télécharge un fichier CSV valide, THEN LE Système SHALL parser le fichier et extraire les transactions
2. THE Système SHALL valider le format du fichier CSV avant l'import (colonnes requises: date, actif, montant, frais)
3. WHEN le fichier CSV contient des erreurs de format, THEN LE Système SHALL rejeter l'import et afficher un rapport d'erreurs détaillé
4. WHEN l'import est réussi, THEN LE Système SHALL insérer les transactions dans la table du compte correspondant
5. THE Système SHALL détecter et ignorer les transactions en double lors de l'import
6. WHEN l'import est terminé, THEN LE Système SHALL afficher un résumé (nombre de transactions importées, ignorées, erreurs)

### Exigence 10: Récupération des Prix des Actifs

**User Story:** En tant qu'utilisateur, je veux que le système récupère automatiquement les prix actuels et historiques de mes actifs, afin de calculer la performance en temps réel de mon portefeuille.

#### Critères d'Acceptation

1. THE Système SHALL identifier chaque actif par son ISIN pour récupérer les données de prix
2. THE Système SHALL se connecter à une API_Financière externe pour récupérer les prix actuels des actifs
3. WHEN un actif est identifié, THEN LE Système SHALL récupérer son prix actuel et le stocker dans la Base_Données
4. THE Système SHALL mettre à jour les prix des actifs de manière périodique (quotidienne ou horaire)
5. WHEN un prix ne peut pas être récupéré pour un actif, THEN LE Système SHALL utiliser le dernier prix connu et afficher un avertissement
6. THE Système SHALL récupérer l'historique des prix pour permettre l'affichage de graphiques de performance par actif
7. FOR ALL actifs dans le portefeuille, THE Système SHALL calculer la valeur actuelle en multipliant la quantité détenue par le Prix_Actuel

### Exigence 11: Déploiement et Packaging

**User Story:** En tant que développeur, je veux que l'application soit facilement packageable et déployable sur une VM, afin de simplifier les mises à jour et les releases.

#### Critères d'Acceptation

1. THE Système SHALL être packageable dans une image Docker multi-stage optimisée
2. THE Système SHALL fournir un fichier Docker Compose pour orchestrer l'application et PostgreSQL
3. THE Système SHALL utiliser GitHub Actions pour automatiser les tests, le build et la création de releases
4. WHEN un tag de version est créé, THEN LE Système SHALL automatiquement créer une release GitHub avec le package de déploiement
5. THE Système SHALL fournir un script de déploiement automatisé pour installer et démarrer l'application sur une VM
6. THE Système SHALL exposer un endpoint de health check pour vérifier l'état de l'application
7. THE Système SHALL supporter le déploiement via Terraform sur une infrastructure cloud
8. THE Système SHALL utiliser le versioning sémantique (vX.Y.Z) pour les releases
