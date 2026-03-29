.PHONY: run stop clear-db logs dashboard-dev install-sdk install-go-sdk test-go-sdk demo-go-sdk

# Run all services (ClickHouse + API)
run:
	docker-compose up --build -d
	@echo "Services starting..."
	@echo "  ClickHouse: http://localhost:8123"
	@echo "  API:        http://localhost:8080"
	@echo ""
	@echo "To start dashboard: make dashboard-dev"

# Run with dashboard
run-all: run dashboard-dev

# Start dashboard development server
dashboard-dev:
	cd dashboard && npm run dev -- --port 3001

# Stop all services
stop:
	docker-compose down
	@-pkill -f "npm run dev" 2>/dev/null || true
	@echo "All services stopped"

# Delete all data from ClickHouse tables
clear-db:
	@echo "Clearing all data from ClickHouse..."
	@curl -s "http://localhost:8123/" -u "default:password" -d "TRUNCATE TABLE IF EXISTS xray.xray_traces" || true
	@curl -s "http://localhost:8123/" -u "default:password" -d "TRUNCATE TABLE IF EXISTS xray.xray_events" || true
	@curl -s "http://localhost:8123/" -u "default:password" -d "TRUNCATE TABLE IF EXISTS xray.xray_decisions" || true
	@echo "All tables cleared!"

# Drop and recreate ClickHouse tables
drop-db:
	@echo "Dropping ClickHouse tables..."
	@curl -s "http://localhost:8123/" -u "default:password" -d "DROP TABLE IF EXISTS xray.xray_traces" || true
	@curl -s "http://localhost:8123/" -u "default:password" -d "DROP TABLE IF EXISTS xray.xray_events" || true
	@curl -s "http://localhost:8123/" -u "default:password" -d "DROP TABLE IF EXISTS xray.xray_decisions" || true
	@echo "Tables dropped! Run 'make run' to recreate them."

# View API logs
logs:
	docker-compose logs -f api

# Rebuild and restart API
rebuild:
	docker-compose up -d --build api

# Run the demo pipeline
demo:
	cd sdk/python/examples && python3 competitor_selection.py

# Install Python SDK in development mode
install-sdk:
	cd sdk/python && pip install -e .

# Prepare Go SDK dependencies
install-go-sdk:
	cd sdk/go && go mod tidy

# Run Go SDK tests
test-go-sdk:
	cd sdk/go && go test ./...

# Run Go SDK basic example
demo-go-sdk:
	cd sdk/go && go run ./examples/basic

# Help
help:
	@echo "X-Ray Makefile Commands:"
	@echo ""
	@echo "  make run          Start ClickHouse and API (docker-compose)"
	@echo "  make run-all      Start all services including dashboard"
	@echo "  make dashboard-dev Start dashboard dev server"
	@echo "  make stop         Stop all services"
	@echo "  make clear-db     Delete all data from ClickHouse tables"
	@echo "  make logs         Tail API logs"
	@echo "  make rebuild      Rebuild and restart API container"
	@echo "  make demo         Run the demo pipeline"
	@echo "  make install-sdk  Install Python SDK in dev mode"
	@echo "  make install-go-sdk Prepare Go SDK dependencies"
	@echo "  make test-go-sdk  Run Go SDK tests"
	@echo "  make demo-go-sdk  Run Go SDK example"
