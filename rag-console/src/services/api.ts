import axios from 'axios';

const API_BASE_URL = '/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface ProcessDocumentRequest {
  url: string;
  title?: string;
  content?: string;
}

export interface QueryRequest {
  query: string;
}

export interface QueryResponse {
  results: Array<{
    content: string;
    score: number;
    document_id: number;
    url: string;
    title: string;
  }>;
}

export interface KnowledgeGraphResponse {
  nodes: Array<{
    id: number;
    name: string;
    type: string;
    properties: Record<string, any>;
    document_id?: number;
    url?: string;
    title?: string;
  }>;
  edges: Array<{
    id: number;
    source_id: number;
    target_id: number;
    relationship_type: string;
    properties: Record<string, any>;
  }>;
}

export interface URLQueueItem {
  id: number;
  url: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  error?: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
  document_id?: number;
}

export const apiService = {
  // Document processing
  processDocument: async (data: ProcessDocumentRequest) => {
    const response = await api.post('/documents', data);
    return response.data;
  },

  // Query
  query: async (data: QueryRequest): Promise<QueryResponse> => {
    const response = await api.post('/query', data);
    return response.data;
  },

  // Knowledge graph
  getKnowledgeGraph: async (query: string): Promise<KnowledgeGraphResponse> => {
    const response = await api.get(`/graph?query=${encodeURIComponent(query)}`);
    return response.data;
  },

  // URL queue
  getURLQueue: async (): Promise<URLQueueItem[]> => {
    const response = await api.get('/queue');
    return response.data.queue;
  },

  // Delete URL
  deleteURL: async (id: number) => {
    await api.delete(`/queue/${id}`);
  },

  // Reindex URL
  reindexURL: async (id: number) => {
    await api.post(`/queue/${id}/reindex`);
  },

  // Document detail endpoints
  getDocumentById: async (id: number) => {
    const response = await api.get(`/documents/${id}`);
    return response.data;
  },

  getDocumentChunks: async (id: number) => {
    const response = await api.get(`/documents/${id}/chunks`);
    return response.data.chunks;
  },

  getDocumentVectors: async (id: number) => {
    const response = await api.get(`/documents/${id}/vectors`);
    return response.data.vectors;
  },

  getDocumentKnowledgeGraph: async (documentId: number) => {
    const response = await api.get(`/documents/${documentId}/graph`);
    return response.data;
  },

  // MCP API
  getMCPLogs: async () => {
    const response = await api.get('/mcp-logs');
    return response.data;
  },
};

export default apiService; 