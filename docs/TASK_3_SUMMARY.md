# Tâche 3 : Service de chiffrement et sécurité

## ✅ Complété

Cette tâche a mis en place le service de chiffrement AES-256-GCM pour sécuriser les identifiants et clés API des comptes financiers, avec une suite complète de tests incluant des tests basés sur les propriétés (Property-Based Testing).

## Ce qui a été fait

### 1. Service de chiffrement AES-256-GCM (Subtask 3.1)

#### ✅ services/encryption.go
Service de chiffrement complet avec AES-256-GCM :

**Structure `EncryptionService`** :
- Encapsule une clé de chiffrement de 32 bytes (256 bits)
- Fournit des méthodes thread-safe pour chiffrer et déchiffrer

**Fonction `NewEncryptionService(key []byte)`** :
- Crée une nouvelle instance du service
- Valide que la clé fait exactement 32 bytes
- Retourne une erreur si la clé est invalide

**Méthode `Encrypt(plaintext string)`** :
- Chiffre le texte en clair avec AES-256-GCM
- Génère un nonce aléatoire unique pour chaque chiffrement
- Ajoute automatiquement le tag d'authentification (GCM)
- Retourne le résultat encodé en base64 : `nonce + ciphertext + tag`
- Gère le cas spécial des chaînes vides

**Méthode `Decrypt(ciphertext string)`** :
- Décode le base64
- Extrait le nonce et le ciphertext
- Déchiffre et vérifie le tag d'authentification
- Retourne le texte en clair original
- Détecte les tentatives de modification (authentification GCM)

**Erreurs personnalisées** :
- `ErrInvalidKeySize` : Clé non conforme (doit être 32 bytes)
- `ErrInvalidCiphertext` : Ciphertext malformé ou trop court
- `ErrDecryptionFailed` : Échec du déchiffrement ou authentification échouée

#### ✅ services/encryption_key.go
Gestion sécurisée des clés de chiffrement :

**Fonction `LoadEncryptionKeyFromEnv()`** :
- Charge la clé depuis la variable d'environnement `ENCRYPTION_KEY`
- Attend une chaîne hexadécimale de 64 caractères (32 bytes)
- Valide le format et la longueur
- Retourne la clé décodée prête à l'emploi

**Fonction `GenerateEncryptionKey()`** :
- Génère une nouvelle clé aléatoire de 32 bytes
- Utilise `crypto/rand` pour une génération cryptographiquement sécurisée
- Retourne la clé encodée en hexadécimal (64 caractères)
- Idéal pour la génération initiale de clé

**Erreurs personnalisées** :
- `ErrKeyNotSet` : Variable d'environnement non définie
- `ErrInvalidKeyFormat` : Format hexadécimal invalide ou longueur incorrecte

#### ✅ services/README.md
Documentation complète du service :
- Guide d'utilisation avec exemples de code
- Instructions de génération et stockage de clé
- Considérations de sécurité détaillées
- Référence API complète
- Guide de rotation de clé
- Documentation de la Propriété 22

### 2. Tests de propriété pour le chiffrement (Subtask 3.2)

#### ✅ services/encryption_test.go
Suite de tests complète avec tests unitaires et tests basés sur les propriétés :

**Tests unitaires de base** :

1. `TestEncryptionServiceCreation` :
   - ✅ Création avec clé valide de 32 bytes
   - ✅ Rejet des clés trop courtes (16 bytes)
   - ✅ Rejet des clés trop longues (64 bytes)
   - ✅ Rejet des clés vides

2. `TestEncryptDecryptBasic` :
   - ✅ Texte simple : "hello world"
   - ✅ Chaîne vide : ""
   - ✅ Caractères spéciaux : "!@#$%^&*()..."
   - ✅ Unicode : "Hello 世界 🌍"
   - ✅ Texte long : 10,000 caractères
   - ✅ Format credentials JSON : `{"username":"...","password":"..."}`

3. `TestEncryptionUniqueness` :
   - ✅ Même texte chiffré 100 fois produit 100 ciphertexts différents
   - ✅ Tous les ciphertexts se déchiffrent correctement
   - ✅ Vérifie l'unicité grâce aux nonces aléatoires

