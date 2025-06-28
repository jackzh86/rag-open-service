package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

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

func main() {
	// Get HTTP endpoint from environment variable or use default
	httpEndpoint := os.Getenv("MCP_HTTP_ENDPOINT")
	if httpEndpoint == "" {
		httpEndpoint = "http://localhost:8080/mcp"
	}

	log.Printf("MCP Proxy starting - forwarding to: %s", httpEndpoint)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Read from stdin and forward to HTTP endpoint
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		log.Printf("Received request: %s", line)

		// Parse the JSON-RPC request
		var req MCPRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		// Forward to HTTP endpoint
		response, err := forwardToHTTP(client, httpEndpoint, line)
		if err != nil {
			log.Printf("Error forwarding to HTTP: %v", err)
			sendError(req.ID, -32603, "Internal error", err.Error())
			continue
		}

		// Send response to stdout
		fmt.Println(response)
		log.Printf("Sent response: %s", response)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stdin: %v", err)
	}
}

func forwardToHTTP(client *http.Client, endpoint, requestBody string) (string, error) {
	// Create HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read HTTP response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return string(responseBody), nil
}

func sendError(id interface{}, code int, message, data string) {
	errorResp := MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	responseBytes, _ := json.Marshal(errorResp)
	fmt.Println(string(responseBytes))
}
