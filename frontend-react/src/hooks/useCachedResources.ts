import { useCachedApi } from './useCachedApi';
import { useCachedMutation } from './useCachedMutation';

// ============================================
// PRODUTOS
// ============================================

export function useProdutos(params?: Record<string, any>) {
  return useCachedApi('/produtos', params, {
    cacheKey: `produtos:${JSON.stringify(params)}`,
    ttl: 5 * 60 * 1000, // 5 minutos
    staleTime: 60 * 1000, // 1 minuto
  });
}

export function useProduto(id: number) {
  return useCachedApi(`/produtos/${id}`, undefined, {
    cacheKey: `produto:${id}`,
    ttl: 5 * 60 * 1000,
  });
}

export function useProdutoMutation() {
  return useCachedMutation('/produtos', 'POST', {
    invalidatePatterns: ['produtos', 'dashboard', 'estoque'],
    onSuccess: (data) => {
      console.log('Produto criado com sucesso:', data);
    },
  });
}

export function useProdutoUpdate(id: number) {
  return useCachedMutation(`/produtos/${id}`, 'PUT', {
    invalidateKeys: [`produto:${id}`],
    invalidatePatterns: ['produtos', 'dashboard', 'estoque'],
    onSuccess: () => {
      console.log('Produto atualizado com sucesso');
    },
  });
}

// ============================================
// VENDAS
// ============================================

export function useVendas(params?: Record<string, any>) {
  return useCachedApi('/vendas', params, {
    cacheKey: `vendas:${JSON.stringify(params)}`,
    ttl: 1 * 60 * 1000, // 1 minuto
  });
}

export function useVenda(id: number) {
  return useCachedApi(`/vendas/${id}`, undefined, {
    cacheKey: `venda:${id}`,
    ttl: 1 * 60 * 1000,
  });
}

export function useVendaMutation() {
  return useCachedMutation('/vendas', 'POST', {
    invalidatePatterns: ['vendas', 'dashboard', 'faturamento', 'relatorio'],
    onSuccess: (data) => {
      console.log('Venda realizada com sucesso:', data);
    },
  });
}

// ============================================
// DASHBOARD
// ============================================

export function useDashboard() {
  return useCachedApi('/dashboard', undefined, {
    cacheKey: 'dashboard',
    ttl: 30 * 1000, // 30 segundos
    staleTime: 10 * 1000, // 10 segundos
  });
}

export function useDashboardFarmacia() {
  return useCachedApi('/dashboard/farmacia', undefined, {
    cacheKey: 'dashboard:farmacia',
    ttl: 30 * 1000,
  });
}

// ============================================
// ESTOQUE
// ============================================

export function useLotes(produtoId: number) {
  return useCachedApi(`/estoque/lotes/${produtoId}`, undefined, {
    cacheKey: `lotes:${produtoId}`,
    ttl: 1 * 60 * 1000,
  });
}

export function useAlertasEstoque() {
  return useCachedApi('/estoque/alertas', undefined, {
    cacheKey: 'estoque:alertas',
    ttl: 1 * 60 * 1000,
  });
}

// ============================================
// CLIENTES
// ============================================

export function useClientes(params?: Record<string, any>) {
  return useCachedApi('/clientes', params, {
    cacheKey: `clientes:${JSON.stringify(params)}`,
    ttl: 5 * 60 * 1000,
  });
}

export function useCliente(id: number) {
  return useCachedApi(`/clientes/${id}`, undefined, {
    cacheKey: `cliente:${id}`,
    ttl: 5 * 60 * 1000,
  });
}

// ============================================
// ORÇAMENTOS
// ============================================

export function useOrcamentos(params?: Record<string, any>) {
  return useCachedApi('/orcamentos', params, {
    cacheKey: `orcamentos:${JSON.stringify(params)}`,
    ttl: 5 * 60 * 1000,
  });
}

export function useOrcamento(id: number) {
  return useCachedApi(`/orcamentos/${id}`, undefined, {
    cacheKey: `orcamento:${id}`,
    ttl: 5 * 60 * 1000,
  });
}