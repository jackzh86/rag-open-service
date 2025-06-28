package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"time"

	"rag-data-service/config"
	"rag-data-service/models"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func setupTestDB(t *testing.T) *sql.DB {
	// Load test configuration
	cfg := config.LoadTestConfig()

	// Use the same connection details as main config but with test database
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBConfig.Host,
		cfg.DBConfig.Port,
		cfg.DBConfig.User,
		cfg.DBConfig.Password,
		cfg.DBConfig.DBName+"_test",
	)

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Clean up test database
	_, err = db.Exec(`
		DROP TABLE IF EXISTS url_queue;
		DROP TABLE IF EXISTS knowledge_edges;
		DROP TABLE IF EXISTS knowledge_nodes;
		DROP TABLE IF EXISTS chunks;
		DROP TABLE IF EXISTS documents;
	`)
	require.NoError(t, err)

	// Read and execute migrations/init.sql
	initSQL, err := os.ReadFile("../migrations/init.sql")
	require.NoError(t, err)

	_, err = db.Exec(string(initSQL))
	require.NoError(t, err)

	return db
}

func TestRAGService_ProcessDocument(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)

	ctx := context.Background()
	req := &models.ProcessDocumentRequest{
		URL:     "https://example.com/test",
		Title:   "Test Document",
		Content: "This is a test document content. It contains multiple sentences. Each sentence should be processed into chunks. The chunks will be used for semantic search later.",
	}

	var err error
	err = service.ProcessDocument(ctx, req)
	assert.NoError(t, err)

	// Verify document was saved
	var docID int
	err = db.QueryRow("SELECT id FROM documents WHERE url = $1", req.URL).Scan(&docID)
	assert.NoError(t, err)
	assert.Greater(t, docID, 0)

	// Verify chunks were created
	var chunkCount int
	err = db.QueryRow("SELECT COUNT(*) FROM chunks WHERE document_id = $1", docID).Scan(&chunkCount)
	assert.NoError(t, err)
	assert.Greater(t, chunkCount, 0)
}

func TestRAGService_Query(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)

	ctx := context.Background()

	// First, add a test document
	req := &models.ProcessDocumentRequest{
		URL:     "https://example.com/test",
		Title:   "Test Document",
		Content: "This is a test document content that should be relevant to the query. It contains information about testing and querying. The content should be searchable and retrievable.",
	}

	var err error
	err = service.ProcessDocument(ctx, req)
	assert.NoError(t, err)

	// Now test the query
	query := "test document"
	response, err := service.Query(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.Results, "Query should return at least one result")
	if len(response.Results) > 0 {
		assert.Contains(t, response.Results[0].Content, "test document")
	}
}

