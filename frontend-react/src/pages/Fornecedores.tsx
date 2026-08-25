import React, { useState, useEffect, useCallback } from "react";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Modal from "../components/ui/Modal";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";

interface Fornecedor {
  id: number;
  razao_social: string;
  nome_fantasia: string;
  cnpj: string;
  telefone: string;
  email: string;
  contato_nome: string;
  ativo: boolean;
}

const Fornecedores: React.FC = () => {
  const [fornecedores, setFornecedores] = useState<Fornecedor[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Fornecedor | null>(null);
  const [formData, setFormData] = useState({
    razao_social: "",
    nome_fantasia: "",
    cnpj: "",
    telefone: "",
    email: "",
    contato_nome: "",
  });
  const { get, post, put } = useApi();
  const { toasts, removeToast, success, error: showError } = useToast();

  const loadFornecedores = useCallback(async () => {
    try {
      const data = await get("/fornecedores");
      setFornecedores(data || []);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar fornecedores");
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    loadFornecedores();
  }, [loadFornecedores]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      if (editing) {
        await put(`/fornecedores/${editing.id}`, formData);
        success("Fornecedor atualizado com sucesso!");
      } else {
        await post("/fornecedores", formData);
        success("Fornecedor criado com sucesso!");
      }
      setShowModal(false);
      setEditing(null);
      setFormData({
        razao_social: "",
        nome_fantasia: "",
        cnpj: "",
        telefone: "",
        email: "",
        contato_nome: "",
      });
      loadFornecedores();
    } catch (err: any) {
      showError(err.message || "Erro ao salvar fornecedor");
    }
  };

  const handleToggle = async (id: number, ativo: boolean) => {
    try {
      await put(`/fornecedores/${id}`, { ativo: !ativo });
      success(`Fornecedor ${ativo ? "desativado" : "ativado"} com sucesso!`);
      loadFornecedores();
    } catch (err: any) {
      showError(err.message || "Erro ao alterar fornecedor");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando fornecedores...</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      {toasts.map((toast) => (
        <Toast
          key={toast.id}
          message={toast.message}
          type={toast.type}
          duration={toast.duration}
          onClose={() => removeToast(toast.id)}
        />
      ))}

      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">🏢 Fornecedores</h1>
          <p className="text-gray-500 text-sm mt-1">
            Gerencie os fornecedores da farmácia
          </p>
        </div>
        <Button
          onClick={() => {
            setEditing(null);
            setFormData({
              razao_social: "",
              nome_fantasia: "",
              cnpj: "",
              telefone: "",
              email: "",
              contato_nome: "",
            });
            setShowModal(true);
          }}
          icon="➕"
        >
          Novo Fornecedor
        </Button>
      </div>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Razão Social
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  CNPJ
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Contato
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                  Ações
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {fornecedores.map((fornecedor) => (
                <tr key={fornecedor.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <p className="text-sm font-medium text-gray-800">
                      {fornecedor.razao_social}
                    </p>
                    <p className="text-xs text-gray-500">
                      {fornecedor.nome_fantasia}
                    </p>
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-600">
                    {fornecedor.cnpj}
                  </td>
                  <td className="px-6 py-4">
                    <p className="text-sm text-gray-800">
                      {fornecedor.contato_nome}
                    </p>
                    <p className="text-xs text-gray-500">
                      {fornecedor.telefone}
                    </p>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`px-2 py-1 text-xs rounded-full ${fornecedor.ativo ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}
                    >
                      {fornecedor.ativo ? "✅ Ativo" : "❌ Inativo"}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-right space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setEditing(fornecedor);
                        setFormData({
                          razao_social: fornecedor.razao_social,
                          nome_fantasia: fornecedor.nome_fantasia || "",
                          cnpj: fornecedor.cnpj,
                          telefone: fornecedor.telefone || "",
                          email: fornecedor.email || "",
                          contato_nome: fornecedor.contato_nome || "",
                        });
                        setShowModal(true);
                      }}
                    >
                      Editar
                    </Button>
                    <Button
                      variant={fornecedor.ativo ? "warning" : "success"}
                      size="sm"
                      onClick={() =>
                        handleToggle(fornecedor.id, fornecedor.ativo)
                      }
                    >
                      {fornecedor.ativo ? "Desativar" : "Ativar"}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editing ? "✏️ Editar Fornecedor" : "➕ Novo Fornecedor"}
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Razão Social *"
            value={formData.razao_social}
            onChange={(e) =>
              setFormData({ ...formData, razao_social: e.target.value })
            }
            required
          />
          <Input
            label="Nome Fantasia"
            value={formData.nome_fantasia}
            onChange={(e) =>
              setFormData({ ...formData, nome_fantasia: e.target.value })
            }
          />
          <Input
            label="CNPJ *"
            value={formData.cnpj}
            onChange={(e) => setFormData({ ...formData, cnpj: e.target.value })}
            required
            placeholder="00.000.000/0000-00"
          />
          <Input
            label="Telefone"
            value={formData.telefone}
            onChange={(e) =>
              setFormData({ ...formData, telefone: e.target.value })
            }
          />
          <Input
            label="E-mail"
            type="email"
            value={formData.email}
            onChange={(e) =>
              setFormData({ ...formData, email: e.target.value })
            }
          />
          <Input
            label="Nome do Contato"
            value={formData.contato_nome}
            onChange={(e) =>
              setFormData({ ...formData, contato_nome: e.target.value })
            }
          />
          <div className="flex gap-3">
            <Button type="submit" fullWidth>
              {editing ? "Salvar" : "Criar"}
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

export default Fornecedores;
