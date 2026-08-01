import React, { useState, useEffect, useCallback } from 'react';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';

interface Produto {
  id: number;
  nome: string;
  descricao: string;
  categoria: string;
  preco: number;
  estoque: number;
  estoque_min: number;
  requere_receita: boolean;
  ativo: boolean;
  created_at: string;
}

const Produtos: React.FC = () => {
  const [produtos, setProdutos] = useState<Produto[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingProduto, setEditingProduto] = useState<Produto | null>(null);
  const [formData, setFormData] = useState({
    nome: '',
    descricao: '',
    categoria: '',
    preco: '',
    estoque: '',
    estoque_min: '',
    requere_receita: false
  });
  const { get, post, put, delete: del } = useApi();
  const { success, error: showError } = useToast();

  const fetchProdutos = useCallback(async () => {
    try {
      const data = await get('/produtos');
      setProdutos(data);
    } catch (err: any) {
      showError(err.message || 'Erro ao carregar produtos');
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    fetchProdutos();
  }, [fetchProdutos]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        ...formData,
        preco: parseFloat(formData.preco),
        estoque: parseInt(formData.estoque),
        estoque_min: parseInt(formData.estoque_min)
      };

      if (editingProduto) {
        await put(`/produtos/${editingProduto.id}`, payload);
        success('Produto atualizado com sucesso!');
      } else {
        await post('/produtos', payload);
        success('Produto criado com sucesso!');
      }

      setShowModal(false);
      setEditingProduto(null);
      setFormData({ nome: '', descricao: '', categoria: '', preco: '', estoque: '', estoque_min: '', requere_receita: false });
      fetchProdutos();
    } catch (err: any) {
      showError(err.message || 'Erro ao salvar produto');
    }
  };

  const handleToggleProduto = async (id: number, ativo: boolean) => {
    try {
      await put(`/produtos/${id}`, { ativo: !ativo });
      success(`Produto ${ativo ? 'desativado' : 'ativado'} com sucesso!`);
      fetchProdutos();
    } catch (err: any) {
      showError(err.message || 'Erro ao alterar produto');
    }
  };

  const handleDeleteProduto = async (id: number) => {
    if (!window.confirm('Tem certeza que deseja excluir este produto?')) return;
    
    try {
      await del(`/produtos/${id}`);
      success('Produto excluído com sucesso!');
      fetchProdutos();
    } catch (err: any) {
      showError(err.message || 'Erro ao excluir produto');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando produtos...</p>
        </div>
      </div>
    );
  }

  const formatCurrency = FormatterService.formatCurrency;

  return (
    <div>
      <div className="flex flex-wrap justify-between items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">📦 Produtos</h1>
          <p className="text-gray-500 text-sm mt-1">Gerencie os produtos da farmácia</p>
        </div>
        <Button
          onClick={() => {
            setEditingProduto(null);
            setFormData({ nome: '', descricao: '', categoria: '', preco: '', estoque: '', estoque_min: '', requere_receita: false });
            setShowModal(true);
          }}
          icon="➕"
        >
          Novo Produto
        </Button>
      </div>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Nome</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Categoria</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Preço</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Estoque</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {produtos.map(produto => (
                <tr key={produto.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <div>
                      <p className="text-sm font-medium text-gray-800">{produto.nome}</p>
                      {produto.requere_receita && (
                        <span className="text-xs text-red-500">⚠️ Requer receita</span>
                      )}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-600">{produto.categoria}</td>
                  <td className="px-6 py-4 text-sm font-medium text-gray-800">
                    {formatCurrency(produto.preco)}
                  </td>
                  <td className="px-6 py-4">
                    <span className={`text-sm ${produto.estoque <= produto.estoque_min ? 'text-red-500 font-medium' : 'text-gray-600'}`}>
                      {produto.estoque}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      produto.ativo 
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {produto.ativo ? '✅ Ativo' : '❌ Inativo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditingProduto(produto);
                        setFormData({
                          nome: produto.nome,
                          descricao: produto.descricao || '',
                          categoria: produto.categoria,
                          preco: produto.preco.toString(),
                          estoque: produto.estoque.toString(),
                          estoque_min: produto.estoque_min.toString(),
                          requere_receita: produto.requere_receita
                        });
                        setShowModal(true);
                      }}
                    >
                      Editar
                    </Button>
                    <Button
                      variant={produto.ativo ? 'warning' : 'success'}
                      size="sm"
                      onClick={() => handleToggleProduto(produto.id, produto.ativo)}
                    >
                      {produto.ativo ? 'Desativar' : 'Ativar'}
                    </Button>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDeleteProduto(produto.id)}
                    >
                      Excluir
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modal */}
      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editingProduto ? '✏️ Editar Produto' : '➕ Novo Produto'}
        footer={
          <div className="flex gap-3">
            <Button
              variant="primary"
              fullWidth
              onClick={handleSubmit}
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
        }
      >
        <div className="space-y-4">
          <Input
            label="Nome"
            value={formData.nome}
            onChange={(e) => setFormData({ ...formData, nome: e.target.value })}
            placeholder="Digite o nome do produto"
            required
          />
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Descrição</label>
            <textarea
              value={formData.descricao}
              onChange={(e) => setFormData({ ...formData, descricao: e.target.value })}
              className="w-full px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              placeholder="Descrição do produto"
            />
          </div>
          <Input
            label="Categoria"
            value={formData.categoria}
            onChange={(e) => setFormData({ ...formData, categoria: e.target.value })}
            placeholder="Ex: Analgésico"
            required
          />
          <Input
            label="Preço (R$)"
            type="number"
            step="0.01"
            value={formData.preco}
            onChange={(e) => setFormData({ ...formData, preco: e.target.value })}
            placeholder="0,00"
            required
          />
          <Input
            label="Estoque"
            type="number"
            value={formData.estoque}
            onChange={(e) => setFormData({ ...formData, estoque: e.target.value })}
            placeholder="Quantidade em estoque"
            required
          />
          <Input
            label="Estoque Mínimo"
            type="number"
            value={formData.estoque_min}
            onChange={(e) => setFormData({ ...formData, estoque_min: e.target.value })}
            placeholder="Quantidade mínima para alerta"
            required
          />
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={formData.requere_receita}
              onChange={(e) => setFormData({ ...formData, requere_receita: e.target.checked })}
              className="w-4 h-4 text-blue-500 rounded focus:ring-blue-500"
            />
            <label className="text-sm text-gray-700">Requere receita médica</label>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default Produtos;