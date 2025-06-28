#!/bin/bash

echo "Building RAG Data Service and MCP Proxy..."

# Set environment variables
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1

# Create build directory if it doesn't exist
mkdir -p build

# Run tests first
echo "Running tests..."
go test ./...
if [ $? -ne 0 ]; then
    echo "Tests failed!"
    exit 1
fi
echo "All tests passed!"

# Build the RAG Data Service
echo "Building RAG Data Service..."
go build -o build/rag-data-service ./cmd/main.go

# Check if RAG service build was successful
if [ $? -ne 0 ]; then
    echo "RAG Data Service build failed!"
    exit 1
fi
echo "RAG Data Service build successful!"

# Build the MCP Proxy
echo "Building MCP Proxy..."
go build -o build/rag-mcp-proxy ./mcp-proxy/main.go

# Check if MCP proxy build was successful
if [ $? -ne 0 ]; then
    echo "MCP Proxy build failed!"
    exit 1
fi
echo "MCP Proxy build successful!"

echo
echo "Build complete! Executables created:"
echo "  - build/rag-data-service"
echo "  - build/rag-mcp-proxy"
echo "Done!" 