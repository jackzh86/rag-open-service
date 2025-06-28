package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"rag-data-service/config"
	"rag-data-service/models"

	"github.com/PuerkitoBio/goquery"
	"github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
)

// DB defines the database interface required by RAGService
type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// ChunkInfo represents chunk information with position data for internal processing
type ChunkInfo struct {
	Content       string `json:"content"`
	ChunkIndex    int    `json:"chunk_index"`
	StartPosition int    `json:"start_position"`
	EndPosition   int    `json:"end_position"`
}

// RAGService handles the RAG operations
type RAGService struct {
	db            DB
	openAIKey     string
	openAIBaseURL string
	mcpEndpoint   string
	openaiClient  *openai.Client
}

// NewRAGService creates a new RAG service instance
func NewRAGService(db DB, openAIKey, openAIBaseURL, mcpEndpoint string) *RAGService {
	config := openai.DefaultConfig(openAIKey)
	if openAIBaseURL != "" {
		config.BaseURL = openAIBaseURL
	}

	return &RAGService{
		db:            db,
		openAIKey:     openAIKey,
		openAIBaseURL: openAIBaseURL,
		mcpEndpoint:   mcpEndpoint,
		openaiClient:  openai.NewClientWithConfig(config),
	}
}

// ProcessDocument processes a document and stores it in the database
func (s *RAGService) ProcessDocument(ctx context.Context, req *models.ProcessDocumentRequest) error {
	// Clean the content
	cleanedContent := s.cleanContent(req.Content)
	if cleanedContent == "" {
		return fmt.Errorf("content is empty after cleaning")
	}

	// Generate embedding for the document
	embedding, err := s.generateEmbedding(ctx, cleanedContent)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store document in database
	var documentID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO documents (url, title, content, embedding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (url) DO UPDATE SET
		  title = EXCLUDED.title,
		  content = EXCLUDED.content,
		  embedding = EXCLUDED.embedding,
		  updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`, req.URL, req.Title, cleanedContent, embedding).Scan(&documentID)
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	// Process chunks
	err = s.chunkDocument(ctx, documentID, cleanedContent)
	if err != nil {
		log.Printf("Warning: failed to chunk document: %v", err)
	}

	// Extract entities and relationships
	err = s.ExtractEntitiesAndRelations(ctx, documentID, cleanedContent)
	if err != nil {
		log.Printf("Warning: failed to extract entities and relations: %v", err)
	}

	return nil
}

// Query searches for relevant content based on the query
func (s *RAGService) Query(ctx context.Context, query string) (*models.QueryResponse, error) {
	// Generate embedding for the query
	queryEmbedding, err := s.generateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Extract keywords from query for text matching
	queryKeywords := extractKeywords(query)

	// Search for relevant chunks with hybrid approach
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			c.content, 
			c.embedding <=> $1 as similarity, 
			d.id as document_id, 
			d.url, 
			d.title,
			-- Add keyword matching score
			CASE 
				WHEN $2 = '' THEN 0
				ELSE (
					SELECT COUNT(*) 
					FROM unnest(string_to_array($2, ' ')) AS keyword 
					WHERE LOWER(c.content) LIKE '%' || LOWER(keyword) || '%'
				)::float / array_length(string_to_array($2, ' '), 1)
			END as keyword_score
		FROM chunks c
		JOIN documents d ON c.document_id = d.id
		WHERE c.embedding <=> $1 < 0.5
		ORDER BY 
			-- Prioritize keyword matches, then vector similarity
			CASE 
				WHEN $2 = '' THEN 0
				ELSE (
					SELECT COUNT(*) 
					FROM unnest(string_to_array($2, ' ')) AS keyword 
					WHERE LOWER(c.content) LIKE '%' || LOWER(keyword) || '%'
				)::float / array_length(string_to_array($2, ' '), 1)
			END DESC,
			c.embedding <=> $1 ASC
		LIMIT 5
	`, queryEmbedding, strings.Join(queryKeywords, " "))
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var content string
		var similarity float64
		var documentID int
		var url string
		var title string
		var keywordScore float64
		if err := rows.Scan(&content, &similarity, &documentID, &url, &title, &keywordScore); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		// Combine vector similarity and keyword matching for final score
		vectorScore := 1.0 - similarity
		finalScore := (vectorScore * 0.3) + (keywordScore * 0.7) // Give more weight to keyword matching

		results = append(results, models.SearchResult{
			Content:    content,
			Score:      finalScore,
			DocumentID: documentID,
			URL:        url,
			Title:      title,
		})
	}

	return &models.QueryResponse{Results: results}, nil
}

