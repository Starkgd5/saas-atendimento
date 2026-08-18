import React, { useState, useEffect } from 'react';
import { MetricCard } from '../components/ui/MetricCard';
import { VendasHojeChart } from '../components/charts/VendasHojeChart';
import { EstoqueAlerta } from '../components/estoque/EstoqueAlerta';
import { UltimasVendas } from '../components/vendas/UltimasVendas';

interface DashboardData {
  vendas_hoje: number;
  vendas_mes: number;
  ticket_medio: number;
  produtos_baixo_estoque: number;
  produtos_vencendo: number;
  receitas_pendentes: number;
  faturamento_diario: number;
  faturamento_mensal: number;
  comparativo_mes_anterior: number;
}

export const DashboardFarmacia: React.FC = () => {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Fetch dados do dashboard
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      const response = await fetch('/api/v1/dashboard/farmacia', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const result = await response.json();
      setData(result);
    } catch (error) {
      console.error('Erro ao carregar dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div>Carregando...</div>;

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">📊 Dashboard Farmácia</h1>
        <div className="flex gap-2">
          <select className="border rounded p-2">
            <option>F01 - Centro</option>
            <option>F02 - Norte</option>
            <option>F14 - Oeste 2</option>
          </select>
          <button 
            onClick={fetchDashboardData}
            className="bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"
          >
            Atualizar
          </button>
        </div>
      </div>

      {/* Métricas Principais */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <MetricCard 
          title="Faturamento Hoje" 
          value={`R$ ${data?.faturamento_diario?.toFixed(2) || '0,00'}`}
          icon="💰"
          color="bg-green-500"
        />
        <MetricCard 
          title="Vendas Hoje" 
          value={data?.vendas_hoje || 0}
          icon="🛒"
          color="bg-blue-500"
        />
        <MetricCard 
          title="Ticket Médio" 
          value={`R$ ${data?.ticket_medio?.toFixed(2) || '0,00'}`}
          icon="🎫"
          color="bg-purple-500"
        />
        <MetricCard 
          title="Receitas Pendentes" 
          value={data?.receitas_pendentes || 0}
          icon="📋"
          color="bg-yellow-500"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Gráfico de Vendas */}
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="font-semibold mb-4">📈 Vendas Hoje</h3>
          <VendasHojeChart />
        </div>

        {/* Alertas de Estoque */}
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="font-semibold mb-4">⚠️ Alertas de Estoque</h3>
          <EstoqueAlerta 
            produtosBaixoEstoque={data?.produtos_baixo_estoque || 0}
            produtosVencendo={data?.produtos_vencendo || 0}
          />
        </div>
      </div>

      {/* Últimas Vendas */}
      <div className="mt-6 bg-white rounded-lg shadow p-4">
        <h3 className="font-semibold mb-4">🛒 Últimas Vendas</h3>
        <UltimasVendas />
      </div>
    </div>
  );
};