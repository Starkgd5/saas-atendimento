import React, { useState } from 'react';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';

export const RelatoriosFarmacia: React.FC = () => {
  const [periodo, setPeriodo] = useState('mensal');
  const [loading, setLoading] = useState(false);

  const relatorios = [
    {
      id: 'vendas',
      nome: '📊 Relatório de Vendas',
      descricao: 'Análise detalhada de vendas por período',
      icon: '📊'
    },
    {
      id: 'estoque',
      nome: '📦 Relatório de Estoque',
      descricao: 'Produtos em estoque, validades e giro',
      icon: '📦'
    },
    {
      id: 'medicamentos_controlados',
      nome: '💊 Medicamentos Controlados',
      descricao: 'Balanço de medicamentos sujeitos a controle especial',
      icon: '💊'
    },
    {
      id: 'receitas',
      nome: '📋 Receitas Validadas',
      descricao: 'Histórico de receitas médicas validadas',
      icon: '📋'
    },
    {
      id: 'financeiro',
      nome: '💰 Relatório Financeiro',
      descricao: 'Contas a receber, pagar e fluxo de caixa',
      icon: '💰'
    },
    {
      id: 'clientes',
      nome: '👥 Relatório de Clientes',
      descricao: 'Perfil de clientes e histórico de compras',
      icon: '👥'
    }
  ];

  const handleExport = (tipo: string) => {
    setLoading(true);
    // Simular exportação
    setTimeout(() => {
      setLoading(false);
      alert(`Relatório ${tipo} exportado com sucesso!`);
    }, 2000);
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">📈 Relatórios</h1>
        <div className="flex gap-2">
          <select 
            value={periodo} 
            onChange={(e) => setPeriodo(e.target.value)}
            className="border rounded p-2"
          >
            <option value="diario">Diário</option>
            <option value="semanal">Semanal</option>
            <option value="mensal">Mensal</option>
            <option value="trimestral">Trimestral</option>
            <option value="anual">Anual</option>
          </select>
          <Button variant="primary" icon="📥">
            Exportar Todos
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {relatorios.map((relatorio) => (
          <Card key={relatorio.id} className="hover:shadow-lg transition">
            <div className="p-6">
              <div className="text-4xl mb-4">{relatorio.icon}</div>
              <h3 className="text-lg font-semibold text-gray-800">{relatorio.nome}</h3>
              <p className="text-sm text-gray-500 mt-2">{relatorio.descricao}</p>
              <div className="mt-4 flex gap-2">
                <Button 
                  variant="primary" 
                  size="sm"
                  onClick={() => handleExport(relatorio.id)}
                  loading={loading}
                >
                  📄 Gerar
                </Button>
                <Button variant="outline" size="sm">
                  👁️ Visualizar
                </Button>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Relatórios Rápidos */}
      <div className="mt-8 bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">⚡ Relatórios Rápidos</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <button className="p-4 bg-blue-50 rounded-lg hover:bg-blue-100 transition text-center">
            <div className="text-2xl mb-2">📊</div>
            <span className="text-sm">Vendas do Dia</span>
          </button>
          <button className="p-4 bg-red-50 rounded-lg hover:bg-red-100 transition text-center">
            <div className="text-2xl mb-2">⚠️</div>
            <span className="text-sm">Produtos Vencendo</span>
          </button>
          <button className="p-4 bg-green-50 rounded-lg hover:bg-green-100 transition text-center">
            <div className="text-2xl mb-2">💰</div>
            <span className="text-sm">Faturamento Mês</span>
          </button>
          <button className="p-4 bg-yellow-50 rounded-lg hover:bg-yellow-100 transition text-center">
            <div className="text-2xl mb-2">📋</div>
            <span className="text-sm">Receitas Pendentes</span>
          </button>
        </div>
      </div>
    </div>
  );
};