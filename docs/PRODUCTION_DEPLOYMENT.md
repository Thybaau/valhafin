# Guide de Déploiement en Production

## Vue d'Ensemble

Ce guide explique comment déployer Valhafin en production. En production, **les utilisateurs n'ont pas besoin de maintenir un fichier `.env`** - les variables d'environnement sont gérées par le système de déploiement (Docker, Kubernetes, VM, etc.).

## Différences Développement vs Production

### Développement (Local)

```bash
# Fichier .env à la racine (ignoré par git)
DATABASE_URL=postgresql://valhafin:valhafin@localhost:5432/valhafin_dev?sslmode=disable
PORT=8080
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef

# Démarrage simple
go run main.go
```

**Avantages:**
- ✅ Fichier `.env` facile à éditer
- ✅ Pas besoin de reconstruire pour changer une variable
- ✅ Chargé automatiquement par `godotenv`

### Production (Déployé)

Les variables d'environnement sont définies par:
- **Docker Compose**: Dans le fichier `docker-compose.yml`
- **Kubernetes**: Dans les ConfigMaps et Secrets
- **VM/Serveur**: Variables d'environnement système
- **Cloud (AWS/GCP/Azure)**: Variables d'environnement du service

**Avantages:**
- ✅ Pas de fichier `.env` à gérer
- ✅ Secrets gérés par le système (AWS Secrets Manager, Kubernetes Secrets, etc.)
- ✅ Variables injectées automatiquement au démarrage
- ✅ Pas de risque de commit de secrets

## Méthodes de Déploiement

### Méthode 1: Docker Compose (Recommandé)

**Fichier: `docker-compose.yml`**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: valhafin-postgres
    environment:
      POSTGRES_DB: valhafin
      POSTGRES_USER: valhafin
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}  # Depuis variable d'environnement système
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  valhafin:
    image: ghcr.io/your-org/valhafin:latest
    container_name: valhafin-app
    environment:
      DATABASE_URL: postgresql://valhafin:${POSTGRES_PASSWORD}@postgres:5432/valhafin?sslmode=disable
      PORT: 8080
      ENCRYPTION_KEY: ${ENCRYPTION_KEY}  # Depuis variable d'environnement système
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
```

**Déploiement:**

```bash
# Sur le serveur, définir les variables d'environnement
export POSTGRES_PASSWORD="votre_mot_de_passe_securise"
export ENCRYPTION_KEY="votre_cle_de_chiffrement_32_bytes"

# Démarrer
docker-compose up -d

# Les variables sont injectées automatiquement dans les conteneurs
# Pas besoin de fichier .env!
```

### Méthode 2: Kubernetes

**Fichier: `k8s/deployment.yaml`**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: valhafin-secrets
type: Opaque
stringData:
  postgres-password: "votre_mot_de_passe"
  encryption-key: "votre_cle_32_bytes"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: valhafin
spec:
  replicas: 2
  selector:
    matchLabels:
      app: valhafin
  template:
    metadata:
      labels:
        app: valhafin
    spec:
      containers:
      - name: valhafin
        image: ghcr.io/your-org/valhafin:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          value: "postgresql://valhafin:$(POSTGRES_PASSWORD)@postgres:5432/valhafin"
        - name: PORT
          value: "8080"
        - name: ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: valhafin-secrets
              key: encryption-key
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: valhafin-secrets
              key: postgres-password
```

**Déploiement:**

```bash
# Appliquer la configuration
kubectl apply -f k8s/

# Les secrets sont gérés par Kubernetes
# Pas besoin de fichier .env!
```

### Méthode 3: VM avec Systemd

**Fichier: `/etc/systemd/system/valhafin.service`**

```ini
[Unit]
Description=Valhafin Portfolio Manager
After=network.target postgresql.service

[Service]
Type=simple
User=valhafin
WorkingDirectory=/opt/valhafin
ExecStart=/opt/valhafin/valhafin
Restart=always

# Variables d'environnement
Environment="DATABASE_URL=postgresql://valhafin:PASSWORD@localhost:5432/valhafin"
Environment="PORT=8080"
Environment="ENCRYPTION_KEY=YOUR_32_BYTE_KEY"

# Ou charger depuis un fichier (mais pas .env dans le repo!)
EnvironmentFile=/etc/valhafin/environment

[Install]
WantedBy=multi-user.target
```

