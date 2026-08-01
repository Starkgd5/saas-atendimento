import React, { useState, useEffect, useCallback } from 'react';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';

interface Reclamacao {
  id: number;
  cliente_id: number;
  cliente: string;
  mensagem: string;
  status: 'pendente' | 'em_analise' | 'resolvido' | 'ignorado';
  prioridade: 'baixa' | 'media' | 'alta' | 'critica';
  categoria: string;
  resposta: string;
  created_at: string;
  resolvido_em?: string;
}

const Reclamacoes: React.FC = () => {
  const [reclamacoes, setReclamacoes] = useState<Reclamacao[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('pendente');
  const [selectedReclamacao, setSelectedReclamacao] = useState<Reclamacao | null>(null);
  const [resposta, setResposta] = useState('');
  const [showModal, setShowModal] = useState(false);
  const { get, post } = useApi();
  const { success, error: showError } = useToast();

  const fetchReclamacoes = useCallback(async () => {
    try {
      const url = filter !== 'todos' ? `/reclamacoes?status=${filter}` : '/reclamacoes';
      const data = await get(url);
      setReclamacoes(data);
    } catch (err: any) {
      showError(err.message || 'Erro ao carregar reclamações');
    } finally {
      setLoading(false);
    }
  }, [filter, get, showError]);

  useEffect(() => {
    fetchReclamacoes();
  }, [fetchReclamacoes]);

  const handleResolver = async () => {
    if (!selectedReclamacao || !resposta.trim()) return;

    try {
      await post(`/reclamacoes/${selectedReclamacao.id}/resolver`, { resposta });
      success('Reclamação resolvida com sucesso!');
      setShowModal(false);
      setResposta('');
      setSelectedReclamacao(null);
      fetchReclamacoes();
    } catch (err: any) {
      showError(err.message || 'Erro ao resolver reclamação');
    }
  };

  const getPriorityColor = (prioridade: string) => {
    switch (prioridade) {
      case 'critica': return 'bg-red-100 text-red-700';
      case 'alta': return 'bg-orange-100 text-orange-700';
      case 'media': return 'bg-yellow-100 text-yellow-700';
      default: return 'bg-blue-100 text-blue-700';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'resolvido': return 'bg-green-100 text-green-700';
      case 'ignorado': return 'bg-gray-100 text-gray-700';
      case 'em_analise': return 'bg-blue-100 text-blue-700';
      default: return 'bg-red-100 text-red-700';
    }
  };

  const getPriorityIcon = (prioridade: string) => {
    switch (prioridade) {
      case 'critica': return '🔴';
      case 'alta': return '🟠';
      case 'media': return '🟡';
      default: return '🔵';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando reclamações...</p>
        </div>
      </div>
    );
  }

  const formatDate = FormatterService.formatDateTime;

  return (
    <div>
      <div className="flex flex-wrap justify-between items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">📋 Reclamações</h1>
          <p className="text-gray-500 text-sm mt-1">Gerencie as reclamações dos clientes</p>
        </div>
        <div className="flex gap-2">
          {['todos', 'pendente', 'em_analise', 'resolvido'].map((status) => (
            <Button
              key={status}
              variant={filter === status ? 'primary' : 'secondary'}
              size="sm"
              onClick={() => setFilter(status)}
            >
              {status === 'todos' ? 'Todos' :
               status === 'pendente' ? '⏳ Pendentes' :
               status === 'em_analise' ? '🔍 Em Análise' :
               '✅ Resolvidos'}
            </Button>
          ))}
        </div>
      </div>

      <div className="grid gap-4">
        {reclamacoes.map(reclamacao => (
          <div key={reclamacao.id} className="bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-2 flex-wrap">
                  <span className={`px-2 py-1 text-xs rounded-full ${getPriorityColor(reclamacao.prioridade)}`}>
                    {getPriorityIcon(reclamacao.prioridade)} {reclamacao.prioridade}
                  </span>
                  <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(reclamacao.status)}`}>
                    {reclamacao.status === 'pendente' ? '⏳' :
                     reclamacao.status === 'em_analise' ? '🔍' :
                     reclamacao.status === 'resolvido' ? '✅' : '❌'} {reclamacao.status}
                  </span>
                  <span className="text-xs text-gray-400">
                    #{reclamacao.id}
                  </span>
                  {reclamacao.categoria && (
                    <span className="px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded-full">
                      📂 {reclamacao.categoria}
                    </span>
                  )}
                </div>
                <p className="font-medium text-gray-800">{reclamacao.cliente}</p>
                <p className="text-sm text-gray-600 mt-1">{reclamacao.mensagem}</p>
                {reclamacao.resposta && (
                  <div className="mt-2 p-3 bg-green-50 rounded-lg">
                    <p className="text-sm text-gray-600">
                      <span className="font-medium">Resposta:</span> {reclamacao.resposta}
                    </p>
                  </div>
                )}
                <p className="text-xs text-gray-400 mt-2">
                  {formatDate(reclamacao.created_at)}
                  {reclamacao.resolvido_em && ` • Resolvido em: ${formatDate(reclamacao.resolvido_em)}`}
                </p>
              </div>
              <div className="flex gap-2">
                {(reclamacao.status === 'pendente' || reclamacao.status === 'em_analise') && (
                  <Button
                    variant="success"
                    onClick={() => {
                      setSelectedReclamacao(reclamacao);
                      setShowModal(true);
                    }}
                    icon="✅"
                  >
                    Resolver
                  </Button>
                )}
                <Button
                  variant="outline"
                  onClick={() => {/* Ver detalhes */}}
                >
                  Detalhes
                </Button>
              </div>
            </div>
          </div>
        ))}
        {reclamacoes.length === 0 && (
          <div className="bg-white rounded-xl shadow-md p-12 text-center">
            <p className="text-gray-400">Nenhuma reclamação encontrada</p>
          </div>
        )}
      </div>

      {/* Modal Resolver */}
      <Modal
        isOpen={showModal}
        onClose={() => {
          setShowModal(false);
          setResposta('');
          setSelectedReclamacao(null);
        }}
        title="✅ Resolver Reclamação"
        footer={
          <div className="flex gap-3">
            <Button
              variant="success"
              fullWidth
              onClick={handleResolver}
              disabled={!resposta.trim()}
            >
              Resolver
            </Button>
            <Button
              variant="secondary"
              fullWidth
              onClick={() => {
                setShowModal(false);
                setResposta('');
                setSelectedReclamacao(null);
              }}
            >
              Cancelar
            </Button>
          </div>
        }
      >
        {selectedReclamacao && (
          <div className="space-y-4">
            <div className="p-4 bg-gray-50 rounded-lg">
              <p className="text-sm">
                <span className="font-medium">Cliente:</span> {selectedReclamacao.cliente}
              </p>
              <p className="text-sm mt-1">
                <span className="font-medium">Prioridade:</span>{' '}
                <span className={`px-2 py-0.5 text-xs rounded-full ${getPriorityColor(selectedReclamacao.prioridade)}`}>
                  {getPriorityIcon(selectedReclamacao.prioridade)} {selectedReclamacao.prioridade}
                </span>
              </p>
            </div>
            <div className="p-4 bg-red-50 rounded-lg">
              <p className="text-sm text-gray-700">
                <span className="font-medium">Mensagem:</span>
              </p>
              <p className="text-sm text-gray-600 mt-1">{selectedReclamacao.mensagem}</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Resposta</label>
              <textarea
                value={resposta}
                onChange={(e) => setResposta(e.target.value)}
                placeholder="Digite sua resposta..."
                className="w-full px-4 py-3 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-green-500 min-h-[100px]"
                required
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Reclamacoes;