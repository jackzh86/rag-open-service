package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"rag-data-service/models"
	"rag-data-service/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	ragService *service.RAGService
}

func NewHandler(ragService *service.RAGService) *Handler {
	return &Handler{
		ragService: ragService,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Document processing endpoints
		r.Post("/documents", h.handleProcessDocument)

		// Query endpoints
		r.Post("/query", h.handleQuery)
		r.Get("/graph", h.handleGetGraph)

		// URL queue endpoints
		r.Get("/queue", h.handleGetQueue)
		r.Delete("/queue/{id}", h.handleDeleteURL)
		r.Post("/queue/{id}/reindex", h.handleReindexURL)

		// Document detail endpoints
		r.Get("/documents/{id}", h.handleGetDocument)
		r.Get("/documents/{id}/chunks", h.handleGetDocumentChunks)
		r.Get("/documents/{id}/vectors", h.handleGetDocumentVectors)
		r.Get("/documents/{id}/graph", h.handleGetDocumentGraph)

		// MCP logs endpoint
		r.Get("/mcp-logs", h.handleGetMCPLogs)
	})
}

func (h *Handler) handleProcessDocument(w http.ResponseWriter, r *http.Request) {
	var req models.ProcessDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// If only URL is provided, queue it for background processing
	if req.Content == "" {
		if err := h.ragService.QueueURL(r.Context(), req.URL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "URL queued for processing",
		})
		return
	}

	// If both URL and content are provided, process immediately
	if err := h.ragService.ProcessDocument(r.Context(), &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Document processed successfully",
	})
}

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	resp, err := h.ragService.Query(r.Context(), req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	// The service layer will now handle the filtering.
	// We pass the query string directly to it.
	nodes, edges, err := h.ragService.GetKnowledgeGraph(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	graph := map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

func (h *Handler) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	queue, err := h.ragService.GetURLQueue(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"queue": queue,
	})
}

func (h *Handler) handleDeleteURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Add debug logging
	log.Printf("Attempting to delete URL with ID: %s", id)

	if err := h.ragService.DeleteURLByID(r.Context(), id); err != nil {
		log.Printf("DeleteURLByID error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully deleted URL with ID: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleReindexURL(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := h.ragService.ReindexURLByID(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	document, err := h.ragService.GetDocumentByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(document)
}

func (h *Handler) handleGetDocumentChunks(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	chunks, err := h.ragService.GetDocumentChunks(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chunks": chunks,
	})
}

func (h *Handler) handleGetDocumentVectors(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	vectors, err := h.ragService.GetDocumentVectors(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vectors": vectors,
	})
}

func (h *Handler) handleGetDocumentGraph(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	nodes, edges, err := h.ragService.GetKnowledgeGraphByDocument(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	graph := map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

func (h *Handler) handleGetMCPLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.ragService.GetMCPLogs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}
