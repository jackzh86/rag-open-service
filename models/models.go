package models

import (
	"encoding/json"
	"time"
)

// Document represents a source document in the system
type Document struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Embedding []float32 `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Chunk represents a text chunk from a document
type Chunk struct {
	ID            int       `json:"id"`
	DocumentID    int       `json:"document_id"`
	Content       string    `json:"content"`
	Embedding     []float32 `json:"-"`
	ChunkIndex    int       `json:"chunk_index"`
	StartPosition int       `json:"start_position"`
	EndPosition   int       `json:"end_position"`
	URL           string    `json:"url"`
	Score         float32   `json:"score"`
	CreatedAt     time.Time `json:"created_at"`
}

// KnowledgeNode represents a node in the knowledge graph
type KnowledgeNode struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Embedding  []float32      `json:"-"`
	DocumentID *int           `json:"document_id,omitempty"`
	URL        *string        `json:"url,omitempty"`
	Title      *string        `json:"title,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// KnowledgeNodeResponse represents a knowledge node for API responses (without embedding)
type KnowledgeNodeResponse struct {
	ID         int            `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	DocumentID *int           `json:"document_id,omitempty"`
	URL        *string        `json:"url,omitempty"`
	Title      *string        `json:"title,omitempty"`
}

// ToResponse converts KnowledgeNode to KnowledgeNodeResponse
func (kn *KnowledgeNode) ToResponse() KnowledgeNodeResponse {
	return KnowledgeNodeResponse{
		ID:         kn.ID,
		Name:       kn.Name,
		Type:       kn.Type,
		Properties: kn.Properties,
		DocumentID: kn.DocumentID,
		URL:        kn.URL,
		Title:      kn.Title,
	}
}

// KnowledgeEdge represents a relationship in the knowledge graph
type KnowledgeEdge struct {
	ID               int            `json:"id"`
	SourceID         int            `json:"source_id"`
	TargetID         int            `json:"target_id"`
	RelationshipType string         `json:"relationship_type"`
	Properties       map[string]any `json:"properties"`
	DocumentID       *int           `json:"document_id,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// KnowledgeEdgeResponse represents a knowledge edge for API responses
type KnowledgeEdgeResponse struct {
	ID               int            `json:"id"`
	SourceID         int            `json:"source_id"`
	TargetID         int            `json:"target_id"`
	RelationshipType string         `json:"relationship_type"`
	Properties       map[string]any `json:"properties"`
	DocumentID       *int           `json:"document_id,omitempty"`
}

// ToResponse converts KnowledgeEdge to KnowledgeEdgeResponse
func (ke *KnowledgeEdge) ToResponse() KnowledgeEdgeResponse {
	return KnowledgeEdgeResponse{
		ID:               ke.ID,
		SourceID:         ke.SourceID,
		TargetID:         ke.TargetID,
		RelationshipType: ke.RelationshipType,
		Properties:       ke.Properties,
		DocumentID:       ke.DocumentID,
	}
}

// KnowledgeGraph represents the complete knowledge graph
type KnowledgeGraph struct {
	Nodes []KnowledgeNodeResponse `json:"nodes"`
	Edges []KnowledgeEdgeResponse `json:"edges"`
}

// Entity represents an extracted entity
type Entity struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Relationship represents an extracted relationship
type Relationship struct {
	SourceID         int            `json:"source_id"`
	TargetID         int            `json:"target_id"`
	RelationshipType string         `json:"relationship_type"`
	Properties       map[string]any `json:"properties"`
}

// URLQueueItem represents an item in the URL queue
type URLQueueItem struct {
	ID         int       `json:"id"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	RetryCount int       `json:"retry_count"`
	DocumentID int       `json:"document_id,omitempty"`
}

// MCPLog represents a log entry for an MCP request/response
type MCPLog struct {
	ID        int             `json:"id"`
	RequestID string          `json:"request_id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Response  json.RawMessage `json:"response"`
	Error     json.RawMessage `json:"error"`
	CreatedAt time.Time       `json:"created_at"`
}

// VectorData represents a vector embedding with position information
type VectorData struct {
	ID            int       `json:"id"`
	Content       string    `json:"content"`
	Embedding     []float32 `json:"embedding"`
	ChunkIndex    int       `json:"chunk_index"`
	StartPosition int       `json:"start_position"`
	EndPosition   int       `json:"end_position"`
}

// Request and Response types

// ProcessDocumentRequest represents a request to process a document
type ProcessDocumentRequest struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// QueryRequest represents a request to query the service
type QueryRequest struct {
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	Query   string `json:"query,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// QueryResponse represents the response from a query
type QueryResponse struct {
	Results []SearchResult `json:"results"`
}

// SearchResult represents a search result with comprehensive information
type SearchResult struct {
	Content      string   `json:"content"`
	Score        float64  `json:"score"`
	DocumentID   int      `json:"document_id"`
	URL          string   `json:"url"`
	Title        string   `json:"title"`
	Source       string   `json:"source,omitempty"`        // For backward compatibility
	RelatedNodes []string `json:"related_nodes,omitempty"` // For backward compatibility
}

// IngestRequest represents a request to ingest new data
type IngestRequest struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}
