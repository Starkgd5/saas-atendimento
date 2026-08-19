import React, { useState } from 'react';
import { useProdutos, useProdutoMutation, useProdutoUpdate } from '../hooks/useCachedResources';
import { useToast } from '../hooks/useToast';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';
import FormatterService from '../services/formatter.service';

interface Produto {
  id: number;
  nome: string;
  preco: number;
  categoria: string;
  estoque: number;
  ativo: boolean;
}

const Produtos: React.FC = () => {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingProduto, setEditingProduto] = useState<Produto | null>(null);
  const [formData, setFormData] = useState({
    nome: '',
    preco: '',
    categoria: '',
    estoque: '',
  });

  const { data, loading, error, refetch, isStale } = useProdutos({
    limit: 20,
    offset: (page - 1) * 20,
  });

  const { mutate: createProduto, loading: creating } = useProdutoMutation();
  const { mutate: updateProduto, loading: updating } = useProdutoUpdate(editingProduto?.id || 0);
  const { success, error: showError } = useToast();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        nome: formData.nome,
        preco: parseFloat(formData.preco),
        categoria: formData.categoria,
        estoque: parseInt(formData.estoque),
      };

      if (editingProduto) {
        await updateProduto(payload);
        success('Produto atualizado com sucesso!');
      } else {
        await createProduto(payload);
        success('Produto criado com sucesso!');
      }

      setShowModal(false);
      setEditingProduto(null);
      setFormData({ nome: '', preco: '', categoria: '', estoque: '' });
      refetch();
    } catch (err: any) {
      showError(err.message || 'Erro ao salvar produto');
    }
  };

  const formatCurrency = FormatterService.formatCurrency;

  if (loading && !data) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando produtos...</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">📦 Produtos</h1>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-sm text-gray-500">
              {data?.total || 0} produtos encontrados
            </p>
            {isStale && (
              <span className="text-xs text-yellow-500 bg-yellow-50 px-2 py-0.5 rounded">
                ⚡ Dados desatualizados
              </span>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Input
            type="text"
            placeholder="Buscar produto..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-64"
          />
          <Button
            variant="primary"
            onClick={() => {
              setEditingProduto(null);
              setFormData({ nome: '', preco: '', categoria: '', estoque: '' });
              setShowModal(true);
            }}
            icon="➕"
          >
            Novo Produto
          </Button>
          <Button
            variant="outline"
            onClick={refetch}
            icon="🔄"
          >
            Atualizar
          </Button>
        </div>
      </div>

      {/* Tabela de produtos */}
      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Nome</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Categoria</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Preço</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Estoque</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Ações</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {data?.items?.map((produto: Produto) => (
              <tr key={produto.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 text-sm font-medium text-gray-800">{produto.nome}</td>
                <td className="px-6 py-4 text-sm text-gray-600">{produto.categoria}</td>
                <td className="px-6 py-4 text-sm font-medium text-gray-800">
                  {formatCurrency(produto.preco)}
                </td>
                <td className="px-6 py-4 text-sm text-gray-600">{produto.estoque}</td>
                <td className="px-6 py-4 text-right space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setEditingProduto(produto);
                      setFormData({
                        nome: produto.nome,
                        preco: produto.preco.toString(),
                        categoria: produto.categoria,
                        estoque: produto.estoque.toString(),
                      });
                      setShowModal(true);
                    }}
                  >
                    Editar
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Paginação */}
      {data && data.total > 20 && (
        <div className="flex justify-between items-center mt-4">
          <span className="text-sm text-gray-500">
            Mostrando {(page - 1) * 20 + 1} - {Math.min(page * 20, data.total)} de {data.total}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page === 1}
              onClick={() => setPage(page - 1)}
            >
              Anterior
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page * 20 >= data.total}
              onClick={() => setPage(page + 1)}
            >
              Próxima
            </Button>
          </div>
        </div>
      )}

      {/* Modal */}
      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editingProduto ? '✏️ Editar Produto' : '➕ Novo Produto'}
      >
        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <Input
              label="Nome"
              value={formData.nome}
              onChange={(e) => setFormData({ ...formData, nome: e.target.value })}
              required
            />
            <Input
              label="Categoria"
              value={formData.categoria}
              onChange={(e) => setFormData({ ...formData, categoria: e.target.value })}
              required
            />
            <Input
              label="Preço (R$)"
              type="number"
              step="0.01"
              value={formData.preco}
              onChange={(e) => setFormData({ ...formData, preco: e.target.value })}
              required
            />
            <Input
              label="Estoque"
              type="number"
              value={formData.estoque}
              onChange={(e) => setFormData({ ...formData, estoque: e.target.value })}
              required
            />
          </div>

          <div className="flex gap-3 mt-6">
            <Button
              type="submit"
              loading={creating || updating}
              fullWidth
            >
              {editingProduto ? 'Salvar' : 'Criar'}
            </Button>
            <Button
              variant="secondary"
              fullWidth
              onClick={() => setShowModal(false)}
            >
              Cancelar
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default Produtos;