// extractKeywords extracts meaningful keywords from the query
func extractKeywords(query string) []string {
	// Convert to lowercase and split into words
	words := strings.Fields(strings.ToLower(query))

	var keywords []string
	for _, word := range words {
		// Remove punctuation and check if it's not a stop word
		word = strings.Trim(word, ".,!?;:()[]{}'\"")
		if len(word) > 2 && !config.IsStopWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// QueueURL adds a URL to the processing queue
func (s *RAGService) QueueURL(ctx context.Context, url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO url_queue (url, status)
		VALUES ($1, 'pending')
	`, url)
	if err != nil {
		return fmt.Errorf("failed to queue URL: %w", err)
	}
	return nil
}

// StartBackgroundWorkers starts the background workers for processing URLs
func (s *RAGService) StartBackgroundWorkers(ctx context.Context, numWorkers int) {
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.processURLQueue(ctx, workerID)
		}(i)
	}
	wg.Wait()
}

// processURLQueue processes URLs from the queue
func (s *RAGService) processURLQueue(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get next URL to process
			var url string
			var queueID int
			err := s.db.QueryRowContext(ctx, `
				UPDATE url_queue
				SET status = 'processing'
				WHERE id = (
					SELECT id
					FROM url_queue
					WHERE status = 'pending'
					ORDER BY created_at ASC
					FOR UPDATE SKIP LOCKED
					LIMIT 1
				)
				RETURNING id, url
			`).Scan(&queueID, &url)

			if err == sql.ErrNoRows {
				// No URLs to process, wait before checking again
				time.Sleep(time.Second)
				continue
			}
			if err != nil {
				log.Printf("Worker %d: Error getting next URL: %v", workerID, err)
				continue
			}

			// Process the URL
			err = s.ProcessURL(ctx, url)
			if err != nil {
				// Update queue status with error
				_, updateErr := s.db.ExecContext(ctx, `
					UPDATE url_queue
					SET status = 'failed',
						error = $1,
						retry_count = retry_count + 1,
						updated_at = CURRENT_TIMESTAMP
					WHERE id = $2
				`, err.Error(), queueID)
				if updateErr != nil {
					log.Printf("Worker %d: Error updating queue status: %v", workerID, updateErr)
				}
				continue
			}

			// Mark URL as processed
			_, err = s.db.ExecContext(ctx, `
				UPDATE url_queue
				SET status = 'completed',
					updated_at = CURRENT_TIMESTAMP
				WHERE id = $1
			`, queueID)
			if err != nil {
				log.Printf("Worker %d: Error marking URL as completed: %v", workerID, err)
			}
		}
	}
}

// processURL processes a URL by fetching content, generating embeddings, and storing in database
func (s *RAGService) ProcessURL(ctx context.Context, url string) error {
	log.Printf("Processing URL: %s", url)

	// Update status to processing
	_, err := s.db.ExecContext(ctx, "UPDATE url_queue SET status = 'processing', updated_at = CURRENT_TIMESTAMP WHERE url = $1", url)
	if err != nil {
		return fmt.Errorf("failed to update status to processing: %w", err)
	}

	// Fetch content from URL
	content, title, err := s.fetchContent(url)
	if err != nil {
		// Update status to failed
		_, updateErr := s.db.ExecContext(ctx,
			"UPDATE url_queue SET status = 'failed', error = $1, updated_at = CURRENT_TIMESTAMP WHERE url = $2",
			err.Error(), url)
		if updateErr != nil {
			log.Printf("Failed to update status to failed: %v", updateErr)
		}
		return fmt.Errorf("failed to fetch content: %w", err)
	}

	// Clean content
	content = s.cleanContent(content)

	// Generate embedding for the full document
	embedding, err := s.generateEmbedding(ctx, content)
	if err != nil {
		// Update status to failed
		_, updateErr := s.db.ExecContext(ctx,
			"UPDATE url_queue SET status = 'failed', error = $1, updated_at = CURRENT_TIMESTAMP WHERE url = $2",
			err.Error(), url)
		if updateErr != nil {
			log.Printf("Failed to update status to failed: %v", updateErr)
		}
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store document in database
	var documentID int
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO documents (url, title, content, embedding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (url) DO UPDATE SET
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`, url, title, content, embedding).Scan(&documentID)

	if err != nil {
		// Update status to failed
		_, updateErr := s.db.ExecContext(ctx,
			"UPDATE url_queue SET status = 'failed', error = $1, updated_at = CURRENT_TIMESTAMP WHERE url = $2",
			err.Error(), url)
		if updateErr != nil {
			log.Printf("Failed to update status to failed: %v", updateErr)
		}
		return fmt.Errorf("failed to store document: %w", err)
	}

	// Chunk the content and store chunks
	err = s.chunkDocument(ctx, documentID, content)
	if err != nil {
		log.Printf("Failed to chunk document: %v", err)
		// Continue processing even if chunking fails
	}

	// Extract entities and relationships (background processing)
	go func() {
		// Create a new context for background processing
		bgCtx := context.Background()
		err := s.ExtractEntitiesAndRelations(bgCtx, documentID, content)
		if err != nil {
			log.Printf("Failed to extract entities and relations for document %d: %v", documentID, err)
		}
	}()

	// Update status to completed
	_, err = s.db.ExecContext(ctx, "UPDATE url_queue SET status = 'completed', updated_at = CURRENT_TIMESTAMP WHERE url = $1", url)
	if err != nil {
		log.Printf("Failed to update status to completed: %v", err)
	}

	log.Printf("Successfully processed URL: %s (Document ID: %d)", url, documentID)
	return nil
}

// Helper functions

func (s *RAGService) cleanContent(content string) string {
	// Remove invalid UTF-8 sequences
	if !utf8.ValidString(content) {
		content = strings.ToValidUTF8(content, "")
	}

	// Remove control characters except newlines and tabs
	var cleaned strings.Builder
	for _, r := range content {
		if r == '\n' || r == '\t' || (r >= 32 && r != 127) {
			cleaned.WriteRune(r)
		}
	}

	// Trim whitespace
	return strings.TrimSpace(cleaned.String())
}

func (s *RAGService) generateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	// For testing, generate a more meaningful vector based on text content
	// In production, this would use OpenAI's API to generate embeddings
	dimensions := 1536
	vector := make([]float32, dimensions)

	// Normalize and clean text for better feature extraction
	text = strings.ToLower(strings.TrimSpace(text))
	words := strings.Fields(text)

	// Create a more sophisticated hash that considers word frequency and position
	wordHash := make(map[string]int)
	for i, word := range words {
		// Give more weight to words at the beginning
		weight := len(words) - i
		wordHash[word] += weight
	}

	// Create a combined hash from word frequencies
	combinedHash := 0
	for word, freq := range wordHash {
		for _, char := range word {
			combinedHash = (combinedHash*31 + int(char)) % 1000000
		}
		combinedHash = (combinedHash * freq) % 1000000
	}

	// Use the combined hash to generate more distinctive vectors
	seed := int64(combinedHash)
	for i := range vector {
		// More sophisticated pseudo-random generation
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		// Add position-based variation to make vectors more unique
		positionFactor := float32(i) / float32(dimensions)
		vector[i] = float32(seed%1000)/1000.0 + positionFactor*0.1
	}

	return pgvector.NewVector(vector), nil
}

