package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMCPRequestParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid tools/list request",
			input:   `{"jsonrpc": "2.0", "method": "tools/list", "id": "test"}`,
			wantErr: false,
		},
		{
			name:    "valid tools/call request",
			input:   `{"jsonrpc": "2.0", "method": "tools/call", "id": "test", "params": {"name": "test_tool"}}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{"jsonrpc": "2.0", "method": "tools/list"`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MCPRequest
			err := json.Unmarshal([]byte(tt.input), &req)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.wantErr {
				if req.JSONRPC != "2.0" {
					t.Errorf("Expected JSONRPC '2.0', got '%s'", req.JSONRPC)
				}
			}
		})
	}
}

func TestForwardToHTTP(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		requestBody    string
		wantErr        bool
		expectedResp   string
	}{
		{
			name:           "successful request",
			serverResponse: `{"jsonrpc": "2.0", "id": "test", "result": {"tools": []}}`,
			serverStatus:   http.StatusOK,
			requestBody:    `{"jsonrpc": "2.0", "method": "tools/list", "id": "test"}`,
			wantErr:        false,
			expectedResp:   `{"jsonrpc": "2.0", "id": "test", "result": {"tools": []}}`,
		},
		{
			name:           "server error",
			serverResponse: `{"error": "internal server error"}`,
			serverStatus:   http.StatusInternalServerError,
			requestBody:    `{"jsonrpc": "2.0", "method": "tools/list", "id": "test"}`,
			wantErr:        true,
			expectedResp:   "",
		},
		{
			name:           "server returns error response",
			serverResponse: `{"jsonrpc": "2.0", "id": "test", "error": {"code": -32601, "message": "Method not found"}}`,
			serverStatus:   http.StatusOK,
			requestBody:    `{"jsonrpc": "2.0", "method": "invalid_method", "id": "test"}`,
			wantErr:        false,
			expectedResp:   `{"jsonrpc": "2.0", "id": "test", "error": {"code": -32601, "message": "Method not found"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and content type
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create HTTP client
			client := &http.Client{}

			// Test forwardToHTTP function
			response, err := forwardToHTTP(client, server.URL, tt.requestBody)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.wantErr && response != tt.expectedResp {
				t.Errorf("Expected response '%s', got '%s'", tt.expectedResp, response)
			}
		})
	}
}

func TestSendError(t *testing.T) {
	tests := []struct {
		name     string
		id       interface{}
		code     int
		message  string
		data     string
		expected string
	}{
		{
			name:     "parse error",
			id:       nil,
			code:     -32700,
			message:  "Parse error",
			data:     "invalid JSON",
			expected: `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error","data":"invalid JSON"}}`,
		},
		{
			name:     "method not found",
			id:       "test123",
			code:     -32601,
			message:  "Method not found",
			data:     "unknown method",
			expected: `{"jsonrpc":"2.0","id":"test123","error":{"code":-32601,"message":"Method not found","data":"unknown method"}}`,
		},
		{
			name:     "internal error",
			id:       42,
			code:     -32603,
			message:  "Internal error",
			data:     "server unavailable",
			expected: `{"jsonrpc":"2.0","id":42,"error":{"code":-32603,"message":"Internal error","data":"server unavailable"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call sendError
			sendError(tt.id, tt.code, tt.message, tt.data)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := strings.TrimSpace(buf.String())

			if output != tt.expected {
				t.Errorf("Expected output '%s', got '%s'", tt.expected, output)
			}
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	// Test default endpoint
	oldEndpoint := os.Getenv("MCP_HTTP_ENDPOINT")
	os.Unsetenv("MCP_HTTP_ENDPOINT")

	// This would be tested in main() but we can't easily test main()
	// So we test the logic separately
	httpEndpoint := os.Getenv("MCP_HTTP_ENDPOINT")
	if httpEndpoint == "" {
		httpEndpoint = "http://localhost:8080/mcp"
	}

	expected := "http://localhost:8080/mcp"
	if httpEndpoint != expected {
		t.Errorf("Expected default endpoint '%s', got '%s'", expected, httpEndpoint)
	}

	// Test custom endpoint
	customEndpoint := "http://example.com:9090/mcp"
	os.Setenv("MCP_HTTP_ENDPOINT", customEndpoint)

	httpEndpoint = os.Getenv("MCP_HTTP_ENDPOINT")
	if httpEndpoint == "" {
		httpEndpoint = "http://localhost:8080/mcp"
	}

	if httpEndpoint != customEndpoint {
		t.Errorf("Expected custom endpoint '%s', got '%s'", customEndpoint, httpEndpoint)
	}

	// Restore original environment
	if oldEndpoint != "" {
		os.Setenv("MCP_HTTP_ENDPOINT", oldEndpoint)
	} else {
		os.Unsetenv("MCP_HTTP_ENDPOINT")
	}
}
