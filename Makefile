.PHONY: build run clean test install

# Build the application
build:
	go build -o valhafin main.go

# Run the application
run:
	go run main.go

# Install dependencies
install:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f valhafin
	rm -rf out/

# Run tests
test:
	go test ./...

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o valhafin-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o valhafin-darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -o valhafin-linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o valhafin-windows-amd64.exe main.go
