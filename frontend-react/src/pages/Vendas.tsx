import React, { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";
import FormatterService from "../services/formatter.service";

interface Venda {
  id: number;
  numero_venda: string;
  cliente: string;
  total: number;
  status: string;
  tipo_pagamento: string;
  created_at: string;
  itens?: any[];
}

const Vendas: React.FC = () => {
  const [vendas, setVendas] = useState<Venda[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("todos");
  const [searchTerm, setSearchTerm] = useState("");
  const navigate = useNavigate();
  const { get } = useApi();
  const { toasts, removeToast, error: showError } = useToast();
  const formatCurrency = FormatterService.formatCurrency;
  const formatDateTime = FormatterService.formatDateTime;

  const loadVendas = useCallback(async () => {
    try {
      let url = "/vendas";
      if (filter !== "todos") {
        url += `?status=${filter}`;
      }
      const data = await get(url);
      setVendas(data.items || []);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar vendas");
    } finally {
      setLoading(false);
    }
  }, [filter, get, showError]);

  useEffect(() => {
    loadVendas();
  }, [loadVendas]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case "Pago":
        return "bg-green-100 text-green-700";
      case "Pendente":
        return "bg-yellow-100 text-yellow-700";
      case "Cancelado":
        return "bg-red-100 text-red-700";
      default:
        return "bg-gray-100 text-gray-700";
    }
  };

  const filteredVendas = vendas.filter(
    (v) =>
      v.numero_venda.toLowerCase().includes(searchTerm.toLowerCase()) ||
      v.cliente?.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando vendas...</p>
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
          <h1 className="text-2xl font-bold text-gray-800">📋 Vendas</h1>
          <p className="text-gray-500 text-sm mt-1">
            Histórico de todas as vendas realizadas
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="primary" onClick={() => navigate("/pdv")} icon="🛒">
            Nova Venda
          </Button>
        </div>
      </div>

      <Card className="mb-6">
        <div className="flex flex-wrap gap-4">
          <Input
            type="text"
            placeholder="Buscar por número ou cliente..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-64"
          />
          <div className="flex gap-2">
            {["todos", "Pago", "Pendente", "Cancelado"].map((status) => (
              <Button
                key={status}
                variant={filter === status ? "primary" : "secondary"}
                size="sm"
                onClick={() => setFilter(status)}
              >
                {status === "todos" ? "Todos" : status}
              </Button>
            ))}
          </div>
          <Button variant="outline" size="sm" onClick={loadVendas} icon="🔄">
            Atualizar
          </Button>
        </div>
      </Card>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Nº Venda
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Cliente
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Total
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Pagamento
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                  Data
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">
                  Ações
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {filteredVendas.length === 0 ? (
                <tr>
                  <td
                    colSpan={7}
                    className="px-6 py-8 text-center text-gray-400"
                  >
                    Nenhuma venda encontrada
                  </td>
                </tr>
              ) : (
                filteredVendas.map((venda) => (
                  <tr key={venda.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm font-medium text-gray-800">
                      {venda.numero_venda}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {venda.cliente || "Cliente não identificado"}
                    </td>
                    <td className="px-6 py-4 text-sm font-semibold text-gray-800">
                      {formatCurrency(venda.total)}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {venda.tipo_pagamento || "Não informado"}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`px-2 py-1 text-xs rounded-full ${getStatusColor(venda.status)}`}
                      >
                        {venda.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {formatDateTime(venda.created_at)}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => navigate(`/vendas/${venda.id}`)}
                      >
                        Ver
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="mt-4 text-sm text-gray-500 flex justify-between">
        <span>Total: {filteredVendas.length} vendas</span>
        <span>
          Valor total:{" "}
          {formatCurrency(
            filteredVendas.reduce(
              (acc, v) => acc + (v.status === "Pago" ? v.total : 0),
              0,
            ),
          )}
        </span>
      </div>
    </div>
  );
};

export default Vendas;
