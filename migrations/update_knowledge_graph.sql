-- Add document_id column to knowledge_nodes table
ALTER TABLE knowledge_nodes ADD COLUMN IF NOT EXISTS document_id INTEGER REFERENCES documents(id);

-- Add document_id column to knowledge_edges table
ALTER TABLE knowledge_edges ADD COLUMN IF NOT EXISTS document_id INTEGER REFERENCES documents(id);

-- Create indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_knowledge_nodes_document_id ON knowledge_nodes(document_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_edges_document_id ON knowledge_edges(document_id);

-- Add unique constraints for knowledge_nodes table
-- This allows ON CONFLICT to work properly
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'knowledge_nodes_name_type_unique') THEN
        ALTER TABLE knowledge_nodes ADD CONSTRAINT knowledge_nodes_name_type_unique UNIQUE (name, type);
    END IF;
END $$;

-- Add unique constraints for knowledge_edges table
-- This allows ON CONFLICT to work properly
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'knowledge_edges_source_target_relationship_unique') THEN
        ALTER TABLE knowledge_edges ADD CONSTRAINT knowledge_edges_source_target_relationship_unique UNIQUE (source_id, target_id, relationship_type);
    END IF;
END $$; 