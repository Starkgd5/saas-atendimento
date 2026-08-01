import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import Button from '../components/ui/Button';
import Input from '../components/ui/Input';
import Modal from '../components/ui/Modal';
import { useApi } from '../hooks/useApi';
import { useToast } from '../hooks/useToast';
import FormatterService from '../services/formatter.service';

interface User {
  id: number;
  nome: string;
  email: string;
  role: string;
  ativo: boolean;
  loja_id: number | null;
  loja_nome?: string;
  created_at: string;
}

const UserManagement: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formData, setFormData] = useState({
    nome: '',
    email: '',
    password: '',
    role: 'atendente',
    loja_id: ''
  });
  // const navigate = useNavigate();
  const { get, post, put, patch, delete: del } = useApi();
  const { success, error: showError } = useToast();

  const fetchUsers = useCallback(async () => {
    try {
      const data = await get('/usuarios');
      setUsers(data);
    } catch (err: any) {
      showError(err.message || 'Erro ao carregar usuários');
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const payload = {
        ...formData,
        loja_id: formData.loja_id ? parseInt(formData.loja_id) : null
      };

      if (editingUser) {
        await put(`/usuarios/${editingUser.id}`, payload);
        success('Usuário atualizado com sucesso!');
      } else {
        await post('/usuarios', payload);
        success('Usuário criado com sucesso!');
      }

      setShowModal(false);
      setEditingUser(null);
      setFormData({ nome: '', email: '', password: '', role: 'atendente', loja_id: '' });
      fetchUsers();
    } catch (err: any) {
      showError(err.message || 'Erro ao salvar usuário');
    }
  };

  const handleToggleUser = async (id: number, ativo: boolean) => {
    try {
      await patch(`/usuarios/${id}/toggle`, { ativo: !ativo });
      success(`Usuário ${ativo ? 'desativado' : 'ativado'} com sucesso!`);
      fetchUsers();
    } catch (err: any) {
      showError(err.message || 'Erro ao alterar usuário');
    }
  };

  const handleDeleteUser = async (id: number) => {
    if (!window.confirm('Tem certeza que deseja excluir este usuário?')) return;
    
    try {
      await del(`/usuarios/${id}`);
      success('Usuário excluído com sucesso!');
      fetchUsers();
    } catch (err: any) {
      showError(err.message || 'Erro ao excluir usuário');
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando usuários...</p>
        </div>
      </div>
    );
  }

  const formatDate = FormatterService.formatDateTime;

  return (
    <div>
      <div className="flex flex-wrap justify-between items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">👥 Gerenciar Usuários</h1>
          <p className="text-gray-500 text-sm mt-1">Gerencie os usuários do sistema</p>
        </div>
        <Button
          onClick={() => {
            setEditingUser(null);
            setFormData({ nome: '', email: '', password: '', role: 'atendente', loja_id: '' });
            setShowModal(true);
          }}
          icon="➕"
        >
          Novo Usuário
        </Button>
      </div>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Nome</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Função</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Loja</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Criado em</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {users.map(user => (
                <tr key={user.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 text-sm font-medium text-gray-800">{user.nome}</td>
                  <td className="px-6 py-4 text-sm text-gray-600">{user.email}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      user.role === 'admin' 
                        ? 'bg-purple-100 text-purple-700'
                        : user.role === 'gerente'
                        ? 'bg-blue-100 text-blue-700'
                        : 'bg-gray-100 text-gray-700'
                    }`}>
                      {user.role === 'admin' ? '👑 Admin' :
                       user.role === 'gerente' ? '📋 Gerente' :
                       '🎯 Atendente'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-600">{user.loja_nome || '-'}</td>
                  <td className="px-6 py-4">
                    <span className={`px-2 py-1 text-xs rounded-full ${
                      user.ativo 
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                    }`}>
                      {user.ativo ? '✅ Ativo' : '❌ Inativo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500">
                    {formatDate(user.created_at)}
                  </td>
                  <td className="px-6 py-4 text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditingUser(user);
                        setFormData({
                          nome: user.nome,
                          email: user.email,
                          password: '',
                          role: user.role,
                          loja_id: user.loja_id?.toString() || ''
                        });
                        setShowModal(true);
                      }}
                    >
                      Editar
                    </Button>
                    <Button
                      variant={user.ativo ? 'warning' : 'success'}
                      size="sm"
                      onClick={() => handleToggleUser(user.id, user.ativo)}
                    >
                      {user.ativo ? 'Desativar' : 'Ativar'}
                    </Button>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDeleteUser(user.id)}
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
        title={editingUser ? '✏️ Editar Usuário' : '➕ Novo Usuário'}
        footer={
          <div className="flex gap-3">
            <Button
              variant="primary"
              fullWidth
              onClick={handleSubmit}
            >
              {editingUser ? 'Salvar' : 'Criar'}
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
            label="Nome completo"
            value={formData.nome}
            onChange={(e) => setFormData({ ...formData, nome: e.target.value })}
            placeholder="Digite o nome completo"
            required
          />
          <Input
            label="E-mail"
            type="email"
            value={formData.email}
            onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            placeholder="email@exemplo.com"
            required
          />
          <Input
            label={editingUser ? 'Nova senha (opcional)' : 'Senha'}
            type="password"
            value={formData.password}
            onChange={(e) => setFormData({ ...formData, password: e.target.value })}
            placeholder={editingUser ? 'Deixe em branco para manter' : 'Digite uma senha'}
            required={!editingUser}
          />
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Função</label>
            <select
              value={formData.role}
              onChange={(e) => setFormData({ ...formData, role: e.target.value })}
              className="w-full px-4 py-2.5 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="atendente">🎯 Atendente</option>
              <option value="gerente">📋 Gerente</option>
              <option value="admin">👑 Administrador</option>
            </select>
          </div>
          <Input
            label="ID da Loja (opcional)"
            type="number"
            value={formData.loja_id}
            onChange={(e) => setFormData({ ...formData, loja_id: e.target.value })}
            placeholder="Digite o ID da loja"
          />
        </div>
      </Modal>
    </div>
  );
};

export default UserManagement;