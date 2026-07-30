import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';

interface Metric {
  total_clientes: number;
  atendimentos_hoje: number;
  atendimentos_mes: number;
  ticket_medio: number;
  taxa_conversao: number;
  orcamentos_gerados: number;
  tempo_medio_espera: number;
  produtos_mais_vendidos: Array<{ nome: string; quantidade: number; total: number }>;
}

interface FilaStatus {
  em_atendimento: number;
  em_espera: number;
  limite: number;
}

const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<Metric | null>(null);
  const [fila, setFila] = useState<FilaStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [user, setUser] = useState<any>(null);
  const navigate = useNavigate();

  const fetchDashboardData = useCallback(async () => {
    try {
      const token = localStorage.getItem('token');
      if (!token) {
        navigate('/login');
        return;
      }

      const response = await fetch('/api/v1/dashboard', {
        headers: { 'Authorization': `Bearer ${token}` }
      });

      if (response.status === 401) {
        navigate('/login');
        return;
      }

      const data = await response.json();
      
      // Separar métricas e fila
      if (data.metrics) {
        setMetrics(data.metrics);
      } else {
        setMetrics(data);
      }
      
      if (data.fila) {
        setFila(data.fila);
      }
    } catch (error) {
      console.error('Erro ao carregar dashboard:', error);
    } finally {
      setLoading(false);
    }
  }, [navigate]);

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      navigate('/login');
      return;
    }

    const userData = localStorage.getItem('user');
    if (userData) {
      setUser(JSON.parse(userData));
    }

    fetchDashboardData();
  }, [fetchDashboardData, navigate]);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-center">
          <div className="text-4xl mb-4">⏳</div>
          <p className="text-gray-600">Carregando dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4 flex justify-between items-center">
          <div className="flex items-center gap-3">
            <span className="text-2xl">💊</span>
            <h1 className="text-xl font-bold text-gray-800">SaaS Atendimento</h1>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">
              👤 {user?.nome || 'Usuário'} ({user?.role || 'atendente'})
            </span>
            <button
              onClick={handleLogout}
              className="px-4 py-2 text-sm bg-red-500 text-white rounded-lg hover:bg-red-600 transition"
            >
              Sair
            </button>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Cards de métricas */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <MetricCard
            title="Clientes"
            value={metrics?.total_clientes || 0}
            icon="👥"
            color="bg-blue-500"
          />
          <MetricCard
            title="Atendimentos Hoje"
            value={metrics?.atendimentos_hoje || 0}
            icon="📅"
            color="bg-green-500"
          />
          <MetricCard
            title="Ticket Médio"
            value={`R$ ${metrics?.ticket_medio?.toFixed(2) || '0,00'}`}
            icon="💰"
            color="bg-purple-500"
          />
          <MetricCard
            title="Taxa Conversão"
            value={`${metrics?.taxa_conversao?.toFixed(1) || 0}%`}
            icon="📈"
            color="bg-orange-500"
          />
        </div>

        {/* Segunda linha */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <MetricCard
            title="Orçamentos Hoje"
            value={metrics?.orcamentos_gerados || 0}
            icon="📋"
            color="bg-teal-500"
          />
          <MetricCard
            title="Tempo Médio Espera"
            value={`${metrics?.tempo_medio_espera || 0}s`}
            icon="⏱️"
            color="bg-yellow-500"
          />
          <MetricCard
            title="Atendimentos Mês"
            value={metrics?.atendimentos_mes || 0}
            icon="📊"
            color="bg-pink-500"
          />
        </div>

        {/* Fila de atendimento */}
        {fila && (
          <div className="bg-white rounded-xl shadow-md p-6 mb-8">
            <h3 className="text-lg font-semibold text-gray-800 mb-4">📌 Fila de Atendimento</h3>
            <div className="grid grid-cols-3 gap-4">
              <div className="text-center p-4 bg-green-50 rounded-lg">
                <p className="text-2xl font-bold text-green-600">{fila.em_atendimento}</p>
                <p className="text-sm text-gray-600">Em atendimento</p>
              </div>
              <div className="text-center p-4 bg-yellow-50 rounded-lg">
                <p className="text-2xl font-bold text-yellow-600">{fila.em_espera}</p>
                <p className="text-sm text-gray-600">Aguardando</p>
              </div>
              <div className="text-center p-4 bg-blue-50 rounded-lg">
                <p className="text-2xl font-bold text-blue-600">{fila.limite}</p>
                <p className="text-sm text-gray-600">Limite máximo</p>
              </div>
            </div>
          </div>
        )}

        {/* Produtos mais vendidos */}
        <div className="bg-white rounded-xl shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-4">🏆 Produtos Mais Vendidos</h3>
          <div className="space-y-3">
            {metrics?.produtos_mais_vendidos?.map((produto, index) => (
              <div key={index} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center gap-3">
                  <span className="text-2xl">#{index + 1}</span>
                  <div>
                    <p className="font-medium text-gray-800">{produto.nome}</p>
                    <p className="text-sm text-gray-500">{produto.quantidade} vendidos</p>
                  </div>
                </div>
                <span className="font-semibold text-green-600">R$ {produto.total.toFixed(2)}</span>
              </div>
            ))}
            {(!metrics?.produtos_mais_vendidos || metrics.produtos_mais_vendidos.length === 0) && (
              <p className="text-gray-400 text-center py-8">Nenhum produto vendido ainda</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

// Componente de Card
const MetricCard: React.FC<{ title: string; value: string | number; icon: string; color: string }> = ({ 
  title, value, icon, color 
}) => (
  <div className="bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition">
    <div className="flex items-center gap-4">
      <div className={`w-12 h-12 ${color} rounded-xl flex items-center justify-center text-white text-2xl`}>
        {icon}
      </div>
      <div>
        <p className="text-sm text-gray-500">{title}</p>
        <p className="text-2xl font-bold text-gray-800">{value}</p>
      </div>
    </div>
  </div>
);

export default Dashboard;