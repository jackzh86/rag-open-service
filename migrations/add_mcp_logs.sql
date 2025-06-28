-- migrations/add_mcp_logs.sql
CREATE TABLE IF NOT EXISTS mcp_logs (
    id SERIAL PRIMARY KEY,
    request_id VARCHAR(255),
    method VARCHAR(255) NOT NULL,
    params JSONB,
    response JSONB,
    error JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mcp_logs_method ON mcp_logs(method);
CREATE INDEX IF NOT EXISTS idx_mcp_logs_created_at ON mcp_logs(created_at); 