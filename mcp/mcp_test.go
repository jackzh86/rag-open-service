package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"rag-data-service/models"
)

// mockRAGService is a mock implementation of the RAGService for testing.
// It allows us to check which methods were called and with what arguments.
type mockRAGService struct {
	logMCPRequestFunc          func(logEntry *models.MCPLog)
	queueURLFunc               func(url string) error
	getURLQueueFunc            func() ([]models.URLQueueItem, error)
	getKnowledgeGraphFunc      func(query string) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error)
	getKnowledgeGraphByDocFunc func(docID int) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error)
	queryFunc                  func(query string) (*models.QueryResponse, error)
	processDocumentFunc        func(req *models.ProcessDocumentRequest) error
}

func (m *mockRAGService) LogMCPRequest(ctx context.Context, logEntry *models.MCPLog) error {
	if m.logMCPRequestFunc != nil {
		m.logMCPRequestFunc(logEntry)
	}
	return nil
}

func (m *mockRAGService) QueueURL(ctx context.Context, url string) error {
	if m.queueURLFunc != nil {
		return m.queueURLFunc(url)
	}
	return nil
}

func (m *mockRAGService) GetURLQueue(ctx context.Context) ([]models.URLQueueItem, error) {
	if m.getURLQueueFunc != nil {
		return m.getURLQueueFunc()
	}
	return nil, nil
}

func (m *mockRAGService) GetKnowledgeGraph(ctx context.Context, query string) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error) {
	if m.getKnowledgeGraphFunc != nil {
		return m.getKnowledgeGraphFunc(query)
	}
	return nil, nil, nil
}

func (m *mockRAGService) GetKnowledgeGraphByDocument(ctx context.Context, documentID int) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error) {
	if m.getKnowledgeGraphByDocFunc != nil {
		return m.getKnowledgeGraphByDocFunc(documentID)
	}
	return nil, nil, nil
}

func (m *mockRAGService) Query(ctx context.Context, query string) (*models.QueryResponse, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query)
	}
	return nil, nil
}

func (m *mockRAGService) ProcessDocument(ctx context.Context, req *models.ProcessDocumentRequest) error {
	if m.processDocumentFunc != nil {
		return m.processDocumentFunc(req)
	}
	return nil
}

func TestMCPHandler(t *testing.T) {
	t.Run("Handle tools/list request", func(t *testing.T) {
		// Setup
		logCalled := false
		mockService := &mockRAGService{
			logMCPRequestFunc: func(logEntry *models.MCPLog) {
				logCalled = true
				if logEntry.Method != "tools/list" {
					t.Errorf("expected method to be 'tools/list', got '%s'", logEntry.Method)
				}
			},
		}
		handler := NewMCPHandler(mockService)

		// Create request
		body := `{"jsonrpc": "2.0", "method": "tools/list", "id": "1"}`
		req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Execute
		handler.HandleRequest(rr, req)

		// Assert
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if !strings.Contains(rr.Body.String(), `"name":"queue_url"`) {
			t.Errorf("handler response body does not contain expected tool 'queue_url': got %v", rr.Body.String())
		}

		if !logCalled {
			t.Error("expected LogMCPRequest to be called, but it was not")
		}
	})

	t.Run("Handle tools/call for queue_url", func(t *testing.T) {
		// Setup
		logCalled := false
		queueURLCalled := false
		testURL := "https://example.com/test"

		mockService := &mockRAGService{
			logMCPRequestFunc: func(logEntry *models.MCPLog) {
				logCalled = true
			},
			queueURLFunc: func(url string) error {
				queueURLCalled = true
				if url != testURL {
					t.Errorf("expected queue_url to be called with '%s', got '%s'", testURL, url)
				}
				return nil
			},
		}
		handler := NewMCPHandler(mockService)

		// Create request
		body := `{"jsonrpc": "2.0", "method": "tools/call", "id": "2", "params": {"name": "queue_url", "arguments": {"url": "https://example.com/test"}}}`
		req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Execute
		handler.HandleRequest(rr, req)

		// Assert
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		if !strings.Contains(rr.Body.String(), "URL queued for processing") {
			t.Errorf("handler response body does not contain success message: got %v", rr.Body.String())
		}

		if !logCalled {
			t.Error("expected LogMCPRequest to be called, but it was not")
		}
		if !queueURLCalled {
			t.Error("expected QueueURL to be called, but it was not")
		}
	})

	t.Run("Handle tools/call for non-existent tool", func(t *testing.T) {
		// Setup
		logCalled := false
		mockService := &mockRAGService{
			logMCPRequestFunc: func(logEntry *models.MCPLog) {
				logCalled = true
				if logEntry.Error == nil {
					t.Error("expected log entry to contain an error, but it was nil")
				}
			},
		}
		handler := NewMCPHandler(mockService)

		// Create request
		body := `{"jsonrpc": "2.0", "method": "tools/call", "id": "3", "params": {"name": "non_existent_tool", "arguments": {}}}`
		req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// Execute
		handler.HandleRequest(rr, req)

		// Assert
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var resp MCPResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("could not unmarshal response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected an error in the response, but it was nil")
		}
		if resp.Error.Code != -32601 {
			t.Errorf("expected error code -32601, got %d", resp.Error.Code)
		}
		if !strings.Contains(resp.Error.Message, "Method not found") {
			t.Errorf("expected error message to contain 'Method not found', got '%s'", resp.Error.Message)
		}

		if !logCalled {
			t.Error("expected LogMCPRequest to be called, but it was not")
		}
	})
}
