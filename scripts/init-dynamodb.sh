#!/bin/bash

until aws dynamodb list-tables --endpoint-url http://dynamodb:8000 --region us-east-1 2>/dev/null; do
  sleep 1
done

if aws dynamodb describe-table --table-name xray_data --endpoint-url http://dynamodb:8000 --region us-east-1 2>/dev/null; then
  echo "Table xray_data already exists"
  exit 0
fi

echo "Creating xray_data table..."
aws dynamodb create-table \
  --endpoint-url http://dynamodb:8000 \
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
aws dynamodb update-time-to-live \
  --endpoint-url http://dynamodb:8000 \
  --region us-east-1 \
  --table-name xray_data \
  --time-to-live-specification "Enabled=true,AttributeName=ttl"

echo "Table created successfully"
