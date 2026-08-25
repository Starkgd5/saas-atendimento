import React, { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../components/ui/Button";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";
import FormatterService from "../services/formatter.service";

interface Compra {
  id: number;
  numero_nf: string;
  fornecedor: string;
  total: number;
  status: string;
  data_emissao: string;
}

const Compras: React.FC = () => {
  const [compras, setCompras] = useState<Compra[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const { get } = useApi();
  const { toasts, removeToast, error: showError } = useToast();
  const formatCurrency = FormatterService.formatCurrency;
  const formatDate = FormatterService.formatDate;

  const loadCompras = useCallback(async () => {
    try {
      const data = await get("/compras");
      setCompras(data || []);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar compras");
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    loadCompras();
  }, [loadCompras]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando compras...</p>
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
          <h1 className="text-2xl font-bold text-gray-800">📦 Compras</h1>
          <p className="text-gray-500 text-sm mt-1">
            Histórico de compras realizadas
          </p>
        </div>
        <Button
          variant="primary"
          onClick={() => navigate("/compras/nova")}
          icon="➕"
        >
          Nova Compra
        </Button>
      </div>

      <div className="bg-white rounded-xl shadow-md overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                NF
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                Fornecedor
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                Total
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
            {compras.map((compra) => (
              <tr key={compra.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 text-sm font-medium text-gray-800">
                  {compra.numero_nf}
                </td>
                <td className="px-6 py-4 text-sm text-gray-600">
                  {compra.fornecedor}
                </td>
                <td className="px-6 py-4 text-sm font-semibold text-gray-800">
                  {formatCurrency(compra.total)}
                </td>
                <td className="px-6 py-4">
                  <span
                    className={`px-2 py-1 text-xs rounded-full ${
                      compra.status === "Recebido"
                        ? "bg-green-100 text-green-700"
                        : compra.status === "Em Andamento"
                          ? "bg-yellow-100 text-yellow-700"
                          : "bg-gray-100 text-gray-700"
                    }`}
                  >
                    {compra.status}
                  </span>
                </td>
                <td className="px-6 py-4 text-sm text-gray-500">
                  {formatDate(compra.data_emissao)}
                </td>
                <td className="px-6 py-4 text-right">
                  <Button variant="outline" size="sm">
                    Detalhes
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default Compras;
