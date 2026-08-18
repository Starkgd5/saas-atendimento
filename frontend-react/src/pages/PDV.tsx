import React, { useState, useEffect } from 'react';
import { Button } from '../components/ui/Button';
import { Input } from '../components/ui/Input';
import { Modal } from '../components/ui/Modal';

interface ProdutoVenda {
  id: number;
  codigo_barras: string;
  nome: string;
  preco: number;
  quantidade: number;
}

interface ItemCarrinho {
  id: number;
  produto_id: number;
  nome: string;
  quantidade: number;
  preco_unit: number;
  total: number;
  lote_id?: number;
}

export const PDV: React.FC = () => {
  const [carrinho, setCarrinho] = useState<ItemCarrinho[]>([]);
  const [codigoBarras, setCodigoBarras] = useState('');
  const [clienteId, setClienteId] = useState<number | null>(null);
  const [clienteNome, setClienteNome] = useState('');
  const [showReceitaModal, setShowReceitaModal] = useState(false);
  const [produtoControlado, setProdutoControlado] = useState<any>(null);
  const [total, setTotal] = useState(0);

  const handleAddProduct = async () => {
    if (!codigoBarras) return;

    try {
      const response = await fetch(`/api/v1/produtos/codigo/${codigoBarras}`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      const produto = await response.json();
      
      if (produto.requere_receita) {
        setProdutoControlado(produto);
        setShowReceitaModal(true);
        return;
      }

      addToCart(produto);
    } catch (error) {
      console.error('Produto não encontrado:', error);
    }
  };

  const addToCart = (produto: any) => {
    const existing = carrinho.find(item => item.produto_id === produto.id);
    
    if (existing) {
      setCarrinho(carrinho.map(item =>
        item.produto_id === produto.id
          ? { ...item, quantidade: item.quantidade + 1, total: item.total + produto.preco }
          : item
      ));
    } else {
      setCarrinho([...carrinho, {
        id: Date.now(),
        produto_id: produto.id,
        nome: produto.nome,
        quantidade: 1,
        preco_unit: produto.preco,
        total: produto.preco
      }]);
    }

    setCodigoBarras('');
    calcularTotal();
  };

  const calcularTotal = () => {
    const novoTotal = carrinho.reduce((acc, item) => acc + item.total, 0);
    setTotal(novoTotal);
  };

  const handleFinalizarVenda = async () => {
    try {
      const venda = {
        cliente_id: clienteId,
        itens: carrinho,
        total: total,
        forma_pagamento: 'dinheiro'
      };

      const response = await fetch('/api/v1/vendas', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify(venda)
      });

      if (response.ok) {
        setCarrinho([]);
        setTotal(0);
        alert('Venda finalizada com sucesso!');
      }
    } catch (error) {
      console.error('Erro ao finalizar venda:', error);
    }
  };

  return (
    <div className="flex h-screen">
      {/* Área de Produtos - Esquerda */}
      <div className="flex-1 p-4">
        <div className="flex gap-4 mb-4">
          <Input
            type="text"
            placeholder="Digite o código de barras ou nome"
            value={codigoBarras}
            onChange={(e) => setCodigoBarras(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleAddProduct()}
            className="flex-1"
          />
          <Button variant="primary" onClick={handleAddProduct}>
            🔍 Buscar
          </Button>
        </div>

        {/* Grid de Produtos */}
        <div className="grid grid-cols-3 gap-4">
          {/* Produtos serão carregados aqui */}
        </div>
      </div>

      {/* Carrinho - Direita */}
      <div className="w-96 bg-white border-l p-4 flex flex-col">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-bold">🛒 Carrinho</h2>
          <span className="text-sm text-gray-500">Cliente: {clienteNome || 'Não informado'}</span>
        </div>

        <div className="flex-1 overflow-y-auto">
          {carrinho.map((item) => (
            <div key={item.id} className="flex justify-between items-center py-2 border-b">
              <div>
                <div className="font-medium">{item.nome}</div>
                <div className="text-sm text-gray-500">
                  {item.quantidade} x R$ {item.preco_unit.toFixed(2)}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-bold">R$ {item.total.toFixed(2)}</span>
                <button 
                  className="text-red-500 hover:text-red-700"
                  onClick={() => {
                    setCarrinho(carrinho.filter(i => i.id !== item.id));
                    calcularTotal();
                  }}
                >
                  ✕
                </button>
              </div>
            </div>
          ))}
        </div>

        <div className="border-t pt-4 mt-4">
          <div className="flex justify-between items-center mb-4">
            <span className="text-lg font-bold">Total:</span>
            <span className="text-2xl font-bold text-green-600">
              R$ {total.toFixed(2)}
            </span>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <Button variant="success" fullWidth onClick={handleFinalizarVenda}>
              ✅ Finalizar
            </Button>
            <Button variant="danger" fullWidth>
              ❌ Cancelar
            </Button>
          </div>

          <Button variant="outline" fullWidth className="mt-2">
            👤 Buscar Cliente
          </Button>
        </div>
      </div>

      {/* Modal de Receita para Medicamentos Controlados */}
      {showReceitaModal && (
        <Modal
          isOpen={showReceitaModal}
          onClose={() => setShowReceitaModal(false)}
          title="⚠️ Medicamento Controlado"
        >
          <div className="p-4">
            <div className="bg-yellow-50 border border-yellow-200 p-4 rounded-lg mb-4">
              <p className="text-yellow-800">
                <strong>{produtoControlado?.nome}</strong> requer receita médica.
              </p>
              <p className="text-sm text-yellow-600">
                Anexe a receita ou informe que o cliente já possui.
              </p>
            </div>

            <div className="flex gap-4">
              <Button variant="primary" fullWidth onClick={() => {
                addToCart(produtoControlado);
                setShowReceitaModal(false);
              }}>
                📎 Anexar Receita e Adicionar
              </Button>
              <Button variant="secondary" fullWidth onClick={() => setShowReceitaModal(false)}>
                Cancelar
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};