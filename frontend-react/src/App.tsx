import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import PrivateRoute from './components/PrivateRoute';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import DashboardFarmacia from './pages/dashboard/DashboardFarmacia';
import Atendimento from './pages/Atendimento';
import UserManagement from './pages/UserManagement';
import Produtos from './pages/Produtos';
import Orcamentos from './pages/Orcamentos';
import Reclamacoes from './pages/Reclamacoes';
import ControleEstoque from './pages/ControleEstoque';
// import EntradaEstoque from './pages/EntradaEstoque';
import PDV from './pages/PDV';
// import Vendas from './pages/Vendas';
// import Receitas from './pages/Receitas';
// import Fornecedores from './pages/Fornecedores';
// import Compras from './pagesCompras';
// import Financeiro from './pages/Financeiro';
import RelatoriosFarmacia from './pages/RelatoriosFarmacia';
// import Configuracoes from './pages/Configuracoes';
require('./App.css');

function App() {
  return (
    <Router>
      <Routes>
        {/* Página de Login - Pública */}
        <Route path="/login" element={<Login />} />

        {/* Rotas Protegidas com Layout */}
        <Route
          path="/"
          element={
            <PrivateRoute>
              <Layout>
                <Navigate to="/dashboard/farmacia" />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Dashboard */}
        <Route
          path="/dashboard"
          element={
            <PrivateRoute>
              <Layout>
                <Dashboard />
              </Layout>
            </PrivateRoute>
          }
        />
        <Route
          path="/dashboard/farmacia"
          element={
            <PrivateRoute>
              <Layout>
                <DashboardFarmacia />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Atendimento */}
        <Route
          path="/atendimento"
          element={
            <PrivateRoute>
              <Layout>
                <Atendimento />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Produtos */}
        <Route
          path="/produtos"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente', 'atendente']}>
              <Layout>
                <Produtos />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Estoque */}
        <Route
          path="/estoque"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <ControleEstoque />
              </Layout>
            </PrivateRoute>
          }
        />
        {/* <Route
          path="/estoque/entrada"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <EntradaEstoque />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Vendas / PDV */}
        <Route
          path="/pdv"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente', 'atendente']}>
              <Layout>
                <PDV />
              </Layout>
            </PrivateRoute>
          }
        />
        {/* <Route
          path="/vendas"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente', 'atendente']}>
              <Layout>
                <Vendas />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Orçamentos */}
        <Route
          path="/orcamentos"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <Orcamentos />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Receitas Médicas */}
        {/* <Route
          path="/receitas"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente', 'atendente']}>
              <Layout>
                <Receitas />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Reclamações */}
        <Route
          path="/reclamacoes"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <Reclamacoes />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Fornecedores */}
        {/* <Route
          path="/fornecedores"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <Fornecedores />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Compras */}
        {/* <Route
          path="/compras"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <Compras />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Financeiro */}
        {/* <Route
          path="/financeiro"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <Financeiro />
              </Layout>
            </PrivateRoute>
          }
        /> */}

        {/* Relatórios */}
        <Route
          path="/relatorios"
          element={
            <PrivateRoute allowedRoles={['admin', 'gerente']}>
              <Layout>
                <RelatoriosFarmacia />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Usuários - Apenas Admin */}
        <Route
          path="/usuarios"
          element={
            <PrivateRoute requiredRole="admin">
              <Layout>
                <UserManagement />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* Configurações - Apenas Admin */}
        {/* <Route
          path="/configuracoes"
          element={
            <PrivateRoute requiredRole="admin">
              <Layout>
                <Configuracoes />
              </Layout>
            </PrivateRoute>
          }
        />

        {/* 404 - Redirecionar para Dashboard */}
        <Route path="*" element={<Navigate to="/dashboard/farmacia" />} />
      </Routes>
    </Router>
  );
}

export default App;