package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"rag-data-service/models"
)

// RAGServicer defines the interface required by MCPHandler from the RAGService.
// This allows for mocking in tests.
type RAGServicer interface {
	LogMCPRequest(ctx context.Context, logEntry *models.MCPLog) error
	QueueURL(ctx context.Context, url string) error
	GetURLQueue(ctx context.Context) ([]models.URLQueueItem, error)
	GetKnowledgeGraph(ctx context.Context, query string) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error)
	GetKnowledgeGraphByDocument(ctx context.Context, documentID int) ([]models.KnowledgeNodeResponse, []models.KnowledgeEdgeResponse, error)
	Query(ctx context.Context, query string) (*models.QueryResponse, error)
	ProcessDocument(ctx context.Context, req *models.ProcessDocumentRequest) error
}

// MCPRequest represents a request from the MCP client
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents a response to the MCP client
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in MCP protocol
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPHandler handles MCP protocol requests
type MCPHandler struct {
	ragService RAGServicer
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(ragService RAGServicer) *MCPHandler {
	return &MCPHandler{
		ragService: ragService,
	}
}

// HandleRequest handles incoming MCP requests
func (h *MCPHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("MCP HandleRequest: Received new request")
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, nil, -32700, "Parse error", err.Error(), nil)
		return
	}
	log.Printf("MCP HandleRequest: Parsed request ID: %v, Method: %s", req.ID, req.Method)

	// Initialize log entry
	var requestIDStr string
	if req.ID != nil {
		requestIDStr = fmt.Sprintf("%v", req.ID)
	}
	logEntry := &models.MCPLog{
		RequestID: requestIDStr,
		Method:    req.Method,
	}
	log.Printf("MCP HandleRequest: Initialized logEntry with RequestID: %s", logEntry.RequestID)

	// Convert params to JSON string for logging
	paramsBytes, _ := json.Marshal(req.Params)
	logEntry.Params = paramsBytes

	// Defer logging the response
	defer func() {
		log.Println("MCP HandleRequest: Executing deferred log function.")
		if err := h.ragService.LogMCPRequest(r.Context(), logEntry); err != nil {
			log.Printf("MCP HandleRequest: FATAL: Failed to log MCP request in defer: %v", err)
		} else {
			log.Println("MCP HandleRequest: Successfully called LogMCPRequest in defer (err is nil).")
		}
	}()

	// Handle different methods
	switch req.Method {
	case "initialize":
		h.handleInitialize(w, &req, logEntry)
	case "tools/list":
		h.handleToolsList(w, &req, logEntry)
	case "tools/call":
		h.handleToolsCall(w, &req, logEntry)
	case "notifications/cancel":
		h.handleCancel(w, &req, logEntry)
	default:
		h.sendError(w, req.ID, -32601, "Method not found", fmt.Sprintf("Method %s not found", req.Method), logEntry)
	}
}

// handleInitialize handles the initialize request
func (h *MCPHandler) handleInitialize(w http.ResponseWriter, req *MCPRequest, logEntry *models.MCPLog) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{
					"listChanged": false,
				},
			},
			"serverInfo": map[string]interface{}{
				"name":    "rag-data-service",
				"version": "1.0.0",
			},
		},
	}

	h.sendResponse(w, response, logEntry)
}

// handleToolsList handles the tools/list request
func (h *MCPHandler) handleToolsList(w http.ResponseWriter, req *MCPRequest, logEntry *models.MCPLog) {
	tools := []map[string]interface{}{
		{
			"name":        "process_document",
			"description": "Process a document by URL or content and store it in the knowledge base",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL of the document to process",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Title of the document",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content of the document (optional if URL is provided)",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "query_knowledge_base",
			"description": "Query the knowledge base for relevant information",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The query to search for in the knowledge base",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "get_knowledge_graph",
			"description": "Get the knowledge graph with entities and relationships",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"document_id": map[string]interface{}{
						"type":        "integer",
						"description": "Optional document ID to get graph for specific document",
					},
				},
			},
		},
		{
			"name":        "queue_url",
			"description": "Add a URL to the processing queue for background processing",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to add to the processing queue",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			"name":        "get_queue_status",
			"description": "Get the status of URLs in the processing queue",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}

	h.sendResponse(w, response, logEntry)
}