func (s *RAGService) chunkDocument(ctx context.Context, documentID int, content string) error {
	// Simple chunking by sentences
	sentences := strings.Split(content, ".")
	chunks := make([]ChunkInfo, 0)
	currentChunk := ""
	currentStart := 0
	chunkIndex := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// Find the position of this sentence in the original content
		sentenceStart := strings.Index(content[currentStart:], sentence)
		if sentenceStart == -1 {
			sentenceStart = 0
		}
		sentenceStart += currentStart

		if len(currentChunk)+len(sentence) < 1000 {
			if currentChunk != "" {
				currentChunk += ". "
			}
			currentChunk += sentence
		} else {
			if currentChunk != "" {
				// Find the end position of the current chunk
				chunkEnd := strings.LastIndex(content[:sentenceStart], currentChunk)
				if chunkEnd == -1 {
					chunkEnd = sentenceStart
				} else {
					chunkEnd += len(currentChunk)
				}

				chunks = append(chunks, ChunkInfo{
					Content:       currentChunk,
					ChunkIndex:    chunkIndex,
					StartPosition: currentStart,
					EndPosition:   chunkEnd,
				})
				chunkIndex++
			}
			currentChunk = sentence
			currentStart = sentenceStart
		}
	}

	if currentChunk != "" {
		// Find the end position of the last chunk
		chunkEnd := strings.LastIndex(content[currentStart:], currentChunk)
		if chunkEnd == -1 {
			chunkEnd = len(content)
		} else {
			chunkEnd += currentStart + len(currentChunk)
		}

		chunks = append(chunks, ChunkInfo{
			Content:       currentChunk,
			ChunkIndex:    chunkIndex,
			StartPosition: currentStart,
			EndPosition:   chunkEnd,
		})
	}

	// Store chunks in database with embeddings
	for _, chunk := range chunks {
		// Generate embedding for the chunk
		embedding, err := s.generateEmbedding(ctx, chunk.Content)
		if err != nil {
			log.Printf("Warning: failed to generate embedding for chunk %d: %v", chunk.ChunkIndex, err)
			// Continue without embedding
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO chunks (document_id, content, chunk_index, start_position, end_position)
				VALUES ($1, $2, $3, $4, $5)
			`, documentID, chunk.Content, chunk.ChunkIndex, chunk.StartPosition, chunk.EndPosition)
		} else {
			// Store chunk with embedding
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO chunks (document_id, content, embedding, chunk_index, start_position, end_position)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, documentID, chunk.Content, embedding, chunk.ChunkIndex, chunk.StartPosition, chunk.EndPosition)
		}

		if err != nil {
			log.Printf("Warning: failed to store chunk: %v", err)
		}
	}

	return nil
}

// GetURLQueue retrieves all URLs from the queue
func (s *RAGService) GetURLQueue(ctx context.Context) ([]models.URLQueueItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT q.id, q.url, q.status, q.created_at, q.updated_at, q.retry_count, d.id as document_id
		FROM url_queue q
		LEFT JOIN documents d ON q.url = d.url
		WHERE q.status != 'deleted'
		ORDER BY q.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query url_queue: %w", err)
	}
	defer rows.Close()

	var queue []models.URLQueueItem
	for rows.Next() {
		var item models.URLQueueItem
		var documentID sql.NullInt32
		if err := rows.Scan(&item.ID, &item.URL, &item.Status, &item.CreatedAt, &item.UpdatedAt, &item.RetryCount, &documentID); err != nil {
			return nil, fmt.Errorf("failed to scan url_queue row: %w", err)
		}
		if documentID.Valid {
			item.DocumentID = int(documentID.Int32)
		}
		queue = append(queue, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating url_queue rows: %w", err)
	}

	return queue, nil
}

// DeleteURL marks a URL as deleted in the queue
func (s *RAGService) DeleteURL(ctx context.Context, url string) error {
	log.Printf("DeleteURL called with URL: %s", url)

	// First check if the URL exists
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM url_queue WHERE url = $1`, url).Scan(&count)
	if err != nil {
		log.Printf("Error checking URL existence: %v", err)
		return fmt.Errorf("failed to check URL existence: %w", err)
	}

	log.Printf("Found %d records with URL: %s", count, url)

	if count == 0 {
		return fmt.Errorf("URL not found in queue: %s", url)
	}

	// Update status to deleted instead of deleting the record
	result, err := s.db.ExecContext(ctx, `UPDATE url_queue SET status = 'deleted', updated_at = CURRENT_TIMESTAMP WHERE url = $1`, url)
	if err != nil {
		log.Printf("Error updating URL status: %v", err)
		return fmt.Errorf("failed to update url_queue status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Updated %d rows for URL: %s", rowsAffected, url)

	if rowsAffected == 0 {
		return fmt.Errorf("no rows were updated for URL: %s", url)
	}

	// Delete related knowledge graph edges first
	edgesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_edges 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge edges: %v", err)
		return fmt.Errorf("failed to delete knowledge edges: %w", err)
	}
	edgesAffected, _ := edgesResult.RowsAffected()
	log.Printf("Deleted %d knowledge edges for URL: %s", edgesAffected, url)

	// Delete related knowledge graph nodes
	nodesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_nodes 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge nodes: %v", err)
		return fmt.Errorf("failed to delete knowledge nodes: %w", err)
	}
	nodesAffected, _ := nodesResult.RowsAffected()
	log.Printf("Deleted %d knowledge nodes for URL: %s", nodesAffected, url)

	// Delete related chunks
	chunksResult, err := s.db.ExecContext(ctx, `
		DELETE FROM chunks 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting chunks: %v", err)
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	chunksAffected, _ := chunksResult.RowsAffected()
	log.Printf("Deleted %d chunks for URL: %s", chunksAffected, url)

	// Finally delete documents
	docsResult, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE url = $1`, url)
	if err != nil {
		log.Printf("Error deleting documents: %v", err)
		return fmt.Errorf("failed to delete documents: %w", err)
	}
	docsAffected, _ := docsResult.RowsAffected()
	log.Printf("Deleted %d documents for URL: %s", docsAffected, url)

	log.Printf("Successfully completed DeleteURL for: %s", url)
	return nil
}

