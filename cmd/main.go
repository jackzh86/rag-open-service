package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rag-data-service/config"
	"rag-data-service/handlers"
	"rag-data-service/mcp"
	"rag-data-service/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting RAG Data Service...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded successfully")

	// Construct database connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBConfig.Host,
		cfg.DBConfig.Port,
		cfg.DBConfig.User,
		cfg.DBConfig.Password,
		cfg.DBConfig.DBName,
	)
	log.Printf("Database connection string: %s", connStr)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure database connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established successfully")

	// Initialize services
	ragService := service.NewRAGService(db, cfg.OpenAIKey, cfg.OpenAIBaseURL, cfg.MCPEndpoint)
	log.Println("RAG service initialized")

	// Create context that will be canceled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background workers
	numWorkers := 5
	go ragService.StartBackgroundWorkers(ctx, numWorkers)
	log.Printf("Started %d background workers", numWorkers)

	// Initialize handlers
	handler := handlers.NewHandler(ragService)
	log.Println("HTTP handlers initialized")

	// Initialize MCP handler
	mcpHandler := mcp.NewMCPHandler(ragService)
	log.Println("MCP handler initialized")

	// Setup router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"http://localhost:3000"}, // Adjust for your frontend URL
		AllowedOrigins:   []string{"*"}, // For development
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Register routes
	handler.RegisterRoutes(r)

	// Register MCP routes
	r.Post("/mcp", mcpHandler.HandleRequest)
	log.Println("Routes registered")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel context to stop background workers
	cancel()

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
