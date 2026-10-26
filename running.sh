#!/bin/bash

echo ""
echo "=== Stopping Docker Compose Services ==="
docker compose down

echo ""
echo "=== Building Docker Compose Services ==="
docker compose build --no-cache

echo ""
echo "=== Starting Docker Compose Services ==="
docker compose up -d

echo ""
echo "=== Printing Docker Compose Logs ==="
docker compose logs -f
