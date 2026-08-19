import React, { useState, useEffect } from 'react';
import { useApi } from '../../hooks/useApi';

interface VendaHora {
  hora: number;
  total: number;
  finalizados: number;
}

export const VendasHojeChart: React.FC = () => {
  const [data, setData] = useState<VendaHora[]>([]);
  const [loading, setLoading] = useState(true);
  const { get } = useApi();

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const response = await get('/dashboard/metricas');
      if (response.metricas_por_hora) {
        setData(response.metricas_por_hora);
      }
    } catch (error) {
      console.error('Erro ao carregar dados do gráfico:', error);
      // Dados mockados para demonstração
      setData([
        { hora: 8, total: 2, finalizados: 2 },
        { hora: 9, total: 5, finalizados: 4 },
        { hora: 10, total: 8, finalizados: 7 },
        { hora: 11, total: 6, finalizados: 5 },
        { hora: 12, total: 3, finalizados: 3 },
        { hora: 13, total: 4, finalizados: 3 },
        { hora: 14, total: 7, finalizados: 6 },
        { hora: 15, total: 9, finalizados: 8 },
        { hora: 16, total: 6, finalizados: 5 },
        { hora: 17, total: 4, finalizados: 4 },
        { hora: 18, total: 2, finalizados: 2 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-400">Carregando gráfico...</div>
      </div>
    );
  }

  const maxTotal = Math.max(...data.map(d => d.total), 1);

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-4 text-sm">
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 bg-blue-500 rounded"></span>
            Vendas
          </span>
          <span className="flex items-center gap-1">
            <span className="w-3 h-3 bg-green-500 rounded"></span>
            Finalizadas
          </span>
        </div>
        <span className="text-xs text-gray-400">Total: {data.reduce((acc, d) => acc + d.total, 0)}</span>
      </div>

      <div className="flex items-end h-48 gap-1">
        {data.map((item, index) => {
          const height = (item.total / maxTotal) * 100;
          const heightFinalizados = (item.finalizados / maxTotal) * 100;
          const isEven = index % 2 === 0;

          return (
            <div key={item.hora} className="flex-1 flex flex-col items-center">
              <div className="relative w-full flex items-end justify-center gap-1 h-40">
                {/* Barra de Finalizados */}
                <div 
                  className="w-3 bg-green-500 rounded-t transition-all duration-500"
                  style={{ height: `${Math.max(heightFinalizados, 2)}%` }}
                />
                {/* Barra de Total */}
                <div 
                  className="w-3 bg-blue-500 rounded-t transition-all duration-500"
                  style={{ height: `${Math.max(height, 2)}%` }}
                />
              </div>
              <span className="text-xs text-gray-500 mt-1">
                {String(item.hora).padStart(2, '0')}:00
              </span>
            </div>
          );
        })}
      </div>

      <div className="flex justify-between mt-2 text-xs text-gray-400">
        <span>Início</span>
        <span>Fim</span>
      </div>
    </div>
  );
};

export default VendasHojeChart;