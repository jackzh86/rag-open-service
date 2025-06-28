# MCP Proxy

This proxy program converts stdio MCP communication to HTTP requests, enabling Claude Desktop to communicate with HTTP-based MCP servers.

## Build

The MCP proxy is now integrated into the main project build system. From the project root directory:

### Windows
```bash
build.bat
```

### Linux/Mac
```bash
./build.sh
```

This will build both the RAG Data Service and the MCP Proxy in one step.

## Usage

### 1. Build Both Programs
From the project root directory, run the build script:
```bash
# Windows
build.bat

# Linux/Mac
./build.sh
```

This creates:
- `build/rag-data-service.exe` (or `rag-data-service` on Unix)
- `build/rag-mcp-proxy.exe` (or `rag-mcp-proxy` on Unix)

### 2. Start the RAG Service
Ensure your RAG data service is running:
```bash
./build/rag-data-service.exe
```
The service should provide an MCP endpoint at `http://localhost:8080/mcp`.

### 3. Configure Claude Desktop
Copy the contents of `claude_desktop_config.json` (in this directory) to Claude Desktop's configuration file:

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Mac**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Important**: Update the `cwd` path to point to your actual project directory using an **absolute path**.

Configuration content:
```json
{
  "mcpServers": {
    "rag-data-service": {
      "command": "./build/rag-mcp-proxy.exe",
      "args": [],
      "env": {
        "MCP_HTTP_ENDPOINT": "http://localhost:8080/mcp"
      },
      "cwd": "/absolute/path/to/rag-data-service",
      "description": "RAG Data Service via stdio-to-HTTP proxy for Claude Desktop compatibility"
    }
  }
}
```

**Examples of absolute cwd paths**:
- Windows: `"D:\\workspace\\rag-data-service"`
- Mac/Linux: `"/Users/username/workspace/rag-data-service"`
- WSL: `"/mnt/d/workspace/rag-data-service"`

**Why absolute paths?** Relative paths like `".."` depend on where Claude Desktop starts execution, which can be unpredictable. Absolute paths ensure the MCP proxy always finds the correct build directory.

### 4. Restart Claude Desktop
After configuration, restart Claude Desktop. You should see a tools icon in the chat interface.

## Development

For development with hot reload:
```bash
# Windows
dev.bat

# Linux/Mac
./dev.sh
```

This will build both programs and start the RAG service with hot reload.

## How It Works

1. Claude Desktop communicates with the proxy program via stdio
2. The proxy program receives JSON-RPC requests
3. The proxy program forwards requests to the HTTP endpoint (`http://localhost:8080/mcp`)
4. The proxy program converts HTTP responses back to stdio format and returns them to Claude Desktop

## Environment Variables

- `MCP_HTTP_ENDPOINT`: URL of the HTTP MCP server endpoint (default: `http://localhost:8080/mcp`)

## Troubleshooting

1. **Ensure RAG service is running**: Accessing `http://localhost:8080/mcp` should return an MCP response
2. **Check proxy logs**: The proxy program outputs logs to stderr
3. **Verify configuration paths**: Ensure the path in Claude Desktop configuration points to the correct proxy executable
4. **Restart Claude Desktop**: Claude Desktop needs to be restarted after configuration changes

## Testing

You can manually test the proxy:

### Windows PowerShell
```powershell
echo '{"jsonrpc": "2.0", "method": "tools/list", "id": "test"}' | ./build/rag-mcp-proxy.exe
```

### Windows Command Prompt
```cmd
echo {"jsonrpc": "2.0", "method": "tools/list", "id": "test"} | .\build\rag-mcp-proxy.exe
```

### Linux/Mac
```bash
echo '{"jsonrpc": "2.0", "method": "tools/list", "id": "test"}' | ./build/rag-mcp-proxy
```

This should return a list of available tools.

**Note**: If you get a JSON parsing error on Windows, try using double quotes instead of single quotes, or escape the quotes properly for your shell.

## File Structure

- `main.go` - Main proxy program source code
- `claude_desktop_config.json` - Claude Desktop configuration for stdio mode
- `README.md` - This documentation

The proxy is built as part of the main project build system in the root directory and the executable is placed in `build/rag-mcp-proxy.exe`. 