import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import Card from "../components/ui/Card";
import { useApi } from "../hooks/useApi";
import { useToast } from "../hooks/useToast";
import Toast from "../components/ui/Toast";

interface Produto {
  id: number;
  nome: string;
  codigo_barras: string;
  categoria: string;
  preco_venda: number;
  preco_custo: number;
}

interface LoteForm {
  produto_id: string;
  numero_lote: string;
  data_fabricacao: string;
  data_validade: string;
  quantidade: string;
  preco_custo: string;
  preco_venda: string;
}

const EntradaEstoque: React.FC = () => {
  const [produtos, setProdutos] = useState<Produto[]>([]);
  const [form, setForm] = useState<LoteForm>({
    produto_id: "",
    numero_lote: "",
    data_fabricacao: "",
    data_validade: "",
    quantidade: "",
    preco_custo: "",
    preco_venda: "",
  });
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const navigate = useNavigate();
  const { get, post } = useApi();
  const { toasts, removeToast, success, error: showError } = useToast();

  useEffect(() => {
    loadProdutos();
  });

  const loadProdutos = async () => {
    try {
      const data = await get("/produtos");
      setProdutos(data.items || []);
    } catch (err) {
      showError("Erro ao carregar produtos");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (
      !form.produto_id ||
      !form.numero_lote ||
      !form.data_validade ||
      !form.quantidade
    ) {
      showError("Preencha todos os campos obrigatórios");
      return;
    }

    setLoading(true);
    try {
      await post("/estoque/entrada", {
        produto_id: parseInt(form.produto_id),
        numero_lote: form.numero_lote,
        data_fabricacao:
          form.data_fabricacao || new Date().toISOString().split("T")[0],
        data_validade: form.data_validade,
        quantidade: parseInt(form.quantidade),
        preco_custo: parseFloat(form.preco_custo) || 0,
        preco_venda: parseFloat(form.preco_venda) || 0,
      });

      success("Entrada de estoque realizada com sucesso!");
      setForm({
        produto_id: "",
        numero_lote: "",
        data_fabricacao: "",
        data_validade: "",
        quantidade: "",
        preco_custo: "",
        preco_venda: "",
      });
      setTimeout(() => navigate("/estoque"), 1500);
    } catch (err: any) {
      showError(err.message || "Erro ao realizar entrada de estoque");
    } finally {
      setLoading(false);
    }
  };

  const filteredProdutos = produtos.filter(
    (p) =>
      p.nome.toLowerCase().includes(searchTerm.toLowerCase()) ||
      p.codigo_barras.includes(searchTerm),
  );

  return (
    <div className="max-w-4xl mx-auto">
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
          <h1 className="text-2xl font-bold text-gray-800">
            📥 Entrada de Estoque
          </h1>
          <p className="text-gray-500 text-sm mt-1">
            Registre a entrada de novos lotes de produtos
          </p>
        </div>
        <Button variant="outline" onClick={() => navigate("/estoque")} icon="←">
          Voltar
        </Button>
      </div>

      <Card>
        <form onSubmit={handleSubmit} className="space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Produto *
            </label>
            <div className="relative">
              <Input
                type="text"
                placeholder="Buscar produto por nome ou código de barras..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full"
              />
              {searchTerm && filteredProdutos.length > 0 && (
                <div className="absolute z-10 w-full bg-white border border-gray-200 rounded-lg shadow-lg mt-1 max-h-60 overflow-y-auto">
                  {filteredProdutos.map((produto) => (
                    <div
                      key={produto.id}
                      className="px-4 py-2 hover:bg-gray-50 cursor-pointer flex justify-between"
                      onClick={() => {
                        setForm({ ...form, produto_id: String(produto.id) });
                        setSearchTerm(produto.nome);
                      }}
                    >
                      <span className="font-medium">{produto.nome}</span>
                      <span className="text-gray-500 text-sm">
                        {produto.codigo_barras}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            {form.produto_id && (
              <p className="text-sm text-green-600 mt-1">
                ✓ Produto selecionado:{" "}
                {produtos.find((p) => p.id === parseInt(form.produto_id))?.nome}
              </p>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Número do Lote *"
              value={form.numero_lote}
              onChange={(e) =>
                setForm({ ...form, numero_lote: e.target.value })
              }
              placeholder="LOTE001"
              required
            />
            <Input
              label="Data de Fabricação"
              type="date"
              value={form.data_fabricacao}
              onChange={(e) =>
                setForm({ ...form, data_fabricacao: e.target.value })
              }
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Data de Validade *"
              type="date"
              value={form.data_validade}
              onChange={(e) =>
                setForm({ ...form, data_validade: e.target.value })
              }
              required
            />
            <Input
              label="Quantidade *"
              type="number"
              value={form.quantidade}
              onChange={(e) => setForm({ ...form, quantidade: e.target.value })}
              placeholder="0"
              required
              min="1"
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Preço de Custo (R$)"
              type="number"
              step="0.01"
              value={form.preco_custo}
              onChange={(e) =>
                setForm({ ...form, preco_custo: e.target.value })
              }
              placeholder="0,00"
            />
            <Input
              label="Preço de Venda (R$)"
              type="number"
              step="0.01"
              value={form.preco_venda}
              onChange={(e) =>
                setForm({ ...form, preco_venda: e.target.value })
              }
              placeholder="0,00"
            />
          </div>

          <div className="flex gap-3 pt-4 border-t border-gray-200">
            <Button type="submit" loading={loading} variant="success" icon="📥">
              Registrar Entrada
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setForm({
                  produto_id: "",
                  numero_lote: "",
                  data_fabricacao: "",
                  data_validade: "",
                  quantidade: "",
                  preco_custo: "",
                  preco_venda: "",
                });
                setSearchTerm("");
              }}
            >
              Limpar
            </Button>
          </div>
        </form>
      </Card>

      <div className="mt-4 text-sm text-gray-500">
        <p>* Campos obrigatórios</p>
        <p className="mt-1">
          ⚠️ Certifique-se de que o produto já está cadastrado antes de
          registrar a entrada.
        </p>
      </div>
    </div>
  );
};

export default EntradaEstoque;
