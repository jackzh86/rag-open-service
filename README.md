# RAG Data Service

A Retrieval-Augmented Generation (RAG) service built in Go that handles document processing, semantic search, and knowledge graph operations.

## Features

- **Data Ingestion**
  - URL-based ETL pipeline
  - Automatic text chunking
  - Vector embeddings generation
  - Knowledge graph construction
  - PostgreSQL with pgvector storage

- **Data Retrieval**
  - MCP (Model Context Protocol) integration
  - Semantic search capabilities
  - Knowledge graph traversal
  - Context-aware retrieval

## Prerequisites

- Go 1.21 or later
- Node.js 16 or later (for the rag-console web UI)
- PostgreSQL 12 or later with pgvector extension
- OpenAI API key

Example steps to install pgvector extension on Ubuntu:
```
sudo apt-get install pgxnclient
pgxn install vector
pgxn load -d ragdb vector
```

## Configuration

Create a `.env` file in the project root with the following settings:

```env
# Database configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ragdb

# OpenAI configuration
OPENAI_API_KEY=your-openai-api-key-here
OPENAI_API_BASE_URL=https://api.openai.com/v1 

# MCP configuration
MCP_ENDPOINT=http://localhost:8080/mcp
```

## Get Started

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Set up the database:
   ```bash
   psql -h localhost -p 5432 -U postgres -d ragdb -f migrations/init.sql
   ```
4. Run the service:
   ```bash
   go run cmd/main.go
   ```

5. Start the RAG Console (Web UI):
   ```bash
   cd rag-console
   npm install
   npm start
   ```
   The console will be available at http://localhost:3000

## API Usage

### Submit URL for Background Processing
```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com"
  }'
```

### Submit Document with Content (Immediate Processing)
```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "title": "Example Document",
    "content": "This is the document content."
  }'
```

### Query the Service
```bash
curl -X POST http://localhost:8080/api/v1/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "your search query here"
  }'
```

### Get Knowledge Graph
```bash
curl "http://localhost:8080/api/v1/graph?query=your%20search%20query"
```

### Check URL Processing Status (if implemented)
```bash
curl "http://localhost:8080/api/v1/queue/status?url=https://example.com"
```

## API Endpoints

- `POST /api/v1/documents` - Process and store a document
  - Accepts URL-only for background processing
  - Accepts URL + content for immediate processing
- `POST /api/v1/query` - Perform semantic search
- `GET /api/v1/graph` - Retrieve knowledge graph for a query
- `GET /api/v1/queue/status` - Check URL processing status (if implemented)

## Development

### Running Tests

```bash
go test ./...
```

## License

MIT 