-- Enable the pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    url TEXT,
    title TEXT,
    content TEXT,
    embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create chunks table
CREATE TABLE IF NOT EXISTS chunks (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id),
    content TEXT,
    embedding vector(1536),
    chunk_index INTEGER,
    start_position INTEGER,
    end_position INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create knowledge graph nodes table
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    id SERIAL PRIMARY KEY,
    name TEXT,
    type TEXT,
    properties JSONB,
    embedding vector(1536),
    document_id INTEGER REFERENCES documents(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name, type)
);

-- Create knowledge graph edges table
CREATE TABLE IF NOT EXISTS knowledge_edges (
    id SERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES knowledge_nodes(id),
    target_id INTEGER REFERENCES knowledge_nodes(id),
    relationship_type TEXT,
    properties JSONB,
    document_id INTEGER REFERENCES documents(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_id, target_id, relationship_type)
);

-- Create URL queue table
CREATE TABLE IF NOT EXISTS url_queue (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_documents_embedding ON documents USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_chunks_embedding ON chunks USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_knowledge_nodes_embedding ON knowledge_nodes USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_nodes_document_id ON knowledge_nodes(document_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_edges_document_id ON knowledge_edges(document_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_edges_source_id ON knowledge_edges(source_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_edges_target_id ON knowledge_edges(target_id);

-- Create indexes for URL queue
CREATE INDEX IF NOT EXISTS idx_url_queue_status ON url_queue(status);
CREATE INDEX IF NOT EXISTS idx_url_queue_created_at ON url_queue(created_at);

-- Create unique indexes to prevent duplicate URLs
CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_url_unique ON documents(url);
CREATE UNIQUE INDEX IF NOT EXISTS idx_url_queue_url_unique ON url_queue(url); 