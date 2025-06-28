@echo off
echo Building and starting RAG Data Service with hot reload...
echo.

:: First build both executables
call build.bat
if %ERRORLEVEL% NEQ 0 (
    echo Build failed, cannot start dev server
    exit /b 1
)

echo.
echo Starting RAG Data Service with hot reload...
air 