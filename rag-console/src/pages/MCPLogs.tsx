import React, { useState, useEffect } from 'react';
import api from '../services/api';

interface McpLog {
  id: number;
  request_id: string;
  method: string;
  params: any;
  response: any;
  error: any;
  created_at: string;
}

const MCPLogs: React.FC = () => {
  const [logs, setLogs] = useState<McpLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchLogs = async () => {
      try {
        setLoading(true);
        const data = await api.getMCPLogs();
        setLogs(data.logs || []);
      } catch (err: any) {
        setError(err.message || 'Failed to fetch MCP logs');
      } finally {
        setLoading(false);
      }
    };

    fetchLogs();
    const interval = setInterval(fetchLogs, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  const renderJson = (data: any) => {
    if (data === null || data === undefined) {
      return <span className="text-gray-400">null</span>;
    }
    
    let text;
    try {
        // The data from Go backend (json.RawMessage) is already a JS object/array here if valid JSON.
        text = JSON.stringify(data, null, 2);
        if (text === '{}' || text === 'null') {
             return <span className="text-gray-400">null</span>;
        }
    } catch (e) {
        return <span className="text-red-400">Invalid JSON</span>
    }
    
    return <pre className="text-xs bg-gray-100 p-2 rounded whitespace-pre-wrap break-all">{text}</pre>;
  };
  
  if (loading && logs.length === 0) return <div className="text-center p-4">Loading...</div>;
  if (error) return <div className="text-center p-4 text-red-500">{error}</div>;

  return (
    <div className="p-4">
      <h1 className="text-2xl font-bold mb-4">MCP Logs</h1>
      <div className="overflow-x-auto bg-white rounded-lg shadow">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Request ID</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Method</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Params</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Response</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Error</th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created At</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {logs.length === 0 && !loading && (
                <tr>
                    <td colSpan={7} className="text-center py-4 text-gray-500">No logs found.</td>
                </tr>
            )}
            {logs.map((log) => (
              <tr key={log.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{log.id}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{log.request_id}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{log.method}</td>
                <td className="px-6 py-4">{renderJson(log.params)}</td>
                <td className="px-6 py-4">{renderJson(log.response)}</td>
                <td className="px-6 py-4">{renderJson(log.error)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{new Date(log.created_at).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default MCPLogs;

export {}; 