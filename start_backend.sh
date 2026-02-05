#!/bin/bash

# Couleurs pour les logs
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Starting Valhafin Backend${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Charger les variables d'environnement depuis .env
if [ -f .env ]; then
    echo -e "${GREEN}✓ Loading environment variables from .env${NC}"
    export $(cat .env | grep -v '^#' | xargs)
else
    echo -e "${YELLOW}⚠ .env file not found, using defaults${NC}"
    export DATABASE_URL="postgresql://valhafin:valhafin_dev_password@localhost:5432/valhafin_dev?sslmode=disable"
    export PORT="8080"
    export ENCRYPTION_KEY="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
fi

# Vérifier que PostgreSQL est en cours d'exécution
echo -e "${BLUE}Checking PostgreSQL connection...${NC}"
if ! docker ps | grep -q valhafin-postgres-dev; then
    echo -e "${YELLOW}⚠ PostgreSQL container not running, starting it...${NC}"
    docker-compose -f docker-compose.dev.yml up -d postgres
    echo -e "${GREEN}✓ Waiting for PostgreSQL to be ready...${NC}"
    sleep 5
fi

# Afficher les informations de connexion
echo -e "\n${GREEN}✓ Environment configured:${NC}"
echo -e "  Database: ${BLUE}$DATABASE_URL${NC}"
echo -e "  Port: ${BLUE}$PORT${NC}"
echo -e "\n${GREEN}🚀 Starting backend server...${NC}\n"

# Démarrer le serveur
go run main.go
