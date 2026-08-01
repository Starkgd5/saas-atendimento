import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Card from '../components/ui/Card';
import Button from '../components/ui/Button';
import { useApi } from '../hooks/useApi';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';

interface Metric {
  total_clientes: number;
  atendimentos_hoje: number;
  atendimentos_mes: number;
  ticket_medio: number;
  taxa_conversao: number;
  orcamentos_gerados: number;
  tempo_medio_espera: number;
  tempo_medio_atendimento: number;
  total_finalizados: number;
  abandonos: number;
  taxa_abandono: number;
  reclamacoes_pendentes: number;
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
  const { user } = useAuth();
  const { get } = useApi();
  const { success, error: showError } = useToast();
  const navigate = useNavigate();

  const fetchDashboardData = useCallback(async () => {
    try {
      const [dashboardData, filaData] = await Promise.all([
        get('/dashboard'),
        get('/fila/status')
      ]);

      if (dashboardData.metrics) {
        setMetrics(dashboardData.metrics);
      } else {
        setMetrics(dashboardData);
      }

      if (dashboardData.fila) {
        setFila(dashboardData.fila);
      } else if (filaData) {
        setFila(filaData);
      }
    } catch (error: any) {
      showError(error.message || 'Erro ao carregar dashboard');
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    fetchDashboardData();

    const interval = setInterval(fetchDashboardData, 30000);
    return () => clearInterval(interval);
  }, [fetchDashboardData]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando dashboard...</p>
        </div>
      </div>
    );
  }

  const formatCurrency = FormatterService.formatCurrency;
  const formatNumber = FormatterService.formatNumber;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">📊 Dashboard</h1>
          <p className="text-gray-500 text-sm mt-1">
            Bem-vindo, {user?.nome}! Aqui estão as métricas do sistema.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={fetchDashboardData}
          icon="🔄"
        >
          Atualizar
        </Button>
      </div>

      {/* Cards de métricas */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <MetricCard
          title="Clientes"
          value={formatNumber(metrics?.total_clientes || 0)}
          icon="👥"
          color="bg-blue-500"
          change="+12% este mês"
        />
        <MetricCard
          title="Atendimentos Hoje"
          value={formatNumber(metrics?.atendimentos_hoje || 0)}
          icon="📅"
          color="bg-green-500"
          change={`${formatNumber(metrics?.atendimentos_mes || 0)} no mês`}
        />
        <MetricCard
          title="Ticket Médio"
          value={formatCurrency(metrics?.ticket_medio || 0)}
          icon="💰"
          color="bg-purple-500"
        />
        <MetricCard
          title="Taxa Conversão"
          value={`${formatNumber(metrics?.taxa_conversao || 0, 1)}%`}
          icon="📈"
          color="bg-orange-500"
        />
      </div>

      {/* Segunda linha */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <MetricCard
          title="Orçamentos Hoje"
          value={formatNumber(metrics?.orcamentos_gerados || 0)}
          icon="📋"
          color="bg-teal-500"
        />
        <MetricCard
          title="Tempo Médio Espera"
          value={`${formatNumber(metrics?.tempo_medio_espera || 0)}s`}
          icon="⏱️"
          color="bg-yellow-500"
        />
        <MetricCard
          title="Tempo Médio Atendimento"
          value={`${formatNumber(Math.round(metrics?.tempo_medio_atendimento || 0))}s`}
          icon="⏳"
          color="bg-pink-500"
        />
        <MetricCard
          title="Reclamações Pendentes"
          value={formatNumber(metrics?.reclamacoes_pendentes || 0)}
          icon="📋"
          color="bg-red-500"
        />
      </div>

      {/* Fila de atendimento */}
      {fila && (
        <Card
          title="📌 Fila de Atendimento"
          className="mb-8"
        >
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
        </Card>
      )}

      {/* Produtos mais vendidos */}
      <Card title="🏆 Produtos Mais Vendidos">
        <div className="space-y-3">
          {metrics?.produtos_mais_vendidos?.map((produto, index) => (
            <div key={index} className="flex justify-between items-center p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition">
              <div className="flex items-center gap-3">
                <span className={`text-xl font-bold w-8 ${index === 0 ? 'text-yellow-500' : index === 1 ? 'text-gray-400' : index === 2 ? 'text-amber-600' : 'text-gray-400'}`}>
                  #{index + 1}
                </span>
                <div>
                  <p className="font-medium text-gray-800">{produto.nome}</p>
                  <p className="text-sm text-gray-500">{formatNumber(produto.quantidade)} vendidos</p>
                </div>
              </div>
              <span className="font-semibold text-green-600">{formatCurrency(produto.total)}</span>
            </div>
          ))}
          {(!metrics?.produtos_mais_vendidos || metrics.produtos_mais_vendidos.length === 0) && (
            <p className="text-gray-400 text-center py-8">Nenhum produto vendido ainda</p>
          )}
        </div>
      </Card>
    </div>
  );
};

// Componente de Card de Métrica
const MetricCard: React.FC<{ 
  title: string; 
  value: string | number; 
  icon: string; 
  color: string;
  change?: string;
}> = ({ title, value, icon, color, change }) => (
  <div className="bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition-shadow duration-200">
    <div className="flex items-center gap-4">
      <div className={`w-12 h-12 ${color} rounded-xl flex items-center justify-center text-white text-2xl flex-shrink-0`}>
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-500 font-medium">{title}</p>
        <p className="text-2xl font-bold text-gray-800">{value}</p>
        {change && (
          <p className="text-xs text-gray-400 mt-1">{change}</p>
        )}
      </div>
    </div>
  </div>
);

export default Dashboard;