import React, { useState, useEffect, useCallback } from "react";
import Card from "../components/ui/Card";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";
import FormatterService from "../services/formatter.service";

interface Conta {
  id: number;
  descricao: string;
  valor: number;
  data_vencimento: string;
  status: string;
  tipo: "receber" | "pagar";
}

const Financeiro: React.FC = () => {
  const [contasReceber, setContasReceber] = useState<Conta[]>([]);
  const [contasPagar, setContasPagar] = useState<Conta[]>([]);
  const [loading, setLoading] = useState(true);
  const { get } = useApi();
  const { toasts, removeToast, error: showError } = useToast();
  const formatCurrency = FormatterService.formatCurrency;
  const formatDate = FormatterService.formatDate;

  const loadFinanceiro = useCallback(async () => {
    try {
      const [receber, pagar] = await Promise.all([
        get("/financeiro/contas-receber"),
        get("/financeiro/contas-pagar"),
      ]);
      setContasReceber(receber || []);
      setContasPagar(pagar || []);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar financeiro");
    } finally {
      setLoading(false);
    }
  }, [get, showError]);

  useEffect(() => {
    loadFinanceiro();
  }, [loadFinanceiro]);

  const totalReceber = contasReceber.reduce(
    (acc, c) => acc + (c.status === "Pendente" ? c.valor : 0),
    0,
  );
  const totalPagar = contasPagar.reduce(
    (acc, c) => acc + (c.status === "Pendente" ? c.valor : 0),
    0,
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando financeiro...</p>
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
          <h1 className="text-2xl font-bold text-gray-800">💰 Financeiro</h1>
          <p className="text-gray-500 text-sm mt-1">
            Contas a receber e a pagar
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <Card
          title="💰 Contas a Receber"
          className="border-t-4 border-t-green-500"
        >
          <div className="text-3xl font-bold text-green-600 mb-4">
            {formatCurrency(totalReceber)}
          </div>
          <div className="space-y-2">
            {contasReceber.map((conta) => (
              <div
                key={conta.id}
                className="flex justify-between items-center py-2 border-b border-gray-100"
              >
                <div>
                  <p className="text-sm font-medium text-gray-800">
                    {conta.descricao}
                  </p>
                  <p className="text-xs text-gray-400">
                    Vence: {formatDate(conta.data_vencimento)}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-gray-800">
                    {formatCurrency(conta.valor)}
                  </p>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${
                      conta.status === "Pago"
                        ? "bg-green-100 text-green-700"
                        : "bg-yellow-100 text-yellow-700"
                    }`}
                  >
                    {conta.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>

        <Card title="📤 Contas a Pagar" className="border-t-4 border-t-red-500">
          <div className="text-3xl font-bold text-red-600 mb-4">
            {formatCurrency(totalPagar)}
          </div>
          <div className="space-y-2">
            {contasPagar.map((conta) => (
              <div
                key={conta.id}
                className="flex justify-between items-center py-2 border-b border-gray-100"
              >
                <div>
                  <p className="text-sm font-medium text-gray-800">
                    {conta.descricao}
                  </p>
                  <p className="text-xs text-gray-400">
                    Vence: {formatDate(conta.data_vencimento)}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold text-gray-800">
                    {formatCurrency(conta.valor)}
                  </p>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${
                      conta.status === "Pago"
                        ? "bg-green-100 text-green-700"
                        : "bg-yellow-100 text-yellow-700"
                    }`}
                  >
                    {conta.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      </div>

      <Card title="📊 Resumo Financeiro">
        <div className="grid grid-cols-2 gap-4">
          <div className="text-center p-4 bg-green-50 rounded-lg">
            <p className="text-sm text-gray-600">Total a Receber</p>
            <p className="text-2xl font-bold text-green-600">
              {formatCurrency(totalReceber)}
            </p>
          </div>
          <div className="text-center p-4 bg-red-50 rounded-lg">
            <p className="text-sm text-gray-600">Total a Pagar</p>
            <p className="text-2xl font-bold text-red-600">
              {formatCurrency(totalPagar)}
            </p>
          </div>
          <div className="text-center p-4 bg-blue-50 rounded-lg col-span-2">
            <p className="text-sm text-gray-600">Saldo Previsto</p>
            <p
              className={`text-2xl font-bold ${totalReceber - totalPagar >= 0 ? "text-green-600" : "text-red-600"}`}
            >
              {formatCurrency(totalReceber - totalPagar)}
            </p>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default Financeiro;
