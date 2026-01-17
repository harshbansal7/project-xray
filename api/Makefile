.PHONY: build run test clean swagger docker

# Variables
BINARY_NAME=xray-api
MAIN_PATH=./cmd/server

# Build the binary
build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run the server locally
run:
	go run $(MAIN_PATH)/main.go

# Run with local ClickHouse
run-local:
	CLICKHOUSE_HOST=localhost CLICKHOUSE_PORT=9000 go run $(MAIN_PATH)/main.go

# Run tests
test:
	go test -v ./...

# Test with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Docker build
docker-build:
	docker build -t xray-api .

# Docker run with ClickHouse
docker-run:
	docker run -p 8080:8080 \
		-e CLICKHOUSE_HOST=host.docker.internal \
		-e CLICKHOUSE_PORT=9000 \
		xray-api