// ReindexURL reprocesses a URL
func (s *RAGService) ReindexURL(ctx context.Context, url string) error {
	// First delete existing data
	err := s.DeleteURL(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to delete existing data: %w", err)
	}

	// Then reprocess the URL
	return s.ProcessURL(ctx, url)
}

// DeleteURLByID marks a URL as deleted in the queue by ID
func (s *RAGService) DeleteURLByID(ctx context.Context, id string) error {
	log.Printf("DeleteURLByID called with ID: %s", id)

	// First check if the ID exists and get the URL
	var url string
	err := s.db.QueryRowContext(ctx, `SELECT url FROM url_queue WHERE id = $1`, id).Scan(&url)
	if err != nil {
		log.Printf("Error getting URL for ID %s: %v", id, err)
		return fmt.Errorf("failed to get URL for ID: %w", err)
	}

	log.Printf("Found URL %s for ID %s", url, id)

	// Update status to deleted instead of deleting the record
	result, err := s.db.ExecContext(ctx, `UPDATE url_queue SET status = 'deleted', updated_at = CURRENT_TIMESTAMP WHERE id = $1`, id)
	if err != nil {
		log.Printf("Error updating URL status: %v", err)
		return fmt.Errorf("failed to update url_queue status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Updated %d rows for ID: %s", rowsAffected, id)

	if rowsAffected == 0 {
		return fmt.Errorf("no rows were updated for ID: %s", id)
	}

	// Delete related knowledge graph edges first
	edgesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_edges 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge edges: %v", err)
		return fmt.Errorf("failed to delete knowledge edges: %w", err)
	}
	edgesAffected, _ := edgesResult.RowsAffected()
	log.Printf("Deleted %d knowledge edges for URL: %s", edgesAffected, url)

	// Delete related knowledge graph nodes
	nodesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_nodes 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge nodes: %v", err)
		return fmt.Errorf("failed to delete knowledge nodes: %w", err)
	}
	nodesAffected, _ := nodesResult.RowsAffected()
	log.Printf("Deleted %d knowledge nodes for URL: %s", nodesAffected, url)

	// Delete related chunks
	chunksResult, err := s.db.ExecContext(ctx, `
		DELETE FROM chunks 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting chunks: %v", err)
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	chunksAffected, _ := chunksResult.RowsAffected()
	log.Printf("Deleted %d chunks for URL: %s", chunksAffected, url)

	// Finally delete documents
	docsResult, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE url = $1`, url)
	if err != nil {
		log.Printf("Error deleting documents: %v", err)
		return fmt.Errorf("failed to delete documents: %w", err)
	}
	docsAffected, _ := docsResult.RowsAffected()
	log.Printf("Deleted %d documents for URL: %s", docsAffected, url)

	log.Printf("Successfully completed DeleteURLByID for ID: %s", id)
	return nil
}

