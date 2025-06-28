import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import DataSources from './pages/DataSources';
import Query from './pages/Query';
import DataDetail from './pages/DataDetail';
import MCPLogs from './pages/MCPLogs';
import './App.css';

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/" element={<DataSources />} />
          <Route path="/data-sources" element={<DataSources />} />
          <Route path="/data-detail/:id" element={<DataDetail />} />
          <Route path="/query" element={<Query />} />
          <Route path="/mcp-logs" element={<MCPLogs />} />
        </Routes>
      </Layout>
    </Router>
  );
}

export default App;
