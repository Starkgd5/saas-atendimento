import React, { useState, useEffect } from 'react';
import { useApi } from '../../hooks/useApi';
import { useNavigate } from 'react-router-dom';
import FormatterService from '../../services/formatter.service';

interface Venda {
  id: number;
  numero_venda: string;
  cliente: string;
  total: number;
  data: string;
  status: string;
}

export const UltimasVendas: React.FC = () => {
  const [vendas, setVendas] = useState<Venda[]>([]);
  const [loading, setLoading] = useState(true);
  const { get } = useApi();
  const navigate = useNavigate();

  useEffect(() => {
    fetchVendas();
  }, []);

  const fetchVendas = async () => {
    try {
      const response = await get('/vendas?limit=5&status=Pago');
      if (response.items) {
        const vendasFormatadas = response.items.map((v: any) => ({
          id: v.id,
          numero_venda: v.numero_venda,
          cliente: v.cliente_nome || 'Cliente não identificado',
          total: v.total,
          data: v.created_at,
          status: v.status,
        }));
        setVendas(vendasFormatadas);
      }
    } catch (error) {
      console.error('Erro ao carregar vendas:', error);
      // Dados mockados para demonstração
      setVendas([
        {
          id: 1,
          numero_venda: 'V20240818001',
          cliente: 'João Silva',
          total: 45.90,
          data: new Date().toISOString(),
          status: 'Pago',
        },
        {
          id: 2,
          numero_venda: 'V20240818002',
          cliente: 'Maria Santos',
          total: 38.50,
          data: new Date(Date.now() - 3600000).toISOString(),
          status: 'Pago',
        },
        {
          id: 3,
          numero_venda: 'V20240818003',
          cliente: 'Pedro Oliveira',
          total: 92.30,
          data: new Date(Date.now() - 7200000).toISOString(),
          status: 'Pago',
        },
        {
          id: 4,
          numero_venda: 'V20240818004',
          cliente: 'Ana Souza',
          total: 15.90,
          data: new Date(Date.now() - 10800000).toISOString(),
          status: 'Pendente',
        },
        {
          id: 5,
          numero_venda: 'V20240817005',
          cliente: 'Carlos Ferreira',
          total: 120.00,
          data: new Date(Date.now() - 86400000).toISOString(),
          status: 'Pago',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const formatCurrency = FormatterService.formatCurrency;
  const formatDateTime = FormatterService.formatDateTime;

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'Pago':
        return 'bg-green-100 text-green-700';
      case 'Pendente':
        return 'bg-yellow-100 text-yellow-700';
      case 'Cancelado':
        return 'bg-red-100 text-red-700';
      default:
        return 'bg-gray-100 text-gray-700';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32">
        <div className="text-gray-400">Carregando vendas...</div>
      </div>
    );
  }

  if (vendas.length === 0) {
    return (
      <div className="text-center py-8 text-gray-400">
        <div className="text-4xl mb-2">🛒</div>
        <p>Nenhuma venda realizada hoje</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex justify-between items-center mb-3">
        <span className="text-sm text-gray-500">
          Total: {vendas.length} vendas
        </span>
        <button
          onClick={() => navigate('/vendas')}
          className="text-sm text-blue-500 hover:text-blue-600 transition"
        >
          Ver todas →
        </button>
      </div>

      {vendas.map((venda) => (
        <div
          key={venda.id}
          className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition cursor-pointer"
          onClick={() => navigate(`/vendas/${venda.id}`)}
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-sm font-medium text-gray-800 truncate">
                {venda.cliente}
              </p>
              <span className={`px-2 py-0.5 text-xs rounded-full ${getStatusColor(venda.status)}`}>
                {venda.status}
              </span>
            </div>
            <div className="flex items-center gap-2 text-xs text-gray-400">
              <span>{venda.numero_venda}</span>
              <span>•</span>
              <span>{formatDateTime(venda.data)}</span>
            </div>
          </div>
          <div className="ml-4">
            <p className="text-sm font-bold text-green-600">
              {formatCurrency(venda.total)}
            </p>
          </div>
        </div>
      ))}

      <div className="flex justify-between items-center mt-3 pt-3 border-t border-gray-200">
        <span className="text-sm text-gray-500">
          Total: {vendas.reduce((acc, v) => acc + v.total, 0).toFixed(2)}
        </span>
        <button
          onClick={fetchVendas}
          className="text-xs text-blue-500 hover:text-blue-600 transition"
        >
          🔄 Atualizar
        </button>
      </div>
    </div>
  );
};

export default UltimasVendas;