// ReindexURLByID reprocesses a URL by ID
func (s *RAGService) ReindexURLByID(ctx context.Context, id string) error {
	log.Printf("ReindexURLByID called with ID: %s", id)

	// First get the URL
	var url string
	err := s.db.QueryRowContext(ctx, `SELECT url FROM url_queue WHERE id = $1`, id).Scan(&url)
	if err != nil {
		log.Printf("Error getting URL for ID %s: %v", id, err)
		return fmt.Errorf("failed to get URL for reprocessing: %w", err)
	}

	log.Printf("Found URL %s for ID %s", url, id)

	// Delete related knowledge graph edges first
	edgesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_edges 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge edges: %v", err)
		return fmt.Errorf("failed to delete knowledge edges: %w", err)
	}

	edgesAffected, _ := edgesResult.RowsAffected()
	log.Printf("Deleted %d knowledge edges for URL: %s", edgesAffected, url)

	// Delete related knowledge graph nodes
	nodesResult, err := s.db.ExecContext(ctx, `
		DELETE FROM knowledge_nodes 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting knowledge nodes: %v", err)
		return fmt.Errorf("failed to delete knowledge nodes: %w", err)
	}

	nodesAffected, _ := nodesResult.RowsAffected()
	log.Printf("Deleted %d knowledge nodes for URL: %s", nodesAffected, url)

	// Delete related chunks
	chunksResult, err := s.db.ExecContext(ctx, `
		DELETE FROM chunks 
		WHERE document_id IN (SELECT id FROM documents WHERE url = $1)
	`, url)
	if err != nil {
		log.Printf("Error deleting chunks: %v", err)
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	chunksAffected, _ := chunksResult.RowsAffected()
	log.Printf("Deleted %d chunks for URL: %s", chunksAffected, url)

	// Finally delete documents
	docsResult, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE url = $1`, url)
	if err != nil {
		log.Printf("Error deleting documents: %v", err)
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	docsAffected, _ := docsResult.RowsAffected()
	log.Printf("Deleted %d documents for URL: %s", docsAffected, url)

	// Reset the queue status to pending for background worker processing
	result, err := s.db.ExecContext(ctx, `
		UPDATE url_queue 
		SET status = 'pending', 
		    error = NULL, 
		    retry_count = 0, 
		    updated_at = CURRENT_TIMESTAMP 
		WHERE id = $1
	`, id)
	if err != nil {
		log.Printf("Error resetting queue status: %v", err)
		return fmt.Errorf("failed to reset queue status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("Reset %d rows for ID: %s, URL will be processed by background worker", rowsAffected, id)
	return nil
}

// GetDocumentByID retrieves a document by ID
func (s *RAGService) GetDocumentByID(ctx context.Context, id int) (*models.Document, error) {
	var doc models.Document
	err := s.db.QueryRowContext(ctx, `
		SELECT id, url, title, content, created_at, updated_at
		FROM documents 
		WHERE id = $1
	`, id).Scan(&doc.ID, &doc.URL, &doc.Title, &doc.Content, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// GetDocumentChunks retrieves chunks for a specific document
func (s *RAGService) GetDocumentChunks(ctx context.Context, documentID int) ([]models.Chunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content, chunk_index, start_position, end_position, created_at
		FROM chunks 
		WHERE document_id = $1
		ORDER BY chunk_index
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []models.Chunk
	for rows.Next() {
		var chunk models.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.Content, &chunk.ChunkIndex, &chunk.StartPosition, &chunk.EndPosition, &chunk.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk row: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chunks rows: %w", err)
	}

	return chunks, nil
}

// GetDocumentVectors retrieves vectors for a specific document
func (s *RAGService) GetDocumentVectors(ctx context.Context, documentID int) ([]models.VectorData, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content, embedding, chunk_index, start_position, end_position
		FROM chunks 
		WHERE document_id = $1
		ORDER BY chunk_index
	`, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query vectors: %w", err)
	}
	defer rows.Close()

	var vectors []models.VectorData
	for rows.Next() {
		var vector models.VectorData
		var embedding pgvector.Vector
		if err := rows.Scan(&vector.ID, &vector.Content, &embedding, &vector.ChunkIndex, &vector.StartPosition, &vector.EndPosition); err != nil {
			return nil, fmt.Errorf("failed to scan vector row: %w", err)
		}
		vector.Embedding = embedding.Slice()
		vectors = append(vectors, vector)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vectors rows: %w", err)
	}

	return vectors, nil
}

// GetDocumentKnowledgeGraph retrieves knowledge graph for a specific document
func (s *RAGService) GetDocumentKnowledgeGraph(ctx context.Context, documentID int) (*models.KnowledgeGraph, error) {
	// Get knowledge graph nodes for this document
	nodes, edges, err := s.GetKnowledgeGraphByDocument(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge graph for document: %w", err)
	}

	return &models.KnowledgeGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetDocumentIDByURL retrieves document ID by URL
func (s *RAGService) GetDocumentIDByURL(ctx context.Context, url string) (int, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `SELECT id FROM documents WHERE url = $1`, url).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get document ID: %w", err)
	}
	return id, nil
}

// ExtractEntitiesAndRelations extracts entities and relationships from document content
func (s *RAGService) ExtractEntitiesAndRelations(ctx context.Context, documentID int, content string) error {
	log.Printf("Extracting entities and relations for document ID: %d", documentID)

	// Extract entities from content
	entities := s.extractEntities(content)

	// Store entities in database
	entityMap := make(map[string]int) // name -> id
	for _, entity := range entities {
		// Check if entity already exists
		var existingID int
		err := s.db.QueryRowContext(ctx, `
			SELECT id FROM knowledge_nodes WHERE name = $1 AND type = $2
		`, entity.Name, entity.Type).Scan(&existingID)

		if err == nil {
			// Entity already exists, use existing ID
			entityMap[entity.Name] = existingID
			log.Printf("Entity already exists: %s (ID: %d, Type: %s)", entity.Name, existingID, entity.Type)
			continue
		} else if err != sql.ErrNoRows {
			log.Printf("Error checking existing entity %s: %v", entity.Name, err)
			continue
		}

		embedding, err := s.generateEmbedding(ctx, entity.Name)
		if err != nil {
			log.Printf("Failed to generate embedding for entity %s: %v", entity.Name, err)
			continue
		}

		// Convert properties map to JSON string
		propertiesJSON, err := json.Marshal(entity.Properties)
		if err != nil {
			log.Printf("Failed to marshal properties for entity %s: %v", entity.Name, err)
			continue
		}

		var id int
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO knowledge_nodes (name, type, properties, embedding, document_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (name, type) DO UPDATE SET
				properties = EXCLUDED.properties,
				embedding = EXCLUDED.embedding,
				document_id = EXCLUDED.document_id
			RETURNING id
		`, entity.Name, entity.Type, propertiesJSON, embedding, documentID).Scan(&id)

		if err != nil {
			log.Printf("Failed to insert entity %s: %v", entity.Name, err)
			continue
		}

		entityMap[entity.Name] = id
		log.Printf("Stored entity: %s (ID: %d, Type: %s)", entity.Name, id, entity.Type)
	}

	// Extract relationships
	relationships := s.extractRelationships(content, entityMap)

	// Store relationships in database
	for _, rel := range relationships {
		// Convert properties map to JSON string
		propertiesJSON, err := json.Marshal(rel.Properties)
		if err != nil {
			log.Printf("Failed to marshal properties for relationship %d -> %d: %v", rel.SourceID, rel.TargetID, err)
			continue
		}

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO knowledge_edges (source_id, target_id, relationship_type, properties, document_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (source_id, target_id, relationship_type) DO UPDATE SET
				properties = EXCLUDED.properties,
				document_id = EXCLUDED.document_id
		`, rel.SourceID, rel.TargetID, rel.RelationshipType, propertiesJSON, documentID)

		if err != nil {
			log.Printf("Failed to insert relationship %d -> %d (%s): %v",
				rel.SourceID, rel.TargetID, rel.RelationshipType, err)
			continue
		}

		log.Printf("Stored relationship: %d -> %d (%s)",
			rel.SourceID, rel.TargetID, rel.RelationshipType)
	}

	log.Printf("Completed entity and relation extraction for document ID: %d", documentID)
	return nil
}

// extractEntities extracts entities from text content
func (s *RAGService) extractEntities(content string) []models.Entity {
	var entities []models.Entity
	seenEntities := make(map[string]bool) // 避免重复实体

	// Improved entity extraction with better filtering

	// Extract person names (capitalized words that might be names)
	personPattern := regexp.MustCompile(`\b[A-Z][a-z]+ [A-Z][a-z]+\b`)
	persons := personPattern.FindAllString(content, -1)
	for _, person := range persons {
		if !isCommonName(person) && !seenEntities[person] && len(person) > 3 {
			entities = append(entities, models.Entity{
				Name: person,
				Type: "person",
				Properties: map[string]any{
					"source": "pattern_matching",
				},
			})
			seenEntities[person] = true
		}
	}

	// Extract organizations (words ending with common org suffixes)
	orgPattern := regexp.MustCompile(`\b[A-Z][a-zA-Z\s&]+(?:Inc|Corp|Company|University|Institute|Foundation|Organization|School|College|Hospital|Museum|Gallery|Library)\b`)
	organizations := orgPattern.FindAllString(content, -1)
	for _, org := range organizations {
		org = strings.TrimSpace(org)
		if !seenEntities[org] && len(org) > 5 {
			entities = append(entities, models.Entity{
				Name: org,
				Type: "organization",
				Properties: map[string]any{
					"source": "pattern_matching",
				},
			})
			seenEntities[org] = true
		}
	}

	// Extract locations (words that might be places)
	locationPattern := regexp.MustCompile(`\b[A-Z][a-z]+(?: City| State| Country| University| Museum| Gallery| Park| Street| Avenue| Road| Airport| Station)\b`)
	locations := locationPattern.FindAllString(content, -1)
	for _, location := range locations {
		location = strings.TrimSpace(location)
		if !seenEntities[location] && len(location) > 4 {
			entities = append(entities, models.Entity{
				Name: location,
				Type: "location",
				Properties: map[string]any{
					"source": "pattern_matching",
				},
			})
			seenEntities[location] = true
		}
	}

	// Extract important concepts (quoted phrases and capitalized terms)
	// Look for quoted text first
	quotedPattern := regexp.MustCompile(`"([^"]{3,50})"`)
	quotedMatches := quotedPattern.FindAllStringSubmatch(content, -1)
	for _, match := range quotedMatches {
		concept := strings.TrimSpace(match[1])

		// More strict filtering for quoted text
		// Skip if it's too long (likely a full sentence)
		if len(concept) > 30 {
			continue
		}

		// Skip if it contains sentence-ending punctuation
		if strings.ContainsAny(concept, ".!?") {
			continue
		}

		// Skip if it starts with common sentence starters
		lowerConcept := strings.ToLower(concept)
		if strings.HasPrefix(lowerConcept, "i ") ||
			strings.HasPrefix(lowerConcept, "we ") ||
			strings.HasPrefix(lowerConcept, "you ") ||
			strings.HasPrefix(lowerConcept, "he ") ||
			strings.HasPrefix(lowerConcept, "she ") ||
			strings.HasPrefix(lowerConcept, "they ") ||
			strings.HasPrefix(lowerConcept, "it ") ||
			strings.HasPrefix(lowerConcept, "this ") ||
			strings.HasPrefix(lowerConcept, "that ") ||
			strings.HasPrefix(lowerConcept, "there ") ||
			strings.HasPrefix(lowerConcept, "here ") {
			continue
		}

		// Skip if it's just a common phrase or generic statement
		commonPhrases := []string{
			"better you than me", "i have to be really careful", "whenever i'd noticed",
			"i think", "i believe", "i know", "i feel", "i want", "i need",
			"we should", "we can", "we will", "we have", "we are",
			"you can", "you should", "you will", "you have", "you are",
			"it is", "it was", "it will", "it can", "it should",
			"this is", "this was", "this will", "this can",
			"that is", "that was", "that will", "that can",
		}

		skipPhrase := false
		for _, phrase := range commonPhrases {
			if strings.Contains(lowerConcept, phrase) {
				skipPhrase = true
				break
			}
		}
		if skipPhrase {
			continue
		}

		// Apply standard filtering
		if !isCommonWord(concept) && !seenEntities[concept] && len(concept) > 2 &&
			!config.IsStopWord(concept) && !config.IsGenericTerm(concept) {
			entities = append(entities, models.Entity{
				Name: concept,
				Type: "concept",
				Properties: map[string]any{
					"source": "quoted_text",
				},
			})
			seenEntities[concept] = true
		}
	}

	// Extract capitalized multi-word concepts (but be more selective)
	conceptPattern := regexp.MustCompile(`\b[A-Z][a-z]+(?: [A-Z][a-z]+){1,3}\b`)
	concepts := conceptPattern.FindAllString(content, -1)
	for _, concept := range concepts {
		concept = strings.TrimSpace(concept)
		// More strict filtering for concepts
		if !isCommonWord(concept) && !isCommonName(concept) && !isCommonPlace(concept) &&
			!seenEntities[concept] && len(concept) > 4 &&
			!config.IsStopWord(concept) && !config.IsGenericTerm(concept) {
			entities = append(entities, models.Entity{
				Name: concept,
				Type: "concept",
				Properties: map[string]any{
					"source": "pattern_matching",
				},
			})
			seenEntities[concept] = true
		}
	}

	// Extract important single words (only if they're significant)
	singleWordPattern := regexp.MustCompile(`\b[A-Z][a-z]{3,}\b`)
	singleWords := singleWordPattern.FindAllString(content, -1)
	for _, word := range singleWords {
		if !isCommonWord(word) && !isCommonName(word) && !isCommonPlace(word) &&
			!seenEntities[word] && !config.IsStopWord(word) && !config.IsGenericTerm(word) &&
			config.IsSignificantWord(word) {
			entities = append(entities, models.Entity{
				Name: word,
				Type: "concept",
				Properties: map[string]any{
					"source": "significant_word",
				},
			})
			seenEntities[word] = true
		}
	}

	return entities
}

// extractRelationships extracts relationships between entities
func (s *RAGService) extractRelationships(content string, entityMap map[string]int) []models.Relationship {
	var relationships []models.Relationship

	// Simple relationship extraction based on proximity and patterns
	// In production, this would use more sophisticated NLP techniques

	// Extract "X is Y" relationships
	isPattern := regexp.MustCompile(`(\b[A-Z][a-z]+ [A-Z][a-z]+\b)\s+(?:is|was|are|were)\s+([^.!?]+)`)
	matches := isPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		entity1 := match[1]
		description := strings.TrimSpace(match[2])

		if id1, exists := entityMap[entity1]; exists {
			// Create a concept entity for the description
			conceptName := extractMainConcept(description)
			if conceptName != "" {
				// Add the concept to entityMap if not exists
				if id2, exists := entityMap[conceptName]; exists {
					relationships = append(relationships, models.Relationship{
						SourceID:         id1,
						TargetID:         id2,
						RelationshipType: "is_a",
						Properties: map[string]any{
							"description": description,
							"source":      "pattern_matching",
						},
					})
				}
			}
		}
	}

	// Extract "X works at Y" relationships
	worksAtPattern := regexp.MustCompile(`(\b[A-Z][a-z]+ [A-Z][a-z]+\b)\s+(?:works at|worked at|studied at|attended)\s+([^.!?]+)`)
	matches = worksAtPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		person := match[1]
		organization := strings.TrimSpace(match[2])

		if id1, exists := entityMap[person]; exists {
			if id2, exists := entityMap[organization]; exists {
				relationships = append(relationships, models.Relationship{
					SourceID:         id1,
					TargetID:         id2,
					RelationshipType: "works_at",
					Properties: map[string]any{
						"source": "pattern_matching",
					},
				})
			}
		}
	}

	// Extract "X in Y" location relationships
	inPattern := regexp.MustCompile(`(\b[A-Z][a-z]+ [A-Z][a-z]+\b)\s+in\s+([^.!?]+)`)
	matches = inPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		entity := match[1]
		location := strings.TrimSpace(match[2])

		if id1, exists := entityMap[entity]; exists {
			if id2, exists := entityMap[location]; exists {
				relationships = append(relationships, models.Relationship{
					SourceID:         id1,
					TargetID:         id2,
					RelationshipType: "located_in",
					Properties: map[string]any{
						"source": "pattern_matching",
					},
				})
			}
		}
	}

	return relationships
}

