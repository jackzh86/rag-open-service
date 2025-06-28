CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_url_unique ON documents(url);
CREATE UNIQUE INDEX IF NOT EXISTS idx_url_queue_url_unique ON url_queue(url);