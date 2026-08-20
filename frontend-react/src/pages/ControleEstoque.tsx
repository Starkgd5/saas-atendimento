import React, { useState, useEffect } from 'react';
import Input from '../components/ui/Input';
import Button from '../components/ui/Button';
import Modal from '../components/ui/Modal';
// import Table from '../components/ui/Table';

interface ProdutoEstoque {
  id: number;
  nome: string;
  codigo_barras: string;
  categoria: string;
  quantidade_total: number;
  lotes: Lote[];
  alerta: 'ok' | 'baixo' | 'critico' | 'vencido';
}

interface Lote {
  id: number;
  numero: string;
  validade: string;
  quantidade: number;
  status: 'ativo' | 'vencido' | 'baixo';
}

export const ControleEstoque: React.FC = () => {
  const [produtos, setProdutos] = useState<ProdutoEstoque[]>([]);
  const [search, setSearch] = useState('');
  const [selectedProduto, setSelectedProduto] = useState<ProdutoEstoque | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchEstoque();
  }, []);

  const fetchEstoque = async () => {
    try {
      const response = await fetch('/api/v1/estoque/produtos', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      setProdutos(data);
    } catch (error) {
      console.error('Erro ao carregar estoque:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (produto: ProdutoEstoque) => {
    if (produto.alerta === 'vencido') return 'bg-red-100 text-red-800';
    if (produto.alerta === 'critico') return 'bg-red-50 text-red-600';
    if (produto.alerta === 'baixo') return 'bg-yellow-50 text-yellow-600';
    return 'bg-green-50 text-green-600';
  };

  const getStatusText = (produto: ProdutoEstoque) => {
    if (produto.alerta === 'vencido') return '⚠️ Vencido';
    if (produto.alerta === 'critico') return '🔴 Crítico';
    if (produto.alerta === 'baixo') return '🟡 Baixo';
    return '🟢 Normal';
  };

  const filteredProdutos = produtos.filter(p => 
    p.nome.toLowerCase().includes(search.toLowerCase()) ||
    p.codigo_barras.includes(search)
  );

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">📦 Controle de Estoque</h1>
        <div className="flex gap-2">
          <Input
            type="text"
            placeholder="Buscar produto..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-64"
          />
          <Button variant="primary" icon="➕">
            Novo Produto
          </Button>
          <Button variant="success" icon="📥">
            Entrada de Estoque
          </Button>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Produto</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Código</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Categoria</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Estoque</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Lotes</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Ações</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {filteredProdutos.map((produto) => (
              <tr key={produto.id} className="hover:bg-gray-50">
                <td className="px-6 py-4">
                  <div className="font-medium text-gray-900">{produto.nome}</div>
                </td>
                <td className="px-6 py-4 text-sm text-gray-500">{produto.codigo_barras}</td>
                <td className="px-6 py-4 text-sm text-gray-500">{produto.categoria}</td>
                <td className="px-6 py-4 text-sm font-medium">{produto.quantidade_total}</td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(produto)}`}>
                    {getStatusText(produto)}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <div className="flex flex-wrap gap-1">
                    {produto.lotes.slice(0, 2).map((lote) => (
                      <span key={lote.id} className="px-2 py-0.5 bg-gray-100 rounded text-xs">
                        {lote.numero} ({lote.quantidade})
                      </span>
                    ))}
                    {produto.lotes.length > 2 && (
                      <span className="text-xs text-gray-400">+{produto.lotes.length - 2}</span>
                    )}
                  </div>
                </td>
                <td className="px-6 py-4 text-right space-x-2">
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={() => {
                      setSelectedProduto(produto);
                      setShowModal(true);
                    }}
                  >
                    Ver Lotes
                  </Button>
                  <Button variant="outline" size="sm">Editar</Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Modal de Lotes */}
      {showModal && selectedProduto && (
        <Modal
          isOpen={showModal}
          onClose={() => setShowModal(false)}
          title={`Lotes - ${selectedProduto.nome}`}
        >
          <div className="space-y-4">
            <div className="grid grid-cols-4 gap-2 text-sm font-medium text-gray-500 border-b pb-2">
              <span>Nº Lote</span>
              <span>Validade</span>
              <span>Quantidade</span>
              <span>Status</span>
            </div>
            {selectedProduto.lotes.map((lote) => (
              <div key={lote.id} className="grid grid-cols-4 gap-2 text-sm">
                <span>{lote.numero}</span>
                <span className={new Date(lote.validade) < new Date() ? 'text-red-600' : ''}>
                  {new Date(lote.validade).toLocaleDateString('pt-BR')}
                </span>
                <span>{lote.quantidade}</span>
                <span>
                  {lote.status === 'vencido' ? '⚠️ Vencido' :
                   lote.status === 'baixo' ? '🟡 Baixo' : '🟢 Ativo'}
                </span>
              </div>
            ))}
          </div>
        </Modal>
      )}
    </div>
  );
};

export default ControleEstoque;