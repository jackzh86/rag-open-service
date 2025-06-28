import React, { useState } from 'react';
import { MagnifyingGlassIcon, DocumentTextIcon, ShareIcon } from '@heroicons/react/24/outline';
import apiService, { QueryResponse, KnowledgeGraphResponse } from '../services/api';

const Query: React.FC = () => {
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<QueryResponse | null>(null);
  const [knowledgeGraph, setKnowledgeGraph] = useState<KnowledgeGraphResponse | null>(null);
  const [activeTab, setActiveTab] = useState<'vector' | 'graph'>('vector');

  const handleVectorSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    try {
      const results = await apiService.query({ query: query.trim() });
      setSearchResults(results);
    } catch (error) {
      console.error('Failed to perform vector search:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleKnowledgeGraphSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    setLoading(true);
    try {
      const results = await apiService.getKnowledgeGraph(query.trim());
      setKnowledgeGraph(results);
    } catch (error) {
      console.error('Failed to get knowledge graph:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    if (activeTab === 'vector') {
      handleVectorSearch(e);
    } else {
      handleKnowledgeGraphSearch(e);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">Query</h1>
        <p className="mt-1 text-sm text-gray-500">
          Search through your documents using vector similarity and knowledge graph
        </p>
      </div>

      {/* Search Form */}
      <div className="bg-white shadow sm:rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <form onSubmit={handleSearch} className="space-y-4">
            <div>
              <label htmlFor="query" className="block text-sm font-medium text-gray-700">
                Search Query
              </label>
              <div className="mt-1 relative rounded-md shadow-sm">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <MagnifyingGlassIcon className="h-5 w-5 text-gray-400" />
                </div>
                <input
                  type="text"
                  id="query"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  className="focus:ring-indigo-500 focus:border-indigo-500 block w-full pl-10 pr-12 sm:text-sm border-gray-300 rounded-md"
                  placeholder="Enter your search query..."
                />
                <div className="absolute inset-y-0 right-0 flex items-center">
                  <button
                    type="submit"
                    disabled={loading || !query.trim()}
                    className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {loading ? (
                      <>
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                        Searching...
                      </>
                    ) : (
                      'Search'
                    )}
                  </button>
                </div>
              </div>
            </div>
          </form>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('vector')}
            className={`py-2 px-1 border-b-2 font-medium text-sm ${
              activeTab === 'vector'
                ? 'border-indigo-500 text-indigo-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            }`}
          >
            Vector Search
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

      {/* Results */}
      {activeTab === 'vector' && searchResults && (
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          <div className="px-4 py-5 sm:px-6">
            <h3 className="text-lg leading-6 font-medium text-gray-900">
              Vector Search Results
            </h3>
            <p className="mt-1 max-w-2xl text-sm text-gray-500">
              Found {searchResults.results.length} results
            </p>
          </div>
          <div className="border-t border-gray-200">
            {searchResults.results.length > 0 ? (
              <ul className="divide-y divide-gray-200">
                {searchResults.results.map((result, index) => (
                  <li key={index} className="px-4 py-4">
                    <div className="flex items-start space-x-3">
                      <div className="flex-shrink-0">
                        <DocumentTextIcon className="h-6 w-6 text-gray-400" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="text-sm text-gray-900">
                          <p className="whitespace-pre-wrap">{result.content}</p>
                        </div>
                        <div className="mt-2 flex items-center space-x-2 text-xs text-gray-500">
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            Score: {result.score.toFixed(3)}
                          </span>
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                            Doc ID: {result.document_id}
                          </span>
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
                            {result.title}
                          </span>
                        </div>
                        <div className="mt-1 text-xs text-gray-400 truncate">
                          {result.url}
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="px-4 py-8 text-center">
                <DocumentTextIcon className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">No results found</h3>
                <p className="mt-1 text-sm text-gray-500">
                  Try adjusting your search query to find more results.
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === 'graph' && knowledgeGraph && (
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          <div className="px-4 py-5 sm:px-6">
            <h3 className="text-lg leading-6 font-medium text-gray-900">
              Knowledge Graph Results
            </h3>
            <p className="mt-1 max-w-2xl text-sm text-gray-500">
              Found {knowledgeGraph.nodes?.length || 0} nodes and {knowledgeGraph.edges?.length || 0} edges
            </p>
          </div>
          <div className="border-t border-gray-200">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 p-6">
              {/* Nodes */}
              <div>
                <h4 className="text-md font-medium text-gray-900 mb-4">Nodes</h4>
                {(knowledgeGraph.nodes?.length || 0) > 0 ? (
                  <div className="space-y-3">
                    {knowledgeGraph.nodes.map((node) => (
                      <div key={node.id} className="bg-gray-50 rounded-lg p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <h5 className="text-sm font-medium text-gray-900">{node.name}</h5>
                            <p className="text-xs text-gray-500">Type: {node.type}</p>
                          </div>
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                            ID: {node.id}
                          </span>
                        </div>
                        {node.document_id && (
                          <div className="mt-2 space-y-1">
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                              {node.title || 'No Title'}
                            </span>
                            <p className="text-xs text-gray-400 truncate">Doc ID: {node.document_id}</p>
                            <p className="text-xs text-gray-400 truncate">{node.url}</p>
                          </div>
                        )}
                        {Object.keys(node.properties).length > 0 && (
                          <div className="mt-2">
                            <p className="text-xs text-gray-500">Properties:</p>
                            <pre className="text-xs text-gray-700 mt-1 bg-white p-2 rounded border">
                              {JSON.stringify(node.properties, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-gray-500">No nodes found</p>
                )}
              </div>

              {/* Edges */}
              <div>
                <h4 className="text-md font-medium text-gray-900 mb-4">Edges</h4>
                {(knowledgeGraph.edges?.length || 0) > 0 ? (
                  <div className="space-y-3">
                    {knowledgeGraph.edges.map((edge) => (
                      <div key={edge.id} className="bg-gray-50 rounded-lg p-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm font-medium text-gray-900">
                              {edge.source_id} → {edge.target_id}
                            </p>
                            <p className="text-xs text-gray-500">
                              Type: {edge.relationship_type}
                            </p>
                          </div>
                          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                            ID: {edge.id}
                          </span>
                        </div>
                        {Object.keys(edge.properties).length > 0 && (
                          <div className="mt-2">
                            <p className="text-xs text-gray-500">Properties:</p>
                            <pre className="text-xs text-gray-700 mt-1 bg-white p-2 rounded border">
                              {JSON.stringify(edge.properties, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-gray-500">No edges found</p>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* No Results State */}
      {!loading && !searchResults && !knowledgeGraph && (
        <div className="text-center py-12">
          <ShareIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No search performed</h3>
          <p className="mt-1 text-sm text-gray-500">
            Enter a query above to start searching through your documents.
          </p>
        </div>
      )}
    </div>
  );
};

export default Query; 