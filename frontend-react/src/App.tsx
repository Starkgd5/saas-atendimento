import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import PrivateRoute from './components/PrivateRoute';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Atendimento from './pages/Atendimento';
import UserManagement from './pages/UserManagement';
import Produtos from './pages/Produtos';
import Orcamentos from './pages/Orcamentos';
import Reclamacoes from './pages/Reclamacoes';
import './App.css';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route path="/" element={
          <PrivateRoute>
            <Layout>
              <Navigate to="/dashboard" />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/dashboard" element={
          <PrivateRoute>
            <Layout>
              <Dashboard />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/atendimento" element={
          <PrivateRoute>
            <Layout>
              <Atendimento />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/produtos" element={
          <PrivateRoute allowedRoles={['admin', 'gerente']}>
            <Layout>
              <Produtos />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/orcamentos" element={
          <PrivateRoute allowedRoles={['admin', 'gerente']}>
            <Layout>
              <Orcamentos />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/reclamacoes" element={
          <PrivateRoute allowedRoles={['admin', 'gerente']}>
            <Layout>
              <Reclamacoes />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="/usuarios" element={
          <PrivateRoute requiredRole="admin">
            <Layout>
              <UserManagement />
            </Layout>
          </PrivateRoute>
        } />
        
        <Route path="*" element={<Navigate to="/dashboard" />} />
      </Routes>
    </Router>
  );
}

export default App;