#!/bin/bash

# VMManager Build Script

set -e

echo "Building VMManager..."

# Build backend
echo "Building backend..."
cd /Users/maxiliang/work/code/VMManager
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server/

# Build frontend
echo "Building frontend..."
cd web
npm install
npm run build

echo "Build completed!"
echo "Backend: ./server"
echo "Frontend: ./dist"
