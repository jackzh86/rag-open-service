@echo off
echo Building RAG Data Service and MCP Proxy...

:: Set environment variables
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1

:: Create build directory if it doesn't exist
if not exist "build" mkdir build

:: Run tests first
echo Running tests...
go test ./...
if %ERRORLEVEL% NEQ 0 (
    echo Tests failed!
    exit /b 1
)
echo All tests passed!

:: Build the RAG Data Service
echo Building RAG Data Service...
go build -o build/rag-data-service.exe ./cmd/main.go

:: Check if RAG service build was successful
if %ERRORLEVEL% NEQ 0 (
    echo RAG Data Service build failed!
    exit /b 1
)
echo RAG Data Service build successful!

:: Build the MCP Proxy
echo Building MCP Proxy...
go build -o build/rag-mcp-proxy.exe ./mcp-proxy/main.go

:: Check if MCP proxy build was successful
if %ERRORLEVEL% NEQ 0 (
    echo MCP Proxy build failed!
    exit /b 1
)
echo MCP Proxy build successful!

echo.
echo Build complete! Executables created:
echo   - build/rag-data-service.exe
echo   - build/rag-mcp-proxy.exe
echo Done! 