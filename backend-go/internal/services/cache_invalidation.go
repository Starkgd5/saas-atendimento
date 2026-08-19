package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/cache"
)

type CacheInvalidationService struct {
	cacheService *cache.CacheService
}

func NewCacheInvalidationService(cacheService *cache.CacheService) *CacheInvalidationService {
	return &CacheInvalidationService{
		cacheService: cacheService,
	}
}

// ============================================
// INVALIDAÇÃO POR EVENTO
// ============================================

// InvalidateProduto invalida caches relacionados a produtos
func (s *CacheInvalidationService) InvalidateProduto(ctx context.Context, produtoID int, lojaID int) {
	patterns := []string{
		fmt.Sprintf("*produtos*loja:%d*", lojaID),
		fmt.Sprintf("*produto:%d*", produtoID),
		"*dashboard*",
		"*estoque*",
		"*orcamentos*",
	}

	for _, pattern := range patterns {
		s.cacheService.ClearByPattern(ctx, pattern)
	}
}

// InvalidateVenda invalida caches relacionados a vendas
func (s *CacheInvalidationService) InvalidateVenda(ctx context.Context, lojaID int) {
	patterns := []string{
		"*dashboard*",
		"*vendas*",
		"*relatorio*",
		"*faturamento*",
	}

	for _, pattern := range patterns {
		s.cacheService.ClearByPattern(ctx, pattern)
	}
}

// InvalidateCliente invalida caches relacionados a clientes
func (s *CacheInvalidationService) InvalidateCliente(ctx context.Context, clienteID int, lojaID int) {
	patterns := []string{
		fmt.Sprintf("*clientes*loja:%d*", lojaID),
		fmt.Sprintf("*cliente:%d*", clienteID),
		"*dashboard*",
	}

	for _, pattern := range patterns {
		s.cacheService.ClearByPattern(ctx, pattern)
	}
}

// InvalidateDashboard invalida caches do dashboard
func (s *CacheInvalidationService) InvalidateDashboard(ctx context.Context, lojaID int) {
	patterns := []string{
		"*dashboard*",
		"*metricas*",
	}

	for _, pattern := range patterns {
		s.cacheService.ClearByPattern(ctx, pattern)
	}
}

// InvalidateAll invalida todos os caches (use com cuidado)
func (s *CacheInvalidationService) InvalidateAll(ctx context.Context) {
	s.cacheService.ClearByPattern(ctx, "*")
}

// ============================================
// INVALIDAÇÃO POR PADRÃO
// ============================================

// InvalidateByPattern invalida caches que correspondem a um padrão
func (s *CacheInvalidationService) InvalidateByPattern(ctx context.Context, pattern string) error {
	return s.cacheService.ClearByPattern(ctx, pattern)
}

// InvalidateByPrefix invalida caches com um prefixo específico
func (s *CacheInvalidationService) InvalidateByPrefix(ctx context.Context, prefix string) error {
	return s.cacheService.ClearByPattern(ctx, prefix+"*")
}
