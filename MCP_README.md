# RAG Data Service MCP Integration

This document explains how to use the RAG Data Service as an MCP (Model Context Protocol) server with Claude Desktop and other MCP-compatible clients.

## Overview

The RAG Data Service now supports the MCP protocol, allowing it to be used as a tool by large language models like Claude Desktop. This enables AI assistants to:

- Process documents and add them to the knowledge base
- Query the knowledge base for relevant information
- Retrieve knowledge graphs with entities and relationships
- Manage URL processing queues

## Available Tools

### 1. process_document
Process a document by URL or content and store it in the knowledge base.

**Parameters:**
- `url` (required): URL of the document to process
- `title` (optional): Title of the document
- `content` (optional): Content of the document (if not provided, content will be fetched from URL)

**Example:**
```json
{
  "name": "process_document",
  "arguments": {
    "url": "https://example.com/article",
    "title": "Example Article"
  }
}
```

### 2. query_knowledge_base
Query the knowledge base for relevant information.

**Parameters:**
- `query` (required): The query to search for in the knowledge base

**Example:**
```json
{
  "name": "query_knowledge_base",
  "arguments": {
    "query": "What is artificial intelligence?"
  }
}
```

### 3. get_knowledge_graph
Get the knowledge graph with entities and relationships.

**Parameters:**
- `document_id` (optional): Document ID to get graph for specific document

**Example:**
```json
{
  "name": "get_knowledge_graph",
  "arguments": {
    "document_id": 42
  }
}
```

### 4. queue_url
Add a URL to the processing queue for background processing.

**Parameters:**
- `url` (required): URL to add to the processing queue

**Example:**
```json
{
  "name": "queue_url",
  "arguments": {
    "url": "https://example.com/article"
  }
}
```

### 5. get_queue_status
Get the status of URLs in the processing queue.

**Parameters:** None

**Example:**
```json
{
  "name": "get_queue_status",
  "arguments": {}
}
```

## Setup Instructions

### For Claude Desktop

1. **Build the service:**
   ```bash
   go build ./cmd
   ```

2. **Configure Claude Desktop:**
   - Open Claude Desktop settings
   - Go to the MCP section
   - Add a new server configuration:
     ```json
     {
       "command": "./rag-data-service",
       "args": [],
       "env": {
         "PORT": "8080"
       }
     }
     ```

3. **Start the service:**
   ```bash
   ./rag-data-service
   ```

### For Other MCP Clients

1. **Use the provided config file:**
   Copy `mcp-config.json` to your MCP client's configuration directory.

2. **Or configure manually:**
   - Set the command to `rag-data-service`
   - Set the endpoint to `http://localhost:8080/mcp`
   - Ensure the service is running on port 8080

## Usage Examples

### Processing Documents

You can ask Claude to process documents:

> "Please process this article about AI: https://example.com/ai-article"

Claude will use the `process_document` tool to add the article to your knowledge base.

### Querying Knowledge Base

You can ask Claude to search your knowledge base:

> "What information do you have about machine learning?"

Claude will use the `query_knowledge_base` tool to search for relevant information.

### Getting Knowledge Graph

You can ask Claude to show you the knowledge graph:

> "Show me the knowledge graph for document 42"

Claude will use the `get_knowledge_graph` tool to retrieve the entities and relationships.

## API Endpoints

The MCP server also exposes the following HTTP endpoints for direct use:

- `POST /mcp` - MCP protocol endpoint
- `POST /api/v1/documents` - Process documents
- `POST /api/v1/query` - Query knowledge base
- `GET /api/v1/knowledge-graph` - Get knowledge graph
- `GET /api/v1/queue` - Get queue status

## Troubleshooting

### Common Issues

1. **Service not starting:**
   - Check database connection
   - Ensure all environment variables are set
   - Check port availability

2. **MCP connection failed:**
   - Verify the service is running on the correct port
   - Check MCP client configuration
   - Ensure the `/mcp` endpoint is accessible

3. **Tool calls failing:**
   - Check service logs for detailed error messages
   - Verify database schema is up to date
   - Ensure background workers are running

### Logs

The service provides detailed logging. Check the console output for:
- MCP request/response logs
- Database operation logs
- Background worker logs
- Error messages

## Security Considerations

- The MCP server runs on HTTP by default
- For production use, consider:
  - Using HTTPS
  - Adding authentication
  - Restricting access to trusted clients
  - Using environment variables for sensitive configuration

## Development

To extend the MCP functionality:

1. Add new tools to the `handleToolsList` method in `mcp/mcp.go`
2. Implement the tool handler in the `handleToolsCall` method
3. Add corresponding service methods if needed
4. Update this documentation

## Support

For issues and questions:
- Check the service logs
- Review the MCP protocol specification
- Check the main README for general service information 