4. `TestDecryptInvalidData` :
   - ✅ Base64 invalide : rejeté
   - ✅ Données trop courtes : rejeté
   - ✅ Données aléatoires : rejeté
   - ✅ Chaîne vide : accepté (cas spécial)

5. `TestDifferentKeysCannotDecrypt` :
   - ✅ Données chiffrées avec clé A ne peuvent pas être déchiffrées avec clé B
   - ✅ Isolation complète entre différentes clés

**Tests basés sur les propriétés (Property-Based Testing)** :

**Propriété 22 : Round-trip chiffrement/déchiffrement**
**Valide : Exigences 1.5**

`TestProperty_RoundTripEncryptionDecryption` avec 4 propriétés vérifiées :

1. **Propriété de round-trip** (100 tests) :
   ```
   ∀ plaintext : decrypt(encrypt(plaintext)) = plaintext
   ```
   - Pour toute chaîne générée aléatoirement
   - Le chiffrement suivi du déchiffrement retourne exactement l'original
   - Aucune perte de données

2. **Propriété de non-identité** (100 tests) :
   ```
   ∀ plaintext ≠ "" : encrypt(plaintext) ≠ plaintext
   ```
   - Le ciphertext est toujours différent du plaintext
   - Exception : chaîne vide (cas spécial)

3. **Propriété d'unicité** (100 tests) :
   ```
   ∀ plaintext : encrypt(plaintext) ≠ encrypt(plaintext)
   ```
   - Deux chiffrements du même texte produisent des résultats différents
   - Grâce aux nonces aléatoires
   - Mais les deux se déchiffrent vers le même plaintext

4. **Propriété de préservation de longueur** (100 tests) :
   ```
   ∀ plaintext : len(decrypt(encrypt(plaintext))) = len(plaintext)
   ```
   - La longueur du texte est préservée après round-trip
   - Pas de padding ou troncature

**Bibliothèque utilisée** : `github.com/leanovate/gopter` v0.2.11
- Framework de Property-Based Testing pour Go
- Génération automatique de cas de test aléatoires
- Shrinking automatique en cas d'échec

#### ✅ services/encryption_example_test.go
Exemples de code exécutables :

- `Example()` : Génération d'une clé de chiffrement
- `ExampleEncryptionService_Encrypt()` : Chiffrement de credentials
- `ExampleEncryptionService_Decrypt()` : Déchiffrement et vérification

Ces exemples apparaissent dans la documentation Go (`go doc`).

## Structure des fichiers créés

```
valhafin/
├── services/
│   ├── encryption.go              # Service de chiffrement AES-256-GCM
│   ├── encryption_key.go          # Gestion des clés (env, génération)
│   ├── encryption_test.go         # Tests unitaires + PBT (400+ lignes)
│   ├── encryption_example_test.go # Exemples de code
│   └── README.md                  # Documentation complète
└── go.mod                         # + gopter v0.2.11
```

## Tests effectués

### Tests unitaires et Property-Based Tests
```bash
$ go test -v ./services/...
=== RUN   TestEncryptionServiceCreation
--- PASS: TestEncryptionServiceCreation (0.00s)
=== RUN   TestEncryptDecryptBasic
--- PASS: TestEncryptDecryptBasic (0.00s)
=== RUN   TestEncryptionUniqueness
--- PASS: TestEncryptionUniqueness (0.00s)
=== RUN   TestDecryptInvalidData
--- PASS: TestDecryptInvalidData (0.00s)
=== RUN   TestDifferentKeysCannotDecrypt
--- PASS: TestDifferentKeysCannotDecrypt (0.00s)
=== RUN   TestProperty_RoundTripEncryptionDecryption
+ encrypt then decrypt returns original plaintext: OK, passed 100 tests.
+ encrypted data is different from plaintext (except empty): OK, passed 100 tests.
+ same plaintext produces different ciphertexts: OK, passed 100 tests.
+ encryption preserves data length information: OK, passed 100 tests.
--- PASS: TestProperty_RoundTripEncryptionDecryption (0.01s)
PASS
ok      valhafin/services       0.536s
```

### Couverture de code
```bash
$ go test ./services/... -cover
ok      valhafin/services       0.412s  coverage: 63.3% of statements
```

