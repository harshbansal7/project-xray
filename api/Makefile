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

# Run with local DynamoDB
run-local:
	AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local DYNAMODB_ENDPOINT=http://localhost:8000 go run $(MAIN_PATH)/main.go

dynamdb-admin:
	AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local DYNAMODB_ENDPOINT=http://localhost:8000 dynamodb-admin

# Generate Swagger docs
swagger:
	swag init -g cmd/server/main.go -o docs

# Install swagger CLI
install-swagger:
	go install github.com/swaggo/swag/cmd/swag@latest

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Start local DynamoDB
dynamodb-local:
	docker run -d -p 8000:8000 --name xray-dynamodb amazon/dynamodb-local

# Stop local DynamoDB
dynamodb-stop:
	docker stop xray-dynamodb && docker rm xray-dynamodb

# Delete existing table
delete-table:
	AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb delete-table \
		--endpoint-url http://localhost:8000 \
		--region us-east-1 \
		--table-name xray_data || true

create-table:
	AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb create-table \
		--endpoint-url http://localhost:8000 \
		--region us-east-1 \
		--table-name xray_data \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
			AttributeName=GSI2PK,AttributeType=S \
			AttributeName=GSI2SK,AttributeType=S \
			AttributeName=GSI3PK,AttributeType=S \
			AttributeName=GSI3SK,AttributeType=S \
			AttributeName=GSI4PK,AttributeType=S \
			AttributeName=GSI4SK,AttributeType=S \
			AttributeName=GSI5PK,AttributeType=S \
			AttributeName=GSI5SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--global-secondary-indexes \
			'[{"IndexName":"pipeline-time-index","KeySchema":[{"AttributeName":"GSI2PK","KeyType":"HASH"},{"AttributeName":"GSI2SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}},{"IndexName":"step-type-analytics-index","KeySchema":[{"AttributeName":"GSI3PK","KeyType":"HASH"},{"AttributeName":"GSI3SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}},{"IndexName":"event-decisions-index","KeySchema":[{"AttributeName":"GSI4PK","KeyType":"HASH"},{"AttributeName":"GSI4SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}},{"IndexName":"item-history-index","KeySchema":[{"AttributeName":"GSI5PK","KeyType":"HASH"},{"AttributeName":"GSI5SK","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
		--billing-mode PAY_PER_REQUEST
	sleep 2
	AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb update-time-to-live \
		--endpoint-url http://localhost:8000 \
		--region us-east-1 \
		--table-name xray_data \
		--time-to-live-specification "Enabled=true,AttributeName=ttl"


# Docker build
docker-build:
	docker build -t xray-api .

# Docker run
docker-run:
	docker run -p 8080:8080 \
		-e AWS_ACCESS_KEY_ID=local \
		-e AWS_SECRET_ACCESS_KEY=local \
		-e DYNAMODB_ENDPOINT=http://host.docker.internal:8000 \
		xray-api
