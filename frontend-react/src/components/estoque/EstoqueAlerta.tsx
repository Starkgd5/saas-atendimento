import React, { useState, useEffect } from 'react';
import { useApi } from '../../hooks/useApi';
import { useToast } from '../../hooks/useToast';

interface EstoqueAlertaProps {
  produtosBaixoEstoque?: number;
  produtosVencendo?: number;
}

interface AlertaItem {
  produto_id: number;
  nome: string;
  estoque?: number;
  estoque_min?: number;
  quantidade?: number;
  data_validade?: string;
  dias_restantes?: number;
  tipo: 'baixo_estoque' | 'vencendo';
}

export const EstoqueAlerta: React.FC<EstoqueAlertaProps> = ({
  produtosBaixoEstoque: propBaixoEstoque,
  produtosVencendo: propVencendo,
}) => {
  const [alertas, setAlertas] = useState<AlertaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const { get } = useApi();
  const { success, error: showError } = useToast();

  useEffect(() => {
    fetchAlertas();
  }, []);

  const fetchAlertas = async () => {
    try {
      const response = await get('/estoque/alertas');
      
      const novosAlertas: AlertaItem[] = [];

      // Alertas de baixo estoque
      if (response.baixo_estoque) {
        response.baixo_estoque.forEach((item: any) => {
          novosAlertas.push({
            produto_id: item.produto_id,
            nome: item.nome,
            estoque: item.estoque,
            estoque_min: item.estoque_min,
            tipo: 'baixo_estoque',
          });
        });
      }

      // Alertas de produtos vencendo
      if (response.vencendo) {
        response.vencendo.forEach((item: any) => {
          novosAlertas.push({
            produto_id: item.produto_id,
            nome: item.nome,
            quantidade: item.quantidade,
            data_validade: item.data_validade,
            dias_restantes: item.dias_restantes,
            tipo: 'vencendo',
          });
        });
      }

      setAlertas(novosAlertas);
    } catch (error) {
      console.error('Erro ao carregar alertas:', error);
      // Dados mockados para demonstração
      setAlertas([
        { 
          produto_id: 1, 
          nome: 'Dipirona Sódica 500mg', 
          estoque: 3, 
          estoque_min: 10, 
          tipo: 'baixo_estoque' 
        },
        { 
          produto_id: 2, 
          nome: 'Amoxicilina 500mg', 
          estoque: 5, 
          estoque_min: 15, 
          tipo: 'baixo_estoque' 
        },
        { 
          produto_id: 3, 
          nome: 'Losartana 50mg', 
          quantidade: 20, 
          data_validade: '2024-02-15', 
          dias_restantes: 5, 
          tipo: 'vencendo' 
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const handleResolverAlerta = async (alerta: AlertaItem) => {
    try {
      // Ações para resolver alerta
      if (alerta.tipo === 'baixo_estoque') {
        // TODO: Navegar para entrada de estoque
        success(`Iniciando entrada de estoque para ${alerta.nome}`);
      } else {
        // TODO: Navegar para ajuste de estoque
        success(`Iniciando ajuste para ${alerta.nome}`);
      }
    } catch (error) {
      showError('Erro ao resolver alerta');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-32">
        <div className="text-gray-400">Carregando alertas...</div>
      </div>
    );
  }

  if (alertas.length === 0) {
    return (
      <div className="text-center py-8 text-gray-400">
        <div className="text-4xl mb-2">✅</div>
        <p>Nenhum alerta de estoque no momento</p>
        <p className="text-sm mt-1">Todos os produtos estão com estoque adequado</p>
      </div>
    );
  }

  const alertasBaixoEstoque = alertas.filter(a => a.tipo === 'baixo_estoque');
  const alertasVencendo = alertas.filter(a => a.tipo === 'vencendo');

  return (
    <div className="space-y-4">
      {alertasBaixoEstoque.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
            <span className="text-yellow-500">⚠️</span>
            Estoque Baixo ({alertasBaixoEstoque.length})
          </h4>
          <div className="space-y-2">
            {alertasBaixoEstoque.map((alerta) => (
              <div 
                key={`${alerta.produto_id}-baixo`}
                className="flex items-center justify-between bg-yellow-50 border border-yellow-200 rounded-lg p-3"
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-800 truncate">{alerta.nome}</p>
                  <p className="text-xs text-gray-500">
                    Estoque: {alerta.estoque} / Mínimo: {alerta.estoque_min}
                  </p>
                </div>
                <button
                  onClick={() => handleResolverAlerta(alerta)}
                  className="ml-2 px-3 py-1 bg-yellow-500 text-white text-xs rounded hover:bg-yellow-600 transition"
                >
                  Repor
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {alertasVencendo.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
            <span className="text-red-500">🔴</span>
            Vencendo em Breve ({alertasVencendo.length})
          </h4>
          <div className="space-y-2">
            {alertasVencendo.map((alerta) => (
              <div 
                key={`${alerta.produto_id}-vencendo`}
                className="flex items-center justify-between bg-red-50 border border-red-200 rounded-lg p-3"
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-800 truncate">{alerta.nome}</p>
                  <p className="text-xs text-gray-500">
                    {alerta.quantidade} unidades • Vence em {alerta.dias_restantes} dias
                  </p>
                </div>
                <button
                  onClick={() => handleResolverAlerta(alerta)}
                  className="ml-2 px-3 py-1 bg-red-500 text-white text-xs rounded hover:bg-red-600 transition"
                >
                  Ajustar
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <button
        onClick={fetchAlertas}
        className="w-full text-center text-xs text-blue-500 hover:text-blue-600 transition"
      >
        🔄 Atualizar alertas
      </button>
    </div>
  );
};

export default EstoqueAlerta;