// handleToolsCall handles the tools/call request
func (h *MCPHandler) handleToolsCall(w http.ResponseWriter, req *MCPRequest, logEntry *models.MCPLog) {
	// Parse the call request
	var callReq struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	// Convert params to the expected structure
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		h.sendError(w, req.ID, -32602, "Invalid params", err.Error(), logEntry)
		return
	}

	if err := json.Unmarshal(paramsBytes, &callReq); err != nil {
		h.sendError(w, req.ID, -32602, "Invalid params", err.Error(), logEntry)
		return
	}

	// Handle different tool calls
	var responseResult interface{}
	var callErr error

	switch callReq.Name {
	case "process_document":
		responseResult, callErr = h.handleProcessDocument(callReq.Arguments)
	case "query_knowledge_base":
		responseResult, callErr = h.handleQueryKnowledgeBase(callReq.Arguments)
	case "get_knowledge_graph":
		responseResult, callErr = h.handleGetKnowledgeGraph(callReq.Arguments)
	case "queue_url":
		responseResult, callErr = h.handleQueueURL(callReq.Arguments)
	case "get_queue_status":
		responseResult, callErr = h.handleGetQueueStatus(callReq.Arguments)
	default:
		h.sendError(w, req.ID, -32601, "Method not found", fmt.Sprintf("Tool %s not found", callReq.Name), logEntry)
		return
	}

	if callErr != nil {
		h.sendError(w, req.ID, -32603, "Internal error", callErr.Error(), logEntry)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  responseResult,
	}

	h.sendResponse(w, response, logEntry)
}

// handleProcessDocument handles the process_document tool call
func (h *MCPHandler) handleProcessDocument(args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required and must be a string")
	}

	title, _ := args["title"].(string)
	content, _ := args["content"].(string)

	req := &models.ProcessDocumentRequest{
		URL:     url,
		Title:   title,
		Content: content,
	}

	if content == "" {
		// Queue for background processing
		err := h.ragService.QueueURL(context.Background(), url)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"message": "URL queued for background processing",
			"url":     url,
		}, nil
	}

	// Process immediately
	err := h.ragService.ProcessDocument(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message": "Document processed successfully",
		"url":     url,
	}, nil
}

// handleQueryKnowledgeBase handles the query_knowledge_base tool call
func (h *MCPHandler) handleQueryKnowledgeBase(args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	resp, err := h.ragService.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"query":   query,
		"results": resp.Results,
	}, nil
}

// handleGetKnowledgeGraph handles the get_knowledge_graph tool call
func (h *MCPHandler) handleGetKnowledgeGraph(args map[string]interface{}) (interface{}, error) {
	var query string
	if q, ok := args["query"].(string); ok {
		query = q
	}

	if documentID, ok := args["document_id"].(float64); ok {
		// Get graph for specific document
		nodes, edges, err := h.ragService.GetKnowledgeGraphByDocument(context.Background(), int(documentID))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"document_id": int(documentID),
			"nodes":       nodes,
			"edges":       edges,
		}, nil
	}

	// Get all or filtered knowledge graph
	nodes, edges, err := h.ragService.GetKnowledgeGraph(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}, nil
}

// handleQueueURL handles the queue_url tool call
func (h *MCPHandler) handleQueueURL(args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required and must be a string")
	}

	err := h.ragService.QueueURL(context.Background(), url)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"message": "URL queued for processing",
		"url":     url,
	}, nil
}

// handleGetQueueStatus handles the get_queue_status tool call
func (h *MCPHandler) handleGetQueueStatus(args map[string]interface{}) (interface{}, error) {
	queue, err := h.ragService.GetURLQueue(context.Background())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"queue": queue,
	}, nil
}

// handleCancel handles the notifications/cancel request
func (h *MCPHandler) handleCancel(w http.ResponseWriter, req *MCPRequest, logEntry *models.MCPLog) {
	// For now, just acknowledge the cancel request
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  nil,
	}

	h.sendResponse(w, response, logEntry)
}

// sendResponse sends a JSON response
func (h *MCPHandler) sendResponse(w http.ResponseWriter, response MCPResponse, logEntry *models.MCPLog) {
	w.Header().Set("Content-Type", "application/json")
	responseBytes, _ := json.Marshal(response.Result)
	logEntry.Response = responseBytes
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error response
func (h *MCPHandler) sendError(w http.ResponseWriter, id interface{}, code int, message, data string, logEntry *models.MCPLog) {
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	errorBytes, _ := json.Marshal(response.Error)
	logEntry.Error = errorBytes

	h.sendResponse(w, response, logEntry)
}
