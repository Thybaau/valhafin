.PHONY: build run clean test install dev-db dev-backend dev-frontend dev setup

# Build the application
build:
	go build -o valhafin main.go

# Run the application (legacy CLI mode)
run:
	go run main.go

# Install dependencies
install:
	go mod download
	go mod tidy
	cd frontend && npm install

# Clean build artifacts
clean:
	rm -f valhafin
	rm -rf out/
	cd frontend && rm -rf dist node_modules

# Run tests
test:
	go test ./...
	cd frontend && npm test

# Development commands
dev-db:
	docker-compose -f docker-compose.dev.yml up -d

dev-db-stop:
	docker-compose -f docker-compose.dev.yml down

dev-backend:
	go run main.go

dev-frontend:
	cd frontend && npm run dev

# Start all development services
dev: dev-db
	@echo "✅ PostgreSQL started"
	@echo "🚀 Start backend with: make dev-backend"
	@echo "🎨 Start frontend with: make dev-frontend"

# Setup project (first time)
setup:
	@echo "📦 Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "📦 Installing frontend dependencies..."
	cd frontend && npm install
	@echo "🐘 Starting PostgreSQL..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "✅ Setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Copy .env.example to .env and configure"
	@echo "2. Run 'make dev-backend' to start the backend"
	@echo "3. Run 'make dev-frontend' to start the frontend"

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o valhafin-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o valhafin-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -o valhafin-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o valhafin-windows-amd64.exe main.go
