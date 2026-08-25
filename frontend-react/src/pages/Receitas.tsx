import React, { useState, useEffect, useCallback } from "react";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Modal from "../components/ui/Modal";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";
import FormatterService from "../services/formatter.service";

interface Receita {
  id: number;
  cliente_id: number;
  cliente: string;
  numero_receita: string;
  medico_nome: string;
  medico_crm: string;
  data_emissao: string;
  data_validade: string;
  status: "Pendente" | "Validada" | "Rejeitada" | "Expirada";
  observacao: string;
  arquivo_url?: string;
  created_at: string;
}

const Receitas: React.FC = () => {
  const [receitas, setReceitas] = useState<Receita[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("Pendente");
  const [selectedReceita, setSelectedReceita] = useState<Receita | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [observacao, setObservacao] = useState("");
  const { get, post } = useApi();
  const { toasts, removeToast, success, error: showError } = useToast();
  const formatDate = FormatterService.formatDate;

  const loadReceitas = useCallback(async () => {
    try {
      const url =
        filter !== "todos" ? `/receitas?status=${filter}` : "/receitas";
      const data = await get(url);
      setReceitas(data || []);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar receitas");
    } finally {
      setLoading(false);
    }
  }, [filter, get, showError]);

  useEffect(() => {
    loadReceitas();
  }, [loadReceitas]);

  const handleValidar = async (id: number) => {
    try {
      await post(`/receitas/${id}/validar`, { observacao });
      success("Receita validada com sucesso!");
      setShowModal(false);
      setSelectedReceita(null);
      setObservacao("");
      loadReceitas();
    } catch (err: any) {
      showError(err.message || "Erro ao validar receita");
    }
  };

  const handleRejeitar = async (id: number) => {
    try {
      await post(`/receitas/${id}/rejeitar`, { observacao });
      success("Receita rejeitada!");
      setShowModal(false);
      setSelectedReceita(null);
      setObservacao("");
      loadReceitas();
    } catch (err: any) {
      showError(err.message || "Erro ao rejeitar receita");
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "Validada":
        return "bg-green-100 text-green-700";
      case "Pendente":
        return "bg-yellow-100 text-yellow-700";
      case "Rejeitada":
        return "bg-red-100 text-red-700";
      case "Expirada":
        return "bg-gray-100 text-gray-700";
      default:
        return "bg-gray-100 text-gray-700";
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando receitas...</p>
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

      <div className="flex flex-wrap justify-between items-center gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">
            📋 Receitas Médicas
          </h1>
          <p className="text-gray-500 text-sm mt-1">
            Validação de receitas para medicamentos controlados
          </p>
        </div>
        <div className="flex gap-2">
          {["todos", "Pendente", "Validada", "Rejeitada", "Expirada"].map(
            (status) => (
              <Button
                key={status}
                variant={filter === status ? "primary" : "secondary"}
                size="sm"
                onClick={() => setFilter(status)}
              >
                {status}
              </Button>
            ),
          )}
        </div>
      </div>

      <div className="grid gap-4">
        {receitas.length === 0 ? (
          <div className="bg-white rounded-xl shadow-md p-12 text-center text-gray-400">
            <div className="text-6xl mb-4">📋</div>
            <p className="text-lg font-medium">Nenhuma receita encontrada</p>
            <p className="text-sm">
              As receitas aparecerão aqui quando forem registradas
            </p>
          </div>
        ) : (
          receitas.map((receita) => (
            <Card key={receita.id} className="hover:shadow-lg transition">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-2 flex-wrap">
                    <span
                      className={`px-2 py-1 text-xs rounded-full ${getStatusColor(receita.status)}`}
                    >
                      {receita.status === "Pendente"
                        ? "⏳"
                        : receita.status === "Validada"
                          ? "✅"
                          : receita.status === "Rejeitada"
                            ? "❌"
                            : "⏰"}{" "}
                      {receita.status}
                    </span>
                    <span className="text-xs text-gray-400">#{receita.id}</span>
                    <span className="text-xs text-gray-400">
                      • {receita.numero_receita}
                    </span>
                  </div>
                  <p className="font-medium text-gray-800">{receita.cliente}</p>
                  <p className="text-sm text-gray-600">
                    Médico: {receita.medico_nome} (CRM: {receita.medico_crm})
                  </p>
                  <p className="text-xs text-gray-400 mt-1">
                    Emissão: {formatDate(receita.data_emissao)} • Validade:{" "}
                    {formatDate(receita.data_validade)}
                  </p>
                  {receita.observacao && (
                    <p className="text-sm text-gray-500 mt-1">
                      {receita.observacao}
                    </p>
                  )}
                </div>
                <div className="flex gap-2">
                  {receita.status === "Pendente" && (
                    <>
                      <Button
                        variant="success"
                        size="sm"
                        onClick={() => {
                          setSelectedReceita(receita);
                          setObservacao("");
                          setShowModal(true);
                        }}
                        icon="✅"
                      >
                        Validar
                      </Button>
                      <Button
                        variant="danger"
                        size="sm"
                        onClick={() => {
                          setSelectedReceita(receita);
                          setObservacao("");
                          setShowModal(true);
                        }}
                        icon="❌"
                      >
                        Rejeitar
                      </Button>
                    </>
                  )}
                  {receita.arquivo_url && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => window.open(receita.arquivo_url, "_blank")}
                      icon="📎"
                    >
                      Ver Receita
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))
        )}
      </div>

      {/* Modal de Validação/Rejeição */}
      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={
          selectedReceita?.status === "Pendente" ? "📋 Validar Receita" : "Ação"
        }
      >
        <div className="space-y-4">
          <div className="p-4 bg-gray-50 rounded-lg">
            <p className="text-sm">
              <span className="font-medium">Cliente:</span>{" "}
              {selectedReceita?.cliente}
            </p>
            <p className="text-sm">
              <span className="font-medium">Médico:</span>{" "}
              {selectedReceita?.medico_nome} (CRM: {selectedReceita?.medico_crm}
              )
            </p>
            <p className="text-sm">
              <span className="font-medium">Nº Receita:</span>{" "}
              {selectedReceita?.numero_receita}
            </p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Observação
            </label>
            <textarea
              value={observacao}
              onChange={(e) => setObservacao(e.target.value)}
              placeholder="Adicione uma observação (opcional)"
              className="w-full px-4 py-2 border border-gray-300 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[80px]"
            />
          </div>
          <div className="flex gap-3">
            <Button
              variant="success"
              fullWidth
              onClick={() =>
                selectedReceita && handleValidar(selectedReceita.id)
              }
            >
              ✅ Validar
            </Button>
            <Button
              variant="danger"
              fullWidth
              onClick={() =>
                selectedReceita && handleRejeitar(selectedReceita.id)
              }
            >
              ❌ Rejeitar
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default Receitas;
