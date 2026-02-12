#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if .env file exists
if [ ! -f .env ]; then
    log_error ".env file not found!"
    log_info "Please create a .env file based on .env.example"
    exit 1
fi

# Load environment variables
source .env

# Check required environment variables
REQUIRED_VARS=("POSTGRES_PASSWORD" "ENCRYPTION_KEY")
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        log_error "Required environment variable $var is not set!"
        exit 1
    fi
done

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed!"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    log_error "Docker Compose is not installed!"
    exit 1
fi

# Determine docker compose command
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

log_info "Starting Valhafin deployment..."

# Stop existing containers
log_info "Stopping existing containers..."
$DOCKER_COMPOSE down

# Build images
log_info "Building Docker images..."
$DOCKER_COMPOSE build --no-cache

# Start services
log_info "Starting services..."
$DOCKER_COMPOSE up -d

# Wait for services to be healthy
log_info "Waiting for services to be healthy..."
sleep 5

# Check backend health
MAX_RETRIES=30
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -f http://localhost:${BACKEND_PORT:-8080}/health &> /dev/null; then
        log_info "Backend is healthy!"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        log_error "Backend failed to start!"
        $DOCKER_COMPOSE logs backend
        exit 1
    fi
    sleep 2
done

# Check frontend health
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -f http://localhost:${FRONTEND_PORT:-80}/health &> /dev/null; then
        log_info "Frontend is healthy!"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
        log_error "Frontend failed to start!"
        $DOCKER_COMPOSE logs frontend
        exit 1
    fi
    sleep 2
done

log_info "Deployment completed successfully!"
log_info "Backend is running on http://localhost:${BACKEND_PORT:-8080}"
log_info "Frontend is running on http://localhost:${FRONTEND_PORT:-80}"
log_info ""
log_info "To view logs: $DOCKER_COMPOSE logs -f"
log_info "To stop services: $DOCKER_COMPOSE down"
