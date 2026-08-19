import React, { useState, useEffect } from 'react';
import MetricCard from '../../components/ui/MetricCard';
import VendasHojeChart from '../../components/charts/VendasHojeChart';
import EstoqueAlerta from '../../components/estoque/EstoqueAlerta';
import UltimasVendas from '../../components/vendas/UltimasVendas';
import { useApi } from '../../hooks/useApi';
import { useAuth } from '../../hooks/useAuth';
import { useToast } from '../../hooks/useToast';
import FormatterService from '../../services/formatter.service';

interface DashboardData {
  vendas_hoje: number;
  faturamento_hoje: number;
  vendas_mes: number;
  faturamento_mes: number;
  ticket_medio: number;
  produtos_baixo_estoque: number;
  produtos_vencendo: number;
  receitas_pendentes: number;
  comparativo_mes: number;
  ultimas_vendas: Array<{
    id: number;
    numero_venda: string;
    cliente: string;
    total: number;
    data: string;
  }>;
  timestamp: string;
}

export const DashboardFarmacia: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const { get } = useApi();
  const { user } = useAuth();
  const { success, error: showError } = useToast();

  useEffect(() => {
    fetchDashboard();
  }, []);

  const fetchDashboard = async () => {
    try {
      const response = await get('/dashboard/farmacia');
      setData(response);
    } catch (error: any) {
      showError(error.message || 'Erro ao carregar dashboard');
    } finally {
      setLoading(false);
    }
  };

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
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold">📊 Dashboard Farmácia</h1>
          <p className="text-gray-500 text-sm mt-1">
            Bem-vindo, {user?.nome}! Aqui estão as métricas da sua farmácia.
          </p>
          {data?.timestamp && (
            <p className="text-xs text-gray-400 mt-1">
              Última atualização: {new Date(data.timestamp).toLocaleString('pt-BR')}
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <select className="border rounded p-2 text-sm">
            <option>F01 - Centro</option>
            <option>F02 - Norte</option>
            <option>F14 - Oeste 2</option>
          </select>
          <button 
            onClick={fetchDashboard}
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600 transition text-sm"
          >
            🔄 Atualizar
          </button>
        </div>
      </div>

      {/* Métricas Principais */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <MetricCard 
          title="Faturamento Hoje" 
          value={formatCurrency(data?.faturamento_hoje || 0)}
          icon="💰"
          color="bg-green-500"
          change={data?.comparativo_mes ? `${data.comparativo_mes > 0 ? '+' : ''}${data.comparativo_mes.toFixed(1)}% vs mês anterior` : undefined}
        />
        <MetricCard 
          title="Vendas Hoje" 
          value={formatNumber(data?.vendas_hoje || 0)}
          icon="🛒"
          color="bg-blue-500"
          subtitle={`${formatNumber(data?.vendas_mes || 0)} no mês`}
        />
        <MetricCard 
          title="Ticket Médio" 
          value={formatCurrency(data?.ticket_medio || 0)}
          icon="🎫"
          color="bg-purple-500"
        />
        <MetricCard 
          title="Receitas Pendentes" 
          value={formatNumber(data?.receitas_pendentes || 0)}
          icon="📋"
          color="bg-yellow-500"
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Gráfico de Vendas */}
        <div className="bg-white rounded-xl shadow-md p-4">
          <h3 className="font-semibold mb-4">📈 Vendas Hoje</h3>
          <VendasHojeChart />
        </div>

        {/* Alertas de Estoque */}
        <div className="bg-white rounded-xl shadow-md p-4">
          <h3 className="font-semibold mb-4">⚠️ Alertas de Estoque</h3>
          <EstoqueAlerta 
            produtosBaixoEstoque={data?.produtos_baixo_estoque || 0}
            produtosVencendo={data?.produtos_vencendo || 0}
          />
        </div>
      </div>

      {/* Últimas Vendas */}
      <div className="bg-white rounded-xl shadow-md p-4">
        <h3 className="font-semibold mb-4">🛒 Últimas Vendas</h3>
        <UltimasVendas />
      </div>
    </div>
  );
};

export default DashboardFarmacia;