**Fichier: `/etc/valhafin/environment`** (géré par l'admin système)

```bash
DATABASE_URL=postgresql://valhafin:PASSWORD@localhost:5432/valhafin
PORT=8080
ENCRYPTION_KEY=YOUR_32_BYTE_KEY
```

**Déploiement:**

```bash
# Installer le service
sudo systemctl daemon-reload
sudo systemctl enable valhafin
sudo systemctl start valhafin

# Les variables sont dans /etc/valhafin/environment
# Pas besoin de .env dans le code!
```

### Méthode 4: Cloud Providers

#### AWS (ECS/Fargate)

```json
{
  "containerDefinitions": [
    {
      "name": "valhafin",
      "image": "ghcr.io/your-org/valhafin:latest",
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:valhafin/database-url"
        },
        {
          "name": "ENCRYPTION_KEY",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:valhafin/encryption-key"
        }
      ]
    }
  ]
}
```

Les secrets sont gérés par **AWS Secrets Manager**.

#### Google Cloud (Cloud Run)

```bash
gcloud run deploy valhafin \
  --image ghcr.io/your-org/valhafin:latest \
  --set-env-vars PORT=8080 \
  --set-secrets DATABASE_URL=valhafin-database-url:latest \
  --set-secrets ENCRYPTION_KEY=valhafin-encryption-key:latest
```

Les secrets sont gérés par **Google Secret Manager**.

#### Azure (Container Instances)

```bash
az container create \
  --resource-group valhafin-rg \
  --name valhafin \
  --image ghcr.io/your-org/valhafin:latest \
  --environment-variables PORT=8080 \
  --secure-environment-variables \
    DATABASE_URL=$DATABASE_URL \
    ENCRYPTION_KEY=$ENCRYPTION_KEY
```

Les secrets sont gérés par **Azure Key Vault**.

## Gestion des Secrets en Production

### ❌ À NE PAS FAIRE

```bash
# NE PAS commiter un .env en production
git add .env  # ❌ DANGER!

# NE PAS hardcoder les secrets dans le code
const encryptionKey = "0123456789abcdef..."  // ❌ DANGER!

# NE PAS mettre les secrets dans docker-compose.yml
environment:
  ENCRYPTION_KEY: "0123456789abcdef..."  # ❌ DANGER!
```

### ✅ À FAIRE

```bash
# Utiliser des variables d'environnement système
export ENCRYPTION_KEY="..."
docker-compose up -d

# Utiliser un gestionnaire de secrets
aws secretsmanager create-secret --name valhafin/encryption-key --secret-string "..."

# Utiliser un fichier de configuration système (hors du repo)
# /etc/valhafin/environment (permissions 600, propriétaire root)
```

## Processus de Release

### 1. Créer une Release

```bash
# 1. Tagger la version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. GitHub Actions build automatiquement:
#    - Exécute les tests
#    - Build l'image Docker
#    - Push sur ghcr.io
#    - Crée la release GitHub
```

### 2. Déployer la Release

#### Avec Docker Compose

```bash
# Sur le serveur
cd /opt/valhafin

# Télécharger la nouvelle version
docker pull ghcr.io/your-org/valhafin:v1.0.0

# Mettre à jour docker-compose.yml
sed -i 's/:latest/:v1.0.0/' docker-compose.yml

# Redémarrer
docker-compose up -d

# Vérifier
curl http://localhost:8080/health
```

#### Avec Kubernetes

```bash
# Mettre à jour l'image
kubectl set image deployment/valhafin valhafin=ghcr.io/your-org/valhafin:v1.0.0

# Vérifier le rollout
kubectl rollout status deployment/valhafin
```

## Configuration Minimale Requise

### Variables d'Environnement Obligatoires

```bash
# Base de données (obligatoire)
DATABASE_URL=postgresql://user:password@host:port/database?sslmode=disable

# Port d'écoute (optionnel, défaut: 8080)
PORT=8080

# Clé de chiffrement (obligatoire, 32 bytes)
ENCRYPTION_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

### Variables d'Environnement Optionnelles

```bash
# API Yahoo Finance (optionnel)
YAHOO_FINANCE_API_KEY=your_api_key

# Niveau de log (optionnel, défaut: info)
LOG_LEVEL=info

# Environnement (optionnel, défaut: production)
ENVIRONMENT=production
```

## Génération de la Clé de Chiffrement

La clé de chiffrement doit faire **exactement 32 bytes** (256 bits) pour AES-256-GCM.

### Méthode 1: OpenSSL

```bash
openssl rand -hex 32
# Output: 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

### Méthode 2: Go

```bash
go run -c 'package main; import ("crypto/rand"; "encoding/hex"; "fmt"); func main() { key := make([]byte, 32); rand.Read(key); fmt.Println(hex.EncodeToString(key)) }'
```

### Méthode 3: Python

```bash
python3 -c "import secrets; print(secrets.token_hex(32))"
```

## Monitoring et Health Checks

### Health Check Endpoint

```bash
curl http://localhost:8080/health
```

**Réponse:**
```json
{
  "status": "healthy",
  "database": "up",
  "version": "v1.0.0",
  "uptime": "2h30m15s"
}
```

### Intégration avec Docker

```yaml
healthcheck:
  test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

### Intégration avec Kubernetes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Backup et Restauration

### Backup Automatique (Cron)

```bash
#!/bin/bash
# /opt/valhafin/backup.sh

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Backup PostgreSQL
docker exec valhafin-postgres pg_dump -U valhafin valhafin > $BACKUP_DIR/valhafin_$DATE.sql
gzip $BACKUP_DIR/valhafin_$DATE.sql

# Garder seulement les 7 derniers backups
find $BACKUP_DIR -name "valhafin_*.sql.gz" -mtime +7 -delete

# Upload vers S3 (optionnel)
aws s3 cp $BACKUP_DIR/valhafin_$DATE.sql.gz s3://valhafin-backups/
```

**Crontab:**
```bash
# Backup quotidien à 2h du matin
0 2 * * * /opt/valhafin/backup.sh
```

### Restauration

```bash
# Décompresser le backup
gunzip valhafin_20260130_020000.sql.gz

# Restaurer
docker exec -i valhafin-postgres psql -U valhafin valhafin < valhafin_20260130_020000.sql
```

## Logs en Production

### Avec Docker Compose

```bash
# Voir les logs
docker-compose logs -f valhafin

# Logs des dernières 100 lignes
docker-compose logs --tail=100 valhafin

# Logs depuis une date
docker-compose logs --since="2026-01-30T10:00:00" valhafin
```

### Avec Systemd

```bash
# Voir les logs
journalctl -u valhafin -f

# Logs des dernières 100 lignes
journalctl -u valhafin -n 100

# Logs depuis une date
journalctl -u valhafin --since="2026-01-30 10:00:00"
```

## Sécurité en Production

### Checklist

- [ ] Clé de chiffrement générée aléatoirement (32 bytes)
- [ ] Mot de passe PostgreSQL fort et unique
- [ ] Variables d'environnement gérées par un gestionnaire de secrets
- [ ] Pas de fichier `.env` dans le repo
- [ ] HTTPS activé (reverse proxy nginx/traefik)
- [ ] Firewall configuré (seulement ports 80/443 ouverts)
- [ ] Backups automatiques configurés
- [ ] Monitoring et alertes configurés
- [ ] Logs centralisés (ELK, Loki, CloudWatch, etc.)

## Résumé

### Développement

```bash
# Fichier .env à la racine
DATABASE_URL=...
ENCRYPTION_KEY=...

# Démarrage simple
go run main.go
```

### Production

```bash
# Variables d'environnement système
export DATABASE_URL=...
export ENCRYPTION_KEY=...

# Ou Docker Compose
docker-compose up -d

# Ou Kubernetes
kubectl apply -f k8s/

# Ou Systemd
systemctl start valhafin
```

**Pas de fichier `.env` en production!** Les variables sont gérées par le système de déploiement.