// Helper functions
func isCommonWord(word string) bool {
	// Use config package instead
	return config.IsStopWord(word)
}

func isCommonName(word string) bool {
	// Use config package instead
	return config.IsGenericTerm(word)
}

func isCommonPlace(word string) bool {
	// Use config package instead
	return config.IsGenericTerm(word)
}

func extractMainConcept(description string) string {
	// Simple concept extraction - take the first significant noun phrase
	words := strings.Fields(description)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:()[]{}'\"")
		if len(word) > 3 && !isCommonWord(word) && word[0] >= 'A' && word[0] <= 'Z' {
			return word
		}
	}
	return ""
}

// GetKnowledgeGraph returns knowledge graph data, optionally filtered by a query string
func (s *RAGService) GetKnowledgeGraph(ctx context.Context, query string) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error) {
	log.Printf("GetKnowledgeGraph: Received query: '%s'", query)
	var args []interface{}
	nodeQuery := `
		SELECT 
			kn.id, kn.name, kn.type, kn.properties, kn.document_id, d.url, d.title 
		FROM knowledge_nodes kn
		LEFT JOIN documents d ON kn.document_id = d.id
	`
	if query != "" {
		nodeQuery += " WHERE kn.name ILIKE $1"
		args = append(args, "%"+query+"%")
	}
	nodeQuery += " ORDER BY kn.id"
	log.Printf("GetKnowledgeGraph: Executing node query: %s with args: %v", nodeQuery, args)

	// Get all nodes
	rows, err := s.db.QueryContext(ctx, nodeQuery, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query knowledge nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.KnowledgeNodeResponse
	nodeIDs := make(map[int]struct{})
	for rows.Next() {
		var node models.KnowledgeNodeResponse
		var propertiesJSON []byte
		var docURL, docTitle sql.NullString
		err := rows.Scan(&node.ID, &node.Name, &node.Type, &propertiesJSON, &node.DocumentID, &docURL, &docTitle)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan knowledge node: %w", err)
		}

		if docURL.Valid {
			node.URL = &docURL.String
		}
		if docTitle.Valid {
			node.Title = &docTitle.String
		}

		if propertiesJSON != nil {
			if err := json.Unmarshal(propertiesJSON, &node.Properties); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal node properties: %w", err)
			}
		}
		nodes = append(nodes, node)
		nodeIDs[node.ID] = struct{}{}
	}
	log.Printf("GetKnowledgeGraph: Found %d nodes from query.", len(nodes))
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating over node rows: %w", err)
	}

	// If no nodes are found for a specific query, return empty results immediately
	if query != "" && len(nodes) == 0 {
		return []models.KnowledgeNodeResponse{}, []models.KnowledgeEdgeResponse{}, nil
	}

	// Regardless of query, fetch all edges and filter in memory if needed.
	// This is simpler and more robust than dynamic SQL, though less performant on huge graphs.
	edgeQuery := `SELECT id, source_id, target_id, relationship_type, properties, document_id FROM knowledge_edges ORDER BY id`
	log.Printf("GetKnowledgeGraph: Executing universal edge query: %s", edgeQuery)

	edgeRows, err := s.db.QueryContext(ctx, edgeQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query all edges: %w", err)
	}
	defer edgeRows.Close()

	var allEdges []models.KnowledgeEdgeResponse
	for edgeRows.Next() {
		var edge models.KnowledgeEdgeResponse
		var propertiesJSON []byte
		err := edgeRows.Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.RelationshipType, &propertiesJSON, &edge.DocumentID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan knowledge edge: %w", err)
		}
		if propertiesJSON != nil {
			if err := json.Unmarshal(propertiesJSON, &edge.Properties); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal edge properties: %w", err)
			}
		}
		allEdges = append(allEdges, edge)
	}
	log.Printf("GetKnowledgeGraph: Found %d total edges to filter from.", len(allEdges))

	var finalEdges []models.KnowledgeEdgeResponse
	if query != "" {
		// A query was provided, so filter the edges.
		// Only keep edges where BOTH source and target are in our initial node list.
		for _, edge := range allEdges {
			_, sourceInNodes := nodeIDs[edge.SourceID]
			_, targetInNodes := nodeIDs[edge.TargetID]
			if sourceInNodes && targetInNodes {
				finalEdges = append(finalEdges, edge)
			}
		}
	} else {
		// No query, so we want all edges.
		finalEdges = allEdges
	}

	log.Printf("GetKnowledgeGraph: Returning %d nodes and %d filtered edges.", len(nodes), len(finalEdges))

	if nodes == nil {
		nodes = make([]models.KnowledgeNodeResponse, 0)
	}
	if finalEdges == nil {
		finalEdges = make([]models.KnowledgeEdgeResponse, 0)
	}

	return nodes, finalEdges, nil
}

