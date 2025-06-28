#!/bin/bash
echo "Building and starting RAG Data Service with hot reload..."
echo

# First build both executables
./build.sh
if [ $? -ne 0 ]; then
    echo "Build failed, cannot start dev server"
    exit 1
fi

echo
echo "Starting RAG Data Service with hot reload..."
air 