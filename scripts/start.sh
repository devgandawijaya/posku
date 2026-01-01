#!/bin/bash
echo "Starting posku..."
docker compose up -d --build
echo "App berjalan → http://localhost:2041"
echo "DB port     → localhost:2042"
