import React, { useState, useEffect } from 'react';
import { 
  PlusIcon, 
  MagnifyingGlassIcon,
  CheckCircleIcon,
  XCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
  TrashIcon,
  ArrowPathIcon as ReindexIcon,
  XMarkIcon
} from '@heroicons/react/24/outline';
import apiService, { URLQueueItem, ProcessDocumentRequest } from '../services/api';
import { Link } from 'react-router-dom';

interface Message {
  id: string;
  type: 'success' | 'error' | 'info';
  text: string;
}

const DataSources: React.FC = () => {
  const [urlQueue, setUrlQueue] = useState<URLQueueItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [showSubmitForm, setShowSubmitForm] = useState(false);
  const [messages, setMessages] = useState<Message[]>([]);
  const [submitForm, setSubmitForm] = useState({
    url: '',
    title: '',
    content: '',
  });

  useEffect(() => {
    loadURLQueue();
  }, []);

  const addMessage = (type: 'success' | 'error' | 'info', text: string) => {
    const id = Date.now().toString();
    const newMessage = { id, type, text };
    setMessages(prev => [...prev, newMessage]);
    
    // Auto remove message after 5 seconds
    setTimeout(() => {
      setMessages(prev => prev.filter(msg => msg.id !== id));
    }, 5000);
  };

  const removeMessage = (id: string) => {
    setMessages(prev => prev.filter(msg => msg.id !== id));
  };

  const loadURLQueue = async () => {
    setLoading(true);
    try {
      const data = await apiService.getURLQueue();
      setUrlQueue(data || []);
    } catch (error) {
      console.error('Failed to load URL queue:', error);
      setUrlQueue([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    
    try {
      const request: ProcessDocumentRequest = {
        url: submitForm.url,
        ...(submitForm.title && { title: submitForm.title }),
        ...(submitForm.content && { content: submitForm.content }),
      };
      
      await apiService.processDocument(request);
      setShowSubmitForm(false);
      setSubmitForm({ url: '', title: '', content: '' });
      
      // Show success message and refresh data
      addMessage('success', 'URL submitted successfully!');
      await loadURLQueue();
    } catch (error) {
      console.error('Failed to submit document:', error);
      addMessage('error', 'Failed to submit URL. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!window.confirm('Are you sure you want to delete this URL? This action cannot be undone.')) {
      return;
    }

    setLoading(true);
    try {
      await apiService.deleteURL(id);
      addMessage('success', 'URL deleted successfully!');
      await loadURLQueue();
    } catch (error) {
      console.error('Failed to delete URL:', error);
      addMessage('error', 'Failed to delete URL. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleReindex = async (id: number) => {
    setLoading(true);
    try {
      await apiService.reindexURL(id);
      addMessage('success', 'URL reindexed successfully!');
      await loadURLQueue();
    } catch (error) {
      console.error('Failed to reindex URL:', error);
      addMessage('error', 'Failed to reindex URL. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircleIcon className="h-5 w-5 text-green-500" />;
      case 'failed':
        return <XCircleIcon className="h-5 w-5 text-red-500" />;
      case 'processing':
        return <ClockIcon className="h-5 w-5 text-yellow-500" />;
      default:
        return <ClockIcon className="h-5 w-5 text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      case 'processing':
        return 'bg-yellow-100 text-yellow-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const filteredQueue = (urlQueue || []).filter(item =>
    item.url.toLowerCase().includes(searchTerm.toLowerCase())
  );

  console.log('filteredQueue', filteredQueue);

  return (
    <div className="space-y-6 relative">
      {/* Messages */}
      {messages.length > 0 && (
        <div className="absolute top-4 left-1/2 transform -translate-x-1/2 z-50 space-y-2 pointer-events-none">
          {messages.map((message) => (
            <div
              key={message.id}
              className={`w-96 bg-white shadow-lg rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden ${
                message.type === 'success' ? 'ring-green-500' : 
                message.type === 'error' ? 'ring-red-500' : 'ring-blue-500'
              }`}
            >
              <div className="px-6 py-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <div className="flex-shrink-0">
                      {message.type === 'success' ? (
                        <CheckCircleIcon className="h-5 w-5 text-green-400" />
                      ) : message.type === 'error' ? (
                        <XCircleIcon className="h-5 w-5 text-red-400" />
                      ) : (
                        <ExclamationTriangleIcon className="h-5 w-5 text-blue-400" />
                      )}
                    </div>
                    <div className="ml-3">
                      <p className={`text-sm font-medium ${
                        message.type === 'success' ? 'text-green-800' : 
                        message.type === 'error' ? 'text-red-800' : 'text-blue-800'
                      }`}>
                        {message.text}
                      </p>
                    </div>
                  </div>
                  <div className="flex-shrink-0">
                    <button
                      className={`bg-white rounded-md inline-flex text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 ${
                        message.type === 'success' ? 'focus:ring-green-500' : 
                        message.type === 'error' ? 'focus:ring-red-500' : 'focus:ring-blue-500'
                      }`}
                      onClick={() => removeMessage(message.id)}
                    >
                      <span className="sr-only">Close</span>
                      <XMarkIcon className="h-5 w-5" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Header */}
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-semibold text-gray-900">Data Sources</h1>
        <div className="flex space-x-3">
          <button
            onClick={loadURLQueue}
            disabled={loading}
            className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md shadow-sm text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
          >
            <ArrowPathIcon className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
          <button
            onClick={() => setShowSubmitForm(true)}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
          >
            <PlusIcon className="h-4 w-4 mr-2" />
            Add URL
          </button>
        </div>
      </div>

      {/* Submit Form Modal */}
      {showSubmitForm && (
        <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
          <div className="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
            <div className="mt-3">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Submit URL</h3>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">URL *</label>
                  <input
                    type="url"
                    required
                    value={submitForm.url}
                    onChange={(e) => setSubmitForm({ ...submitForm, url: e.target.value })}
                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="https://example.com"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Title (optional)</label>
                  <input
                    type="text"
                    value={submitForm.title}
                    onChange={(e) => setSubmitForm({ ...submitForm, title: e.target.value })}
                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="Document title"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">Content (optional)</label>
                  <textarea
                    value={submitForm.content}
                    onChange={(e) => setSubmitForm({ ...submitForm, content: e.target.value })}
                    rows={4}
                    className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-indigo-500 focus:border-indigo-500"
                    placeholder="Document content (if not provided, will be fetched from URL)"
                  />
                </div>
                <div className="flex justify-end space-x-3">
                  <button
                    type="button"
                    onClick={() => setShowSubmitForm(false)}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded-md hover:bg-gray-200"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={loading}
                    className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {loading ? 'Submitting...' : 'Submit'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Search */}
      <div className="relative">
        <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
          <MagnifyingGlassIcon className="h-5 w-5 text-gray-400" />
        </div>
        <input
          type="text"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
          placeholder="Search URLs..."
        />
      </div>

      {/* URL Queue Table */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <div className="px-4 py-5 sm:px-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900">URL Queue</h3>
          <p className="mt-1 max-w-2xl text-sm text-gray-500">
            Manage and monitor URL processing status
          </p>
        </div>
        
        {loading ? (
          <div className="px-4 py-8 text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto"></div>
            <p className="mt-2 text-sm text-gray-500">Loading...</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    URL
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Updated
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredQueue.map((item) => (
                  <tr key={item.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        {getStatusIcon(item.status)}
                        <span className={`ml-2 inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusColor(item.status)}`}>
                          {item.status}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <div className="text-sm text-gray-900">
                        <Link 
                          to={`/data-detail/${item.document_id || item.id}`}
                          className="text-indigo-600 hover:text-indigo-900 underline"
                        >
                          {item.url}
                        </Link>
                      </div>
                      {item.error && (
                        <div className="text-sm text-red-600 mt-1 flex items-center">
                          <ExclamationTriangleIcon className="h-4 w-4 mr-1" />
                          {item.error}
                        </div>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(item.created_at).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(item.updated_at).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <div className="flex space-x-2">
                        <button
                          onClick={() => handleDelete(item.id)}
                          disabled={loading}
                          className="inline-flex items-center px-2 py-1 border border-transparent text-xs font-medium rounded text-red-700 bg-red-100 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 disabled:opacity-50"
                          title="Delete URL"
                        >
                          <TrashIcon className="h-3 w-3 mr-1" />
                          Delete
                        </button>
                        <button
                          onClick={() => handleReindex(item.id)}
                          disabled={loading}
                          className="inline-flex items-center px-2 py-1 border border-transparent text-xs font-medium rounded text-indigo-700 bg-indigo-100 hover:bg-indigo-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                          title="Reindex URL"
                        >
                          <ReindexIcon className="h-3 w-3 mr-1" />
                          Reindex
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        
        {!loading && filteredQueue.length === 0 && (
          <div className="px-4 py-8 text-center">
            <p className="text-sm text-gray-500">No URLs found</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default DataSources; 