### Compilation
```bash
$ go build ./services/...
✅ Build successful
```

## Caractéristiques de sécurité

### Algorithme : AES-256-GCM

**AES-256** :
- Chiffrement par bloc symétrique
- Clé de 256 bits (32 bytes)
- Standard industriel approuvé par le NIST
- Résistant aux attaques connues

**Mode GCM (Galois/Counter Mode)** :
- Chiffrement authentifié (AEAD)
- Fournit à la fois confidentialité et intégrité
- Détecte toute modification du ciphertext
- Tag d'authentification de 128 bits
- Parallélisable et performant

### Nonce aléatoire

- Génération d'un nonce unique de 12 bytes pour chaque chiffrement
- Utilise `crypto/rand` (générateur cryptographiquement sécurisé)
- Garantit que le même plaintext produit des ciphertexts différents
- Prévient les attaques par analyse de fréquence

### Encodage base64

- Ciphertext encodé en base64 pour stockage sûr
- Compatible avec les bases de données SQL (TEXT/VARCHAR)
- Pas de caractères spéciaux problématiques
- Format : `base64(nonce || ciphertext || tag)`

### Gestion des clés

**Génération** :
- Clés générées avec `crypto/rand`
- 32 bytes (256 bits) d'entropie
- Format hexadécimal pour faciliter le stockage

**Stockage** :
- Variable d'environnement `ENCRYPTION_KEY`
- Jamais committée dans le code source
- Jamais loggée ou exposée
- Rotation possible via migration

**Validation** :
- Vérification stricte de la longueur (32 bytes)
- Validation du format hexadécimal
- Erreurs explicites en cas de problème

## Utilisation dans l'application

### Configuration initiale

```bash
# 1. Générer une clé de chiffrement
go run -c 'package main; import "valhafin/services"; func main() { 
    key, _ := services.GenerateEncryptionKey(); 
    println(key) 
}'

# 2. Ajouter à .env
echo "ENCRYPTION_KEY=<votre_clé_64_caractères>" >> .env
```

### Intégration dans le code

```go
package main

import (
    "log"
    "valhafin/services"
)

func main() {
    // Charger la clé depuis l'environnement
    key, err := services.LoadEncryptionKeyFromEnv()
    if err != nil {
        log.Fatal("Failed to load encryption key:", err)
    }

    // Créer le service de chiffrement
    encryptionService, err := services.NewEncryptionService(key)
    if err != nil {
        log.Fatal("Failed to create encryption service:", err)
    }

    // Chiffrer les credentials avant stockage
    credentials := `{"username":"user123","password":"secret","pin":"1234"}`
    encrypted, err := encryptionService.Encrypt(credentials)
    if err != nil {
        log.Fatal("Encryption failed:", err)
    }

    // Stocker 'encrypted' dans la base de données
    // ...

    // Plus tard, déchiffrer pour utilisation
    decrypted, err := encryptionService.Decrypt(encrypted)
    if err != nil {
        log.Fatal("Decryption failed:", err)
    }

    // Utiliser les credentials déchiffrés
    // ...
}
```

### Intégration avec les modèles

Le service sera utilisé dans la couche d'accès aux données :

```go
// database/accounts.go
func (db *DB) CreateAccount(account *models.Account, encryptionService *services.EncryptionService) error {
    // Chiffrer les credentials avant insertion
    encrypted, err := encryptionService.Encrypt(account.Credentials)
    if err != nil {
        return fmt.Errorf("failed to encrypt credentials: %w", err)
    }
    
    account.Credentials = encrypted
    
    // Insérer dans la base de données
    // ...
}

func (db *DB) GetAccountByID(id string, encryptionService *services.EncryptionService) (*models.Account, error) {
    // Récupérer depuis la base de données
    // ...
    
    // Déchiffrer les credentials
    decrypted, err := encryptionService.Decrypt(account.Credentials)
    if err != nil {
        return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
    }
    
    account.Credentials = decrypted
    return account, nil
}
```

## Propriété de correction validée

### Propriété 22 : Round-trip chiffrement/déchiffrement

**Énoncé formel** :
> Pour tout identifiant ou clé API chiffré et stocké, le déchiffrement doit retourner exactement la valeur originale, et aucune perte de données ne doit survenir lors du round-trip chiffrement → stockage → déchiffrement.

