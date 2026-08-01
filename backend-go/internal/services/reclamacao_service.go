package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type ReclamacaoService struct {
	repo        *repository.ReclamacaoRepository
	clienteRepo *repository.ClienteRepository
}

func NewReclamacaoService(repo *repository.ReclamacaoRepository, clienteRepo *repository.ClienteRepository) *ReclamacaoService {
	return &ReclamacaoService{
		repo:        repo,
		clienteRepo: clienteRepo,
	}
}

// CriarReclamacao cria uma nova reclamação
func (s *ReclamacaoService) CriarReclamacao(ctx context.Context, reclamacao *models.Reclamacao) error {
	if reclamacao.ClienteID == 0 {
		return fmt.Errorf("cliente é obrigatório")
	}
	if reclamacao.Mensagem == "" {
		return fmt.Errorf("mensagem é obrigatória")
	}
	if reclamacao.LojaID == 0 {
		return fmt.Errorf("loja é obrigatória")
	}

	// Verificar cliente
	cliente, err := s.clienteRepo.BuscarClientePorID(reclamacao.ClienteID)
	if err != nil {
		return fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	if cliente == nil {
		return fmt.Errorf("cliente não encontrado")
	}

	// Definir status e prioridade padrão
	if reclamacao.Status == "" {
		reclamacao.Status = models.ReclamacaoPendente
	}
	if reclamacao.Prioridade == "" {
		reclamacao.Prioridade = models.PrioridadeMedia
	}

	return s.repo.CriarReclamacao(reclamacao)
}

// BuscarReclamacaoPorID busca uma reclamação pelo ID
func (s *ReclamacaoService) BuscarReclamacaoPorID(ctx context.Context, id, lojaID int) (*models.Reclamacao, error) {
	if id <= 0 {
		return nil, fmt.Errorf("ID inválido")
	}
	return s.repo.BuscarReclamacaoPorID(id, lojaID)
}

// ListarReclamacoes lista reclamações com paginação
func (s *ReclamacaoService) ListarReclamacoes(ctx context.Context, lojaID int, status string, prioridade string, limit, offset int) ([]*models.Reclamacao, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListarReclamacoes(lojaID, status, prioridade, limit, offset)
}

// ListarReclamacoesPendentes lista reclamações pendentes
func (s *ReclamacaoService) ListarReclamacoesPendentes(ctx context.Context, lojaID int) ([]*models.Reclamacao, error) {
	return s.repo.ListarReclamacoesPendentes(lojaID)
}

// AtualizarReclamacao atualiza uma reclamação
func (s *ReclamacaoService) AtualizarReclamacao(ctx context.Context, reclamacao *models.Reclamacao) error {
	if reclamacao.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}

	existing, err := s.repo.BuscarReclamacaoPorID(reclamacao.ID, reclamacao.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar reclamação: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("reclamação não encontrada")
	}

	return s.repo.AtualizarReclamacao(reclamacao)
}

// ResolverReclamacao resolve uma reclamação
func (s *ReclamacaoService) ResolverReclamacao(ctx context.Context, id, lojaID int, resposta string) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}
	if resposta == "" {
		return fmt.Errorf("resposta é obrigatória")
	}

	existing, err := s.repo.BuscarReclamacaoPorID(id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar reclamação: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("reclamação não encontrada")
	}

	if existing.Status == models.ReclamacaoResolvido {
		return fmt.Errorf("reclamação já foi resolvida")
	}

	return s.repo.ResolverReclamacao(id, lojaID, resposta)
}

// IgnorarReclamacao ignora uma reclamação
func (s *ReclamacaoService) IgnorarReclamacao(ctx context.Context, id, lojaID int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}

	existing, err := s.repo.BuscarReclamacaoPorID(id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar reclamação: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("reclamação não encontrada")
	}

	if existing.Status == models.ReclamacaoResolvido {
		return fmt.Errorf("reclamação já foi resolvida")
	}

	return s.repo.IgnorarReclamacao(id, lojaID)
}

// GetMetricasReclamacao retorna métricas de reclamações
func (s *ReclamacaoService) GetMetricasReclamacao(ctx context.Context, lojaID int) (map[string]interface{}, error) {
	pendentes, err := s.repo.ContarReclamacoesPendentes(lojaID)
	if err != nil {
		return nil, err
	}

	resolvidas, err := s.repo.ContarReclamacoesPorStatus(lojaID, models.ReclamacaoResolvido)
	if err != nil {
		return nil, err
	}

	ignoradas, err := s.repo.ContarReclamacoesPorStatus(lojaID, models.ReclamacaoIgnorado)
	if err != nil {
		return nil, err
	}

	criticas, err := s.repo.ContarReclamacoesPorPrioridade(lojaID, models.PrioridadeCritica)
	if err != nil {
		return nil, err
	}

	altas, err := s.repo.ContarReclamacoesPorPrioridade(lojaID, models.PrioridadeAlta)
	if err != nil {
		return nil, err
	}

	total := pendentes + resolvidas + ignoradas
	taxaResolucao := 0.0
	if total > 0 {
		taxaResolucao = float64(resolvidas) / float64(total) * 100
	}

	return map[string]interface{}{
		"pendentes":      pendentes,
		"resolvidas":     resolvidas,
		"ignoradas":      ignoradas,
		"total":          total,
		"taxa_resolucao": taxaResolucao,
		"criticas":       criticas,
		"altas":          altas,
	}, nil
}