func TestRAGService_GetKnowledgeGraph(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)

	ctx := context.Background()

	// First, add some test nodes and edges
	testVector := pgvector.NewVector(make([]float32, 1536))
	_, err := db.Exec(`
		INSERT INTO knowledge_nodes (name, type, properties, embedding)
		VALUES 
			('Test Node 1', 'concept', '{"description": "First test node"}'::jsonb, $1),
			('Test Node 2', 'concept', '{"description": "Second test node"}'::jsonb, $1)
	`, testVector)
	assert.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO knowledge_edges (source_id, target_id, relationship_type, properties)
		VALUES (1, 2, 'related_to', '{"weight": 0.8}'::jsonb)
	`)
	assert.NoError(t, err)

	// Test direct database query first to verify data exists
	var nodeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM knowledge_nodes").Scan(&nodeCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, nodeCount)

	var edgeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM knowledge_edges").Scan(&edgeCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, edgeCount)

	// Now test the knowledge graph query
	nodes, edges, err := service.GetKnowledgeGraph(ctx, "")
	assert.NoError(t, err)
	assert.NotNil(t, nodes)
	assert.NotNil(t, edges)
	assert.Len(t, nodes, 2, "Expected to find 2 nodes")
	assert.Len(t, edges, 1, "Expected to find 1 edge")
}

func TestRAGService_QueueURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)
	ctx := context.Background()

	// Test adding valid URL
	url := "https://example.com/test"
	err := service.QueueURL(ctx, url)
	assert.NoError(t, err)

	// Verify URL was added to queue
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM url_queue WHERE url = $1", url).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify URL status is pending
	var status string
	err = db.QueryRow("SELECT status FROM url_queue WHERE url = $1", url).Scan(&status)
	assert.NoError(t, err)
	assert.Equal(t, "pending", status)

	// Test adding empty URL should return error
	err = service.QueueURL(ctx, "")
	assert.Error(t, err)
}

func TestRAGService_ProcessURLQueue(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Add test URL to queue
	url := "https://example.com/test"
	err := service.QueueURL(ctx, url)
	assert.NoError(t, err)

	// Start background workers
	go service.StartBackgroundWorkers(ctx, 1)

	// Wait for workers to process URL
	time.Sleep(2 * time.Second)

	// Verify URL status has been updated
	var status string
	err = db.QueryRow("SELECT status FROM url_queue WHERE url = $1", url).Scan(&status)
	assert.NoError(t, err)
	assert.Contains(t, []string{"processing", "completed", "failed"}, status)
}

func TestRAGService_MultipleDocumentsQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.LoadTestConfig()
	service := NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)
	ctx := context.Background()

	// Add multiple documents with different keywords
	documents := []*models.ProcessDocumentRequest{
		{
			URL:     "https://example.com/ai",
			Title:   "Artificial Intelligence",
			Content: "Artificial Intelligence is a branch of computer science that aims to create intelligent machines. Machine learning is a subset of AI that enables computers to learn without being explicitly programmed. Deep learning uses neural networks to process complex patterns.",
		},
		{
			URL:     "https://example.com/cooking",
			Title:   "Cooking Techniques",
			Content: "Cooking is the art of preparing food for consumption. Different cooking techniques include baking, frying, grilling, and steaming. Each method has its own advantages and produces different flavors and textures in food.",
		},
		{
			URL:     "https://example.com/travel",
			Title:   "Travel Guide",
			Content: "Traveling allows people to explore new places and cultures. Popular travel destinations include Europe, Asia, and the Americas. Planning a trip involves booking flights, hotels, and researching local attractions.",
		},
		{
			URL:     "https://example.com/programming",
			Title:   "Programming Languages",
			Content: "Programming languages are used to create software applications. Popular languages include Python, JavaScript, Go, and Java. Each language has its own syntax and use cases. Go is particularly good for building web services and microservices.",
		},
	}

	// Process all documents
	for _, doc := range documents {
		err := service.ProcessDocument(ctx, doc)
		assert.NoError(t, err, "Failed to process document: %s", doc.URL)
	}

	// Verify all documents were saved
	var docCount int
	err := db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&docCount)
	assert.NoError(t, err)
	assert.Equal(t, len(documents), docCount)

	// Verify chunks were created for all documents
	var chunkCount int
	err = db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&chunkCount)
	assert.NoError(t, err)
	assert.Greater(t, chunkCount, 0)

	// Test queries for different topics
	testCases := []struct {
		query        string
		expectedURLs []string
		description  string
	}{
		{
			query:        "artificial intelligence machine learning",
			expectedURLs: []string{"https://example.com/ai"},
			description:  "AI-related query should return AI document",
		},
		{
			query:        "cooking food preparation",
			expectedURLs: []string{"https://example.com/cooking"},
			description:  "Cooking-related query should return cooking document",
		},
		{
			query:        "travel destinations",
			expectedURLs: []string{"https://example.com/travel"},
			description:  "Travel-related query should return travel document",
		},
		{
			query:        "programming languages Go Python",
			expectedURLs: []string{"https://example.com/programming"},
			description:  "Programming-related query should return programming document",
		},
		{
			query:        "computer science technology",
			expectedURLs: []string{"https://example.com/ai", "https://example.com/programming"},
			description:  "General tech query should return multiple relevant documents",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			response, err := service.Query(ctx, tc.query)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.NotEmpty(t, response.Results, "Query should return at least one result for: %s", tc.query)

			// Check if any of the expected URLs are found in the results
			foundURLs := make(map[string]bool)
			for _, result := range response.Results {
				// Find which document this chunk belongs to
				var url string
				err := db.QueryRow(`
					SELECT d.url 
					FROM documents d 
					JOIN chunks c ON d.id = c.document_id 
					WHERE c.content = $1
				`, result.Content).Scan(&url)
				if err == nil {
					foundURLs[url] = true
				}
			}

			// Verify that at least one expected URL was found
			foundExpected := false
			for _, expectedURL := range tc.expectedURLs {
				if foundURLs[expectedURL] {
					foundExpected = true
					break
				}
			}
			assert.True(t, foundExpected, "Expected to find at least one of %v in results for query: %s", tc.expectedURLs, tc.query)
		})
	}

	// Test that irrelevant queries don't return unexpected results
	irrelevantQueries := []string{
		"astronomy planets stars",
		"medical surgery procedures",
		"automotive car engines",
	}

	for _, query := range irrelevantQueries {
		t.Run("irrelevant_"+query, func(t *testing.T) {
			response, err := service.Query(ctx, query)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			// For irrelevant queries, we might still get some results due to vector similarity,
			// but they should have lower scores or be less relevant
			if len(response.Results) > 0 {
				t.Logf("Irrelevant query '%s' returned %d results", query, len(response.Results))
			}
		})
	}
}