**Validation** : ✅ Vérifiée par Property-Based Testing
- 100 tests avec chaînes aléatoires
- Tous les cas passent avec succès
- Aucune perte de données détectée
- Préservation exacte du contenu et de la longueur

**Exigences satisfaites** : ✅ Exigence 1.5

## Prochaines étapes

La tâche 3 est terminée. Vous pouvez maintenant passer à la **Tâche 4 : API REST - Gestion des comptes**.

Le service de chiffrement est prêt à être intégré dans :
- La création de comptes (chiffrement des credentials)
- La récupération de comptes (déchiffrement pour utilisation)
- Les scrapers (déchiffrement pour authentification)

### Exemple d'intégration dans l'API

```go
// api/handlers.go
func (h *Handler) CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
    var account models.Account
    json.NewDecoder(r.Body).Decode(&account)
    
    // Chiffrer les credentials
    encrypted, err := h.encryptionService.Encrypt(account.Credentials)
    if err != nil {
        http.Error(w, "Encryption failed", http.StatusInternalServerError)
        return
    }
    
    account.Credentials = encrypted
    
    // Sauvegarder dans la base de données
    err = h.db.CreateAccount(&account)
    // ...
}
```

## Exigences satisfaites

- ✅ **Exigence 1.5** : Chiffrement des identifiants et clés API avant stockage
  - AES-256-GCM implémenté
  - Nonces aléatoires uniques
  - Authentification intégrée
  - Validation par Property-Based Testing

## Dépendances ajoutées

- ✅ `github.com/leanovate/gopter` v0.2.11 : Property-Based Testing framework

## Notes techniques

### Choix de conception

1. **AES-256-GCM** : Choisi pour sa sécurité éprouvée et son authentification intégrée. GCM détecte automatiquement toute modification du ciphertext.

2. **Nonces aléatoires** : Chaque chiffrement génère un nouveau nonce, garantissant que le même plaintext produit des ciphertexts différents. Essentiel pour la sécurité.

3. **Base64 encoding** : Permet un stockage sûr dans les bases de données SQL sans problème de caractères spéciaux.

4. **Clé en environnement** : Séparation de la clé du code source, suivant les meilleures pratiques de sécurité (12-factor app).

5. **Property-Based Testing** : Valide les propriétés universelles du chiffrement sur des centaines de cas générés aléatoirement, offrant une meilleure couverture que les tests unitaires seuls.

### Avantages de GCM

- **Authentification** : Détecte toute modification du ciphertext
- **Performance** : Parallélisable, plus rapide que CBC
- **Sécurité** : Résistant aux attaques par padding oracle
- **Standard** : Recommandé par le NIST et largement utilisé

### Rotation de clé

Si vous devez changer la clé de chiffrement :

```go
// 1. Charger l'ancienne et la nouvelle clé
oldKey, _ := hex.DecodeString(os.Getenv("OLD_ENCRYPTION_KEY"))
newKey, _ := hex.DecodeString(os.Getenv("NEW_ENCRYPTION_KEY"))

oldService, _ := services.NewEncryptionService(oldKey)
newService, _ := services.NewEncryptionService(newKey)

// 2. Pour chaque compte
accounts, _ := db.GetAllAccounts()
for _, account := range accounts {
    // Déchiffrer avec l'ancienne clé
    decrypted, _ := oldService.Decrypt(account.Credentials)
    
    // Re-chiffrer avec la nouvelle clé
    encrypted, _ := newService.Encrypt(decrypted)
    
    // Mettre à jour en base
    account.Credentials = encrypted
    db.UpdateAccount(&account)
}

// 3. Mettre à jour ENCRYPTION_KEY avec la nouvelle clé
```

### Améliorations futures possibles

- Ajouter un système de versioning des clés pour faciliter la rotation
- Implémenter un cache de clés déchiffrées en mémoire (avec TTL)
- Ajouter des métriques de performance (temps de chiffrement/déchiffrement)
- Support de multiples algorithmes de chiffrement (pour migration future)
- Intégration avec des systèmes de gestion de clés (AWS KMS, HashiCorp Vault)

