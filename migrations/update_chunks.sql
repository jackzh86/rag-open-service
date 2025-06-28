-- Add position columns to existing chunks table
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS start_position INTEGER DEFAULT 0;
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS end_position INTEGER DEFAULT 0;

-- Update existing chunks with estimated positions (this is a fallback for existing data)
-- For existing chunks, we'll estimate positions based on chunk_index
UPDATE chunks 
SET start_position = chunk_index * 1000,
    end_position = (chunk_index + 1) * 1000
WHERE start_position = 0 OR end_position = 0; 