// GetKnowledgeGraphByDocument returns knowledge graph data for a specific document
func (s *RAGService) GetKnowledgeGraphByDocument(ctx context.Context, documentID int) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error) {
	// Get nodes for the document
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			kn.id, kn.name, kn.type, kn.properties, kn.document_id, d.url, d.title
		FROM knowledge_nodes kn
		LEFT JOIN documents d ON kn.document_id = d.id
		WHERE kn.document_id = $1
		ORDER BY kn.id
	`, documentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query knowledge nodes: %w", err)
	}
	defer rows.Close()

	var nodes []models.KnowledgeNodeResponse
	for rows.Next() {
		var node models.KnowledgeNodeResponse
		var propertiesJSON []byte
		var docURL, docTitle sql.NullString
		err := rows.Scan(&node.ID, &node.Name, &node.Type, &propertiesJSON, &node.DocumentID, &docURL, &docTitle)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan knowledge node: %w", err)
		}

		if docURL.Valid {
			node.URL = &docURL.String
		}
		if docTitle.Valid {
			node.Title = &docTitle.String
		}

		if propertiesJSON != nil {
			if err := json.Unmarshal(propertiesJSON, &node.Properties); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal node properties: %w", err)
			}
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating over node rows: %w", err)
	}

	// Get edges for the document
	edgeRows, err := s.db.QueryContext(ctx, `
		SELECT id, source_id, target_id, relationship_type, properties, document_id
		FROM knowledge_edges
		WHERE document_id = $1
		ORDER BY id
	`, documentID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query knowledge edges: %w", err)
	}
	defer edgeRows.Close()

	var edges []models.KnowledgeEdgeResponse
	for edgeRows.Next() {
		var edge models.KnowledgeEdgeResponse
		var propertiesJSON []byte
		err := edgeRows.Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.RelationshipType, &propertiesJSON, &edge.DocumentID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan knowledge edge: %w", err)
		}

		if propertiesJSON != nil {
			if err := json.Unmarshal(propertiesJSON, &edge.Properties); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal edge properties: %w", err)
			}
		}

		edges = append(edges, edge)
	}

	if err := edgeRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating over edge rows: %w", err)
	}

	if nodes == nil {
		nodes = make([]models.KnowledgeNodeResponse, 0)
	}
	if edges == nil {
		edges = make([]models.KnowledgeEdgeResponse, 0)
	}

	return nodes, edges, nil
}

// fetchContent fetches content from a URL
func (s *RAGService) fetchContent(url string) (string, string, error) {
	// Fetch the URL
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Extract title
	title := doc.Find("title").Text()
	if title == "" {
		title = url
	}

	// Extract main content
	content := ""
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		// Remove script and style elements
		s.Find("script, style").Remove()
		// Get text content
		content = s.Text()
	})

	// Clean up content
	content = s.cleanContent(content)
	if content == "" {
		return "", "", fmt.Errorf("no content found at URL")
	}

	return content, title, nil
}

// LogMCPRequest logs an MCP request to the database
func (s *RAGService) LogMCPRequest(ctx context.Context, logEntry *models.MCPLog) error {
	log.Printf("RAGService LogMCPRequest: Attempting to insert log for RequestID: %s", logEntry.RequestID)

	// The pq driver does not correctly handle nil []byte slices for JSONB columns.
	// We need to explicitly provide a valid JSON 'null' if the slice is nil or empty.
	params := logEntry.Params
	if len(params) == 0 {
		params = []byte("null")
	}
	response := logEntry.Response
	if len(response) == 0 {
		response = []byte("null")
	}
	errorVal := logEntry.Error
	if len(errorVal) == 0 {
		errorVal = []byte("null")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_logs (request_id, method, params, response, error)
		VALUES ($1, $2, $3, $4, $5)
	`, logEntry.RequestID, logEntry.Method, params, response, errorVal)
	if err != nil {
		log.Printf("RAGService LogMCPRequest: FAILED to log MCP request. DB error: %v", err)
		return fmt.Errorf("failed to log MCP request: %w", err)
	}
	log.Printf("RAGService LogMCPRequest: Successfully executed INSERT for RequestID: %s. Error is nil.", logEntry.RequestID)
	return nil
}

// GetMCPLogs retrieves MCP logs from the database
func (s *RAGService) GetMCPLogs(ctx context.Context) ([]models.MCPLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, request_id, method, params, response, error, created_at
		FROM mcp_logs
		ORDER BY created_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP logs: %w", err)
	}
	defer rows.Close()

	var logs []models.MCPLog
	for rows.Next() {
		var logEntry models.MCPLog
		var params, response, errorBytes []byte

		if err := rows.Scan(&logEntry.ID, &logEntry.RequestID, &logEntry.Method, &params, &response, &errorBytes, &logEntry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan MCP log: %w", err)
		}

		// Handle nil/null values properly for JSON fields
		if params == nil {
			logEntry.Params = json.RawMessage("null")
		} else {
			logEntry.Params = params
		}

		if response == nil {
			logEntry.Response = json.RawMessage("null")
		} else {
			logEntry.Response = response
		}

		if errorBytes == nil {
			logEntry.Error = json.RawMessage("null")
		} else {
			logEntry.Error = errorBytes
		}

		logs = append(logs, logEntry)
	}

	return logs, nil
}
