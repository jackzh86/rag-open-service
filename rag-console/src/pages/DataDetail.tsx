import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeftIcon } from '@heroicons/react/24/outline';
import apiService from '../services/api';

interface DataDetailProps {}

interface DocumentData {
  id: number;
  url: string;
  title: string;
  content: string;
  created_at: string;
  updated_at: string;
}

interface ChunkData {
  id: number;
  content: string;
  chunk_index: number;
  start_position: number;
  end_position: number;
  created_at: string;
}

interface VectorData {
  id: number;
  content: string;
  embedding: number[];
  chunk_index: number;
  start_position: number;
  end_position: number;
}

interface KnowledgeNode {
  id: number;
  name: string;
  type: string;
  properties: Record<string, any>;
}

interface KnowledgeEdge {
  id: number;
  source_id: number;
  target_id: number;
  relationship_type: string;
  properties: Record<string, any>;
}

interface KnowledgeGraph {
  nodes: KnowledgeNode[];
  edges: KnowledgeEdge[];
}

const DataDetail: React.FC<DataDetailProps> = () => {
  const { id } = useParams<{ id: string }>();
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'chunks' | 'vectors' | 'graph'>('chunks');
  const [documentData, setDocumentData] = useState<DocumentData | null>(null);
  const [chunks, setChunks] = useState<ChunkData[]>([]);
  const [vectors, setVectors] = useState<VectorData[]>([]);
  const [graphData, setGraphData] = useState<{
    nodes: Array<{
      id: number;
      name: string;
      type: string;
      properties: any;
      document_id?: number;
    }>;
    edges: Array<{
      id: number;
      source_id: number;
      target_id: number;
      relationship_type: string;
      properties: any;
      document_id?: number;
    }>;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [highlightedChunk, setHighlightedChunk] = useState<number | null>(null);

  useEffect(() => {
    if (id) {
      loadDocumentData();
    }
  }, [id]);

  // Clear highlight when switching tabs
  useEffect(() => {
    setHighlightedChunk(null);
  }, [activeTab]);

  // Scroll to chunk when highlighted
  useEffect(() => {
    if (highlightedChunk !== null) {
      const chunkElement = document.getElementById(`chunk-${highlightedChunk}`);
      const vectorElement = document.getElementById(`vector-${highlightedChunk}`);
      const targetElement = chunkElement || vectorElement;
      
      if (targetElement) {
        targetElement.scrollIntoView({ behavior: 'smooth', block: 'center' });
      }

      // Also scroll the document content to the highlighted section
      setTimeout(() => {
        const documentContent = document.querySelector('.prose');
        if (documentContent && chunks.length > 0 && highlightedChunk !== null) {
          const selectedChunk = chunks[highlightedChunk];
          if (selectedChunk) {
            // Calculate scroll position based on chunk's start position
            const documentLength = documentData?.content.length || 0;
            if (documentLength > 0) {
              const scrollRatio = selectedChunk.start_position / documentLength;
              const scrollHeight = documentContent.scrollHeight;
              const scrollTop = scrollRatio * scrollHeight;
              
              documentContent.scrollTo({
                top: Math.max(0, scrollTop - 100), // Offset by 100px to show some context
                behavior: 'smooth'
              });
            }
          }
        }
      }, 100);
    }
  }, [highlightedChunk, chunks.length, documentData]);

  // Function to highlight text in document content
  const highlightTextInDocument = (text: string, searchText: string) => {
    if (!searchText || !text) return text;
    
    // Simple approach: find the first occurrence and highlight it
    const index = text.indexOf(searchText);
    if (index === -1) {
      // Try with cleaned text
      const cleanSearchText = searchText.trim().replace(/\s+/g, ' ');
      const cleanIndex = text.indexOf(cleanSearchText);
      if (cleanIndex === -1) return text;
      
      const before = text.substring(0, cleanIndex);
      const highlighted = text.substring(cleanIndex, cleanIndex + cleanSearchText.length);
      const after = text.substring(cleanIndex + cleanSearchText.length);
      
      return `${before}<mark class="bg-yellow-300 border-2 border-yellow-500 px-1 rounded font-semibold shadow-sm">${highlighted}</mark>${after}`;
    }
    
    const before = text.substring(0, index);
    const highlighted = text.substring(index, index + searchText.length);
    const after = text.substring(index + searchText.length);
    
    return `${before}<mark class="bg-yellow-300 border-2 border-yellow-500 px-1 rounded font-semibold shadow-sm">${highlighted}</mark>${after}`;
  };

  // Function to render document content with highlighting
  const renderDocumentContent = () => {
    if (!documentData) return null;

    let content = documentData.content;
    
    // If a chunk is highlighted, highlight its content in the document
    if (highlightedChunk !== null && chunks.length > highlightedChunk) {
      const selectedChunk = chunks[highlightedChunk];
      if (selectedChunk) {
        // Try to find and highlight the chunk content
        content = highlightTextInDocument(content, selectedChunk.content);
        
        // If exact match not found, try with cleaned content
        if (content === documentData.content) {
          const cleanChunkContent = selectedChunk.content.trim().replace(/\s+/g, ' ');
          content = highlightTextInDocument(content, cleanChunkContent);
        }
      }
    }

    return (
      <div 
        className="whitespace-pre-wrap text-gray-800 leading-relaxed"
        dangerouslySetInnerHTML={{ __html: content }}
      />
    );
  };

  // Function to highlight document section based on chunk position
  const highlightDocumentSection = () => {
    if (!documentData || highlightedChunk === null || chunks.length === 0) return null;

    const selectedChunk = chunks[highlightedChunk];
    if (!selectedChunk) return null;

    // Use precise position information from the chunk
    const startPos = selectedChunk.start_position;
    const endPos = selectedChunk.end_position;

    const before = documentData.content.substring(0, startPos);
    const highlighted = documentData.content.substring(startPos, endPos);
    const after = documentData.content.substring(endPos);

    return (
      <div 
        className="whitespace-pre-wrap text-gray-800 leading-relaxed"
        dangerouslySetInnerHTML={{ 
          __html: `${before}<mark class="bg-yellow-300 border-2 border-yellow-500 px-1 rounded font-semibold shadow-sm">${highlighted}</mark>${after}` 
        }}
      />
    );
  };

  const loadDocumentData = async () => {
    if (!id) return;
    
    setLoading(true);
    setError(null);
    
    try {
      // Load document data
      const docData = await apiService.getDocumentById(parseInt(id));
      setDocumentData(docData);
      
      // Load chunks
      const chunksData = await apiService.getDocumentChunks(parseInt(id));
      setChunks(chunksData);
      
      // Load vectors
      const vectorsData = await apiService.getDocumentVectors(parseInt(id));
      setVectors(vectorsData);
      
      // Load knowledge graph
      try {
        const graphData = await apiService.getDocumentKnowledgeGraph(parseInt(id));
        setGraphData(graphData);
      } catch (graphError) {
        console.error('Failed to load knowledge graph:', graphError);
        // Set empty graph data structure instead of null
        setGraphData({
          nodes: [],
          edges: []
        });
      }
      
    } catch (err) {
      console.error('Failed to load document data:', err);
      setError('Failed to load document data');
    } finally {
      setLoading(false);
    }
  };

  const renderChunksTab = () => (
    <div className="space-y-4">
      <h3 className="text-lg font-medium text-gray-900">Document Chunks</h3>
      {chunks.length === 0 ? (
        <p className="text-gray-500">No chunks found for this document.</p>
      ) : (
        <div className="space-y-6">
          {chunks.map((chunk, index) => (
            <div 
              key={chunk.id} 
              id={`chunk-${chunk.chunk_index}`}
              className={`p-4 rounded-lg border-l-4 cursor-pointer transition-colors ${
                highlightedChunk === chunk.chunk_index
                  ? 'bg-indigo-50 border-indigo-600 shadow-md'
                  : 'bg-gray-50 border-indigo-500 hover:bg-gray-100'
              }`}
              onClick={() => setHighlightedChunk(chunk.chunk_index)}
            >
              <div className="flex justify-between items-center mb-3">
                <div className="flex items-center space-x-3">
                  <span className="inline-flex items-center justify-center w-6 h-6 bg-indigo-100 text-indigo-800 text-xs font-medium rounded-full">
                    {chunk.chunk_index + 1}
                  </span>
                  <span className="text-sm font-medium text-gray-700">
                    Chunk #{chunk.chunk_index + 1}
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <span className="text-xs text-gray-500">
                    {new Date(chunk.created_at).toLocaleString()}
                  </span>
                  <span className="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded">
                    {chunk.content.length} chars
                  </span>
                </div>
              </div>
              <div className="bg-white p-3 rounded border">
                <p className="text-gray-800 whitespace-pre-wrap leading-relaxed text-sm">
                  {chunk.content}
                </p>
              </div>
              <div className="mt-2 text-xs text-gray-500">
                <span className="font-medium">Position:</span> Characters {chunk.start_position}-{chunk.end_position} 
                ({Math.round((chunk.start_position / (documentData?.content.length || 1)) * 100)}% through document)
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  const renderVectorsTab = () => (
    <div className="space-y-4">
      <h3 className="text-lg font-medium text-gray-900">Vector Embeddings</h3>
      {vectors.length === 0 ? (
        <p className="text-gray-500">No vectors found for this document.</p>
      ) : (
        <div className="space-y-6">
          {vectors.map((vector) => (
            <div 
              key={vector.id} 
              id={`vector-${vector.chunk_index}`}
              className={`p-4 rounded-lg border-l-4 cursor-pointer transition-colors ${
                highlightedChunk === vector.chunk_index
                  ? 'bg-green-50 border-green-600 shadow-md'
                  : 'bg-gray-50 border-green-500 hover:bg-gray-100'
              }`}
              onClick={() => setHighlightedChunk(vector.chunk_index)}
            >
              <div className="flex justify-between items-center mb-3">
                <div className="flex items-center space-x-3">
                  <span className="inline-flex items-center justify-center w-6 h-6 bg-green-100 text-green-800 text-xs font-medium rounded-full">
                    {vector.chunk_index + 1}
                  </span>
                  <span className="text-sm font-medium text-gray-700">
                    Vector #{vector.chunk_index + 1}
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded">
                    {vector.embedding.length} dimensions
                  </span>
                </div>
              </div>
              <div className="bg-white p-3 rounded border mb-3">
                <p className="text-gray-800 mb-2 text-sm font-medium">Content:</p>
                <p className="text-gray-700 whitespace-pre-wrap leading-relaxed text-sm">
                  {vector.content}
                </p>
              </div>
              <div className="bg-gray-100 p-3 rounded">
                <p className="text-xs text-gray-600 mb-2 font-medium">Vector (first 10 dimensions):</p>
                <div className="text-xs font-mono text-gray-600 overflow-x-auto">
                  <div className="whitespace-nowrap">
                    [{vector.embedding.slice(0, 10).map(v => v.toFixed(4)).join(', ')}
                    {vector.embedding.length > 10 && '...'}
                    ]
                  </div>
                </div>
                <div className="mt-2 text-xs text-gray-500">
                  <span className="font-medium">Position:</span> Characters {vector.start_position}-{vector.end_position} 
                  ({Math.round((vector.start_position / (documentData?.content.length || 1)) * 100)}% through document)
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  const renderGraphTab = () => (
    <div className="h-full overflow-auto">
      <div className="p-4">
        <h3 className="text-lg font-semibold mb-4">Knowledge Graph</h3>
        {graphData ? (
          <div className="space-y-6">
            {/* Entities Section */}
            <div>
              <h4 className="text-md font-medium mb-3 text-gray-700">Entities ({graphData.nodes?.length || 0})</h4>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {graphData.nodes?.map((node) => (
                  <div key={node.id} className="bg-white border border-gray-200 rounded-lg p-3 shadow-sm">
                    <div className="flex items-center justify-between mb-2">
                      <span className="font-medium text-gray-900">{node.name}</span>
                      <span className={`px-2 py-1 text-xs rounded-full ${
                        node.type === 'person' ? 'bg-blue-100 text-blue-800' :
                        node.type === 'organization' ? 'bg-green-100 text-green-800' :
                        node.type === 'location' ? 'bg-purple-100 text-purple-800' :
                        'bg-gray-100 text-gray-800'
                      }`}>
                        {node.type}
                      </span>
                    </div>
                    {node.properties && Object.keys(node.properties).length > 0 && (
                      <div className="text-xs text-gray-600">
                        {Object.entries(node.properties).map(([key, value]) => (
                          <div key={key}>
                            <span className="font-medium">{key}:</span> {String(value)}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )) || []}
              </div>
            </div>

            {/* Relationships Section */}
            <div>
              <h4 className="text-md font-medium mb-3 text-gray-700">Relationships ({graphData.edges?.length || 0})</h4>
              <div className="space-y-2">
                {graphData.edges?.map((edge) => {
                  const sourceNode = graphData.nodes?.find(n => n.id === edge.source_id);
                  const targetNode = graphData.nodes?.find(n => n.id === edge.target_id);
                  
                  return (
                    <div key={edge.id} className="bg-white border border-gray-200 rounded-lg p-3 shadow-sm">
                      <div className="flex items-center space-x-2">
                        <span className="font-medium text-blue-600">
                          {sourceNode?.name || `Node ${edge.source_id}`}
                        </span>
                        <span className="text-gray-500">→</span>
                        <span className={`px-2 py-1 text-xs rounded-full ${
                          edge.relationship_type === 'is_a' ? 'bg-yellow-100 text-yellow-800' :
                          edge.relationship_type === 'works_at' ? 'bg-indigo-100 text-indigo-800' :
                          edge.relationship_type === 'located_in' ? 'bg-pink-100 text-pink-800' :
                          'bg-gray-100 text-gray-800'
                        }`}>
                          {edge.relationship_type}
                        </span>
                        <span className="text-gray-500">→</span>
                        <span className="font-medium text-green-600">
                          {targetNode?.name || `Node ${edge.target_id}`}
                        </span>
                      </div>
                      {edge.properties && Object.keys(edge.properties).length > 0 && (
                        <div className="mt-2 text-xs text-gray-600">
                          {Object.entries(edge.properties).map(([key, value]) => (
                            <div key={key}>
                              <span className="font-medium">{key}:</span> {String(value)}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  );
                }) || []}
              </div>
            </div>

            {(!graphData.nodes || graphData.nodes.length === 0) && (!graphData.edges || graphData.edges.length === 0) && (
              <div className="text-center py-8 text-gray-500">
                <p>No knowledge graph data available for this document.</p>
                <p className="text-sm mt-2">Entities and relationships will be extracted automatically during document processing.</p>
              </div>
            )}
          </div>
        ) : (
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-2 text-gray-600">Loading knowledge graph...</p>
          </div>
        )}
      </div>
    </div>
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
        <span className="ml-2 text-gray-600">Loading document data...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-8">
        <p className="text-red-600 mb-4">{error}</p>
        <Link
          to="/data-sources"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
        >
          <ArrowLeftIcon className="h-4 w-4 mr-2" />
          Back to Data Sources
        </Link>
      </div>
    );
  }

  if (!documentData) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-600 mb-4">Document not found</p>
        <Link
          to="/data-sources"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
        >
          <ArrowLeftIcon className="h-4 w-4 mr-2" />
          Back to Data Sources
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/data-sources"
            className="inline-flex items-center px-3 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
          >
            <ArrowLeftIcon className="h-4 w-4 mr-2" />
            Back
          </Link>
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">Document Details</h1>
            <p className="text-sm text-gray-500">{documentData.url}</p>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 h-[calc(100vh-200px)]">
        {/* Left Panel - Document Content */}
        <div className="bg-white shadow rounded-lg flex flex-col">
          <div className="px-6 py-4 border-b border-gray-200 flex-shrink-0">
            <h2 className="text-lg font-medium text-gray-900">Document Content</h2>
            <p className="text-sm text-gray-500 mt-1">
              {new Date(documentData.created_at).toLocaleString()}
            </p>
          </div>
          <div className="p-6 flex-1 overflow-y-auto">
            <h3 className="text-lg font-medium text-gray-900 mb-4">{documentData.title}</h3>
            <div className="prose max-w-none">
              {highlightedChunk !== null && (
                <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-yellow-800">
                      📍 Highlighting Chunk {highlightedChunk + 1} of {chunks.length}
                    </span>
                    <button
                      onClick={() => setHighlightedChunk(null)}
                      className="text-xs text-yellow-600 hover:text-yellow-800 underline"
                    >
                      Clear highlight
                    </button>
                  </div>
                </div>
              )}
              {highlightedChunk !== null ? highlightDocumentSection() : renderDocumentContent()}
              {activeTab === 'chunks' && chunks.length > 0 && (
                <div className="mt-6 p-4 bg-blue-50 rounded-lg border border-blue-200">
                  <div className="flex justify-between items-center mb-2">
                    <h4 className="text-sm font-medium text-blue-900">Chunk Navigation</h4>
                    {highlightedChunk !== null && (
                      <button
                        onClick={() => setHighlightedChunk(null)}
                        className="text-xs text-blue-600 hover:text-blue-800 underline"
                      >
                        Clear selection
                      </button>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {chunks.map((chunk) => (
                      <button
                        key={chunk.id}
                        onClick={() => setHighlightedChunk(chunk.chunk_index)}
                        className={`px-3 py-1 text-xs rounded-full border ${
                          highlightedChunk === chunk.chunk_index
                            ? 'bg-blue-600 text-white border-blue-600'
                            : 'bg-white text-blue-700 border-blue-300 hover:bg-blue-100'
                        }`}
                      >
                        Chunk {chunk.chunk_index + 1}
                      </button>
                    ))}
                  </div>
                  {highlightedChunk !== null && (
                    <div className="mt-3 p-3 bg-white rounded border">
                      <p className="text-sm text-gray-700">
                        <strong>Selected:</strong> Chunk {highlightedChunk + 1} 
                        ({Math.round(((highlightedChunk + 1) / chunks.length) * 100)}% through document)
                      </p>
                    </div>
                  )}
                </div>
              )}
              {activeTab === 'vectors' && vectors.length > 0 && (
                <div className="mt-6 p-4 bg-green-50 rounded-lg border border-green-200">
                  <div className="flex justify-between items-center mb-2">
                    <h4 className="text-sm font-medium text-green-900">Vector Navigation</h4>
                    {highlightedChunk !== null && (
                      <button
                        onClick={() => setHighlightedChunk(null)}
                        className="text-xs text-green-600 hover:text-green-800 underline"
                      >
                        Clear selection
                      </button>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {vectors.map((vector) => (
                      <button
                        key={vector.id}
                        onClick={() => setHighlightedChunk(vector.chunk_index)}
                        className={`px-3 py-1 text-xs rounded-full border ${
                          highlightedChunk === vector.chunk_index
                            ? 'bg-green-600 text-white border-green-600'
                            : 'bg-white text-green-700 border-green-300 hover:bg-green-100'
                        }`}
                      >
                        Vector {vector.chunk_index + 1}
                      </button>
                    ))}
                  </div>
                  {highlightedChunk !== null && (
                    <div className="mt-3 p-3 bg-white rounded border">
                      <p className="text-sm text-gray-700">
                        <strong>Selected:</strong> Vector {highlightedChunk + 1} 
                        ({Math.round(((highlightedChunk + 1) / vectors.length) * 100)}% through document)
                      </p>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right Panel - Tabs */}
        <div className="bg-white shadow rounded-lg flex flex-col">
          <div className="px-6 py-4 border-b border-gray-200 flex-shrink-0">
            <nav className="flex space-x-8">
              <button
                onClick={() => setActiveTab('chunks')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'chunks'
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Chunks
              </button>
              <button
                onClick={() => setActiveTab('vectors')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'vectors'
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Vectors
              </button>
              <button
                onClick={() => setActiveTab('graph')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'graph'
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Knowledge Graph
              </button>
            </nav>
          </div>
          <div className="p-6 flex-1 overflow-y-auto">
            {activeTab === 'chunks' && renderChunksTab()}
            {activeTab === 'vectors' && renderVectorsTab()}
            {activeTab === 'graph' && renderGraphTab()}
          </div>
        </div>
      </div>
    </div>
  );
};

export default DataDetail; 