import React, { useState, useEffect } from "react";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";

interface Configuracao {
  chave: string;
  valor: string;
  descricao: string;
}

const Configuracoes: React.FC = () => {
  const [configs, setConfigs] = useState<Configuracao[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const { get, put } = useApi();
  const { toasts, removeToast, success, error: showError } = useToast();

  useEffect(() => {
    loadConfigs();
  });

  const loadConfigs = async () => {
    try {
      const data = await get("/configuracoes");
      const configArray = Object.entries(data).map(([chave, valor]) => ({
        chave,
        valor: valor as string,
        descricao: getDescricao(chave),
      }));
      setConfigs(configArray);
    } catch (err: any) {
      showError(err.message || "Erro ao carregar configurações");
    } finally {
      setLoading(false);
    }
  };

  const getDescricao = (chave: string): string => {
    const descricoes: Record<string, string> = {
      max_clientes_fila: "Número máximo de clientes em atendimento simultâneo",
      limite_orcamento_auto: "Limite para orçamento automático (R$)",
      whatsapp_phone_id: "ID do telefone WhatsApp",
      tempo_espera_maximo: "Tempo máximo de espera (segundos)",
    };
    return descricoes[chave] || "";
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = configs.reduce(
        (acc, config) => {
          acc[config.chave] = config.valor;
          return acc;
        },
        {} as Record<string, string>,
      );

      await put("/configuracoes", payload);
      success("Configurações salvas com sucesso!");
    } catch (err: any) {
      showError(err.message || "Erro ao salvar configurações");
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl animate-spin mb-4">⏳</div>
          <p className="text-gray-600">Carregando configurações...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
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
          <h1 className="text-2xl font-bold text-gray-800">⚙️ Configurações</h1>
          <p className="text-gray-500 text-sm mt-1">
            Configure as preferências do sistema
          </p>
        </div>
      </div>

      <Card>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSave();
          }}
          className="space-y-4"
        >
          {configs.map((config) => (
            <div key={config.chave}>
              <Input
                label={config.chave.replace(/_/g, " ").toUpperCase()}
                value={config.valor}
                onChange={(e) => {
                  setConfigs(
                    configs.map((c) =>
                      c.chave === config.chave
                        ? { ...c, valor: e.target.value }
                        : c,
                    ),
                  );
                }}
                placeholder={config.descricao}
              />
              <p className="text-xs text-gray-400 mt-1">{config.descricao}</p>
            </div>
          ))}

          <div className="flex gap-3 pt-4 border-t border-gray-200">
            <Button type="submit" loading={saving} variant="primary" icon="💾">
              Salvar Configurações
            </Button>
            <Button type="button" variant="secondary" onClick={loadConfigs}>
              🔄 Reverter
            </Button>
          </div>
        </form>
      </Card>

      <div className="mt-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
        <h4 className="text-sm font-medium text-yellow-800">⚠️ Atenção</h4>
        <p className="text-sm text-yellow-700 mt-1">
          Alterar estas configurações pode afetar o funcionamento do sistema.
          Certifique-se de entender o impacto de cada alteração.
        </p>
      </div>
    </div>
  );
};

export default Configuracoes;
