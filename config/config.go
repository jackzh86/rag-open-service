package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	DBConfig      DBConfig
	OpenAIKey     string
	OpenAIBaseURL string
	MCPEndpoint   string
}

// DBConfig holds database configuration
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// loadEnvFile attempts to load .env file from multiple locations
func loadEnvFile() {
	// Try loading from current directory
	_ = godotenv.Load()

	// Try loading from project root (2 levels up from current directory)
	rootEnv := filepath.Join("..", "..", ".env")
	_ = godotenv.Load(rootEnv)

	// Try loading from absolute path in project root
	if wd, err := os.Getwd(); err == nil {
		// If we're in a subdirectory, go up to project root
		for {
			envPath := filepath.Join(wd, ".env")
			if _, err := os.Stat(envPath); err == nil {
				_ = godotenv.Load(envPath)
				break
			}
			// Go up one directory
			parent := filepath.Dir(wd)
			if parent == wd {
				break // Reached root directory
			}
			wd = parent
		}
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists, but don't error if it doesn't
	loadEnvFile()

	// Database configuration
	dbConfig := DBConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("DB_PORT", 5432),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("DB_NAME", "ragdb"),
	}

	// OpenAI configuration
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	// OpenAI base URL is optional, defaults to official API endpoint
	openAIBaseURL := getEnvOrDefault("OPENAI_API_BASE_URL", "https://api.openai.com/v1")

	// MCP configuration
	mcpEndpoint := os.Getenv("MCP_ENDPOINT")
	if mcpEndpoint == "" {
		return nil, fmt.Errorf("MCP_ENDPOINT environment variable is required")
	}

	return &Config{
		DBConfig:      dbConfig,
		OpenAIKey:     openAIKey,
		OpenAIBaseURL: openAIBaseURL,
		MCPEndpoint:   mcpEndpoint,
	}, nil
}

// LoadTestConfig loads configuration for testing
func LoadTestConfig() *Config {
	// Load .env file if it exists, but don't error if it doesn't
	loadEnvFile()

	// Database configuration
	dbConfig := DBConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("DB_PORT", 5432),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("DB_NAME", "ragdb"),
	}

	// OpenAI configuration
	openAIKey := os.Getenv("OPENAI_API_KEY")
	if openAIKey == "" {
		openAIKey = "test-key" // Fallback for testing
	}

	// OpenAI base URL is optional, defaults to official API endpoint
	openAIBaseURL := getEnvOrDefault("OPENAI_API_BASE_URL", "https://api.openai.com/v1")

	// MCP configuration
	mcpEndpoint := os.Getenv("MCP_ENDPOINT")
	if mcpEndpoint == "" {
		mcpEndpoint = "http://localhost:8080" // Fallback for testing
	}

	return &Config{
		DBConfig:      dbConfig,
		OpenAIKey:     openAIKey,
		OpenAIBaseURL: openAIBaseURL,
		MCPEndpoint:   mcpEndpoint,
	}
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}
