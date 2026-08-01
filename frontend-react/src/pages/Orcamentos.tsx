import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../components/ui/Button';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';

interface Orcamento {
  id: number;
  cliente_id: number;
  cliente: string;
  total: number;
  status: 'pendente' | 'aprovado' | 'rejeitado' | 'expirado';
  created_at: string;
  expirado_em?: string;
}

const Orcamentos: React.FC = () => {
  const [orcamentos, setOrcamentos] = useState<Orcamento[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('todos');
  const navigate = useNavigate();
  const { get, post } = useApi();
  const { success, error: showError } = useToast();

  const fetchOrcamentos = useCallback(async () => {
    try {
      const url = filter !== 'todos' ? `/orcamentos?status=${filter}` : '/orcamentos';
      const data = await get(url);
      setOrcamentos(data);
    } catch (err: any) {
      showError(err.message || 'Erro ao carregar orçamentos');
    } finally {
      setLoading(false);
    }
  }, [filter, get, showError]);

  useEffect(() => {
    fetchOrcamentos();
  }, [fetchOrcamentos]);

  const handleStatusChange = async (id: number, status: string) => {
    try {
      const action = status === 'aprovado' ? 'aprovar' : 'rejeitar';
      await post(`/orcamentos/${id}/${action}`, {});
      success(`Orçamento ${status === 'aprovado' ? 'aprovado' : 'rejeitado'} com sucesso!`);
      fetchOrcamentos();
    } catch (err: any) {
      showError(err.message || 'Erro ao atualizar orçamento');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'aprovado': return 'bg-green-100 text-green-700';
      case 'rejeitado': return 'bg-red-100 text-red-700';
      case 'expirado': return 'bg-gray-100 text-gray-700';
      default: return 'bg-yellow-100 text-yellow-700';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'aprovado': return '✅';
      case 'rejeitado': return '❌';
      case 'expirado': return '⏰';
      default: return '⏳';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando orçamentos...</p>
        </div>
      </div>
    );
  }

  const formatCurrency = FormatterService.formatCurrency;
  const formatDate = FormatterService.formatDateTime;

  return (
    <div>
      <div className="flex flex-wrap justify-between items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">💰 Orçamentos</h1>
          <p className="text-gray-500 text-sm mt-1">Gerencie os orçamentos do sistema</p>
        </div>
        <div className="flex gap-2">
          {['todos', 'pendente', 'aprovado', 'rejeitado'].map((status) => (
            <Button
              key={status}
              variant={filter === status ? 'primary' : 'secondary'}
              size="sm"
              onClick={() => setFilter(status)}
            >
              {status === 'todos' ? 'Todos' :
               status === 'pendente' ? '⏳ Pendentes' :
               status === 'aprovado' ? '✅ Aprovados' : '❌ Rejeitados'}
            </Button>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Cliente</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Total</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Data</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {orcamentos.map(orcamento => (
                <tr key={orcamento.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 text-sm text-gray-600">#{orcamento.id}</td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-800">{orcamento.cliente}</td>
                  <td className="px-6 py-4 text-sm font-semibold text-gray-800">
                    {formatCurrency(orcamento.total)}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(orcamento.status)}`}>
                      {getStatusIcon(orcamento.status)} {orcamento.status}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {formatDate(orcamento.created_at)}
                  </td>
                  <td className="px-6 py-4 text-right space-x-2">
                    {orcamento.status === 'pendente' && (
                      <>
                        <Button
                          variant="success"
                          size="sm"
                          onClick={() => handleStatusChange(orcamento.id, 'aprovado')}
                        >
                          ✅ Aprovar
                        </Button>
                        <Button
                          variant="danger"
                          size="sm"
                          onClick={() => handleStatusChange(orcamento.id, 'rejeitado')}
                        >
                          ❌ Rejeitar
                        </Button>
                      </>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => navigate(`/orcamentos/${orcamento.id}`)}
                    >
                      Ver
                    </Button>
                  </td>
                </tr>
              ))}
              {orcamentos.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-gray-400">
                    Nenhum orçamento encontrado
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default Orcamentos;