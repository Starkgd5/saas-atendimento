package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type OrcamentoService struct {
	repo          *repository.OrcamentoRepository
	produtoRepo   *repository.ProdutoRepository
	clienteRepo   *repository.ClienteRepository
}

func NewOrcamentoService(repo *repository.OrcamentoRepository, produtoRepo *repository.ProdutoRepository, clienteRepo *repository.ClienteRepository) *OrcamentoService {
	return &OrcamentoService{
		repo:        repo,
		produtoRepo: produtoRepo,
		clienteRepo: clienteRepo,
	}
}

// CriarOrcamento cria um novo orçamento
func (s *OrcamentoService) CriarOrcamento(ctx context.Context, orcamento *models.Orcamento, itens []models.OrcamentoItem) error {
	if orcamento.ClienteID == 0 {
		return fmt.Errorf("cliente é obrigatório")
	}
	if orcamento.LojaID == 0 {
		return fmt.Errorf("loja é obrigatória")
	}
	if len(itens) == 0 {
		return fmt.Errorf("orçamento deve ter pelo menos um item")
	}

	// Verificar cliente
	cliente, err := s.clienteRepo.BuscarClientePorID(orcamento.ClienteID)
	if err != nil {
		return fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	if cliente == nil {
		return fmt.Errorf("cliente não encontrado")
	}

	// Calcular total
	var total float64
	for _, item := range itens {
		// Verificar produto
		produto, err := s.produtoRepo.BuscarProdutoPorID(item.ProdutoID, orcamento.LojaID)
		if err != nil {
			return fmt.Errorf("erro ao buscar produto: %w", err)
		}
		if produto == nil {
			return fmt.Errorf("produto %d não encontrado", item.ProdutoID)
		}
		if !produto.Ativo {
			return fmt.Errorf("produto %s está inativo", produto.Nome)
		}
		if produto.Estoque < item.Quantidade {
			return fmt.Errorf("estoque insuficiente para %s (disponível: %d)", produto.Nome, produto.Estoque)
		}

		item.ProdutoNome = produto.Nome
		item.PrecoUnit = produto.Preco
		item.Total = produto.Preco * float64(item.Quantidade)
		total += item.Total
	}

	orcamento.Total = total
	orcamento.TotalComDesconto = total - orcamento.Desconto
	orcamento.Status = models.OrcamentoPendente

	// Salvar orçamento
	if err := s.repo.CriarOrcamento(orcamento); err != nil {
		return err
	}

	// Salvar itens (simplificado - em produção, seria um batch insert)
	for _, item := range itens {
		item.OrcamentoID = orcamento.ID
		// TODO: Implementar inserção de itens
	}

	return nil
}

// BuscarOrcamentoPorID busca um orçamento pelo ID
func (s *OrcamentoService) BuscarOrcamentoPorID(ctx context.Context, id, lojaID int) (*models.Orcamento, error) {
	if id <= 0 {
		return nil, fmt.Errorf("ID inválido")
	}
	return s.repo.BuscarOrcamentoPorID(id, lojaID)
}

// ListarOrcamentos lista orçamentos com paginação
func (s *OrcamentoService) ListarOrcamentos(ctx context.Context, lojaID int, status string, clienteID *int, limit, offset int) ([]*models.Orcamento, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListarOrcamentos(lojaID, status, clienteID, limit, offset)
}

// AtualizarOrcamento atualiza um orçamento
func (s *OrcamentoService) AtualizarOrcamento(ctx context.Context, orcamento *models.Orcamento) error {
	if orcamento.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}

	existing, err := s.repo.BuscarOrcamentoPorID(orcamento.ID, orcamento.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar orçamento: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("orçamento não encontrado")
	}

	if existing.Status != models.OrcamentoPendente {
		return fmt.Errorf("orçamento não pode ser alterado (status: %s)", existing.Status)
	}

	return s.repo.AtualizarOrcamento(orcamento)
}

// AprovarOrcamento aprova um orçamento
func (s *OrcamentoService) AprovarOrcamento(ctx context.Context, id, lojaID int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}

	// Buscar orçamento
	orcamento, err := s.repo.BuscarOrcamentoPorID(id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar orçamento: %w", err)
	}
	if orcamento == nil {
		return fmt.Errorf("orçamento não encontrado")
	}

	if orcamento.Status != models.OrcamentoPendente {
		return fmt.Errorf("orçamento não pode ser aprovado (status: %s)", orcamento.Status)
	}

	return s.repo.AprovarOrcamento(id, lojaID)
}

// RejeitarOrcamento rejeita um orçamento
func (s *OrcamentoService) RejeitarOrcamento(ctx context.Context, id, lojaID int, motivo string) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}

	orcamento, err := s.repo.BuscarOrcamentoPorID(id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar orçamento: %w", err)
	}
	if orcamento == nil {
		return fmt.Errorf("orçamento não encontrado")
	}

	if orcamento.Status != models.OrcamentoPendente {
		return fmt.Errorf("orçamento não pode ser rejeitado (status: %s)", orcamento.Status)
	}

	orcamento.Observacao = motivo
	return s.repo.RejeitarOrcamento(id, lojaID)
}

// ExpirarOrcamentosExpirados expira orçamentos pendentes com mais de 7 dias
func (s *OrcamentoService) ExpirarOrcamentosExpirados(ctx context.Context) error {
	// TODO: Implementar expiração em lote
	return nil
}

// GetMetricasOrcamento retorna métricas de orçamentos
func (s *OrcamentoService) GetMetricasOrcamento(ctx context.Context, lojaID int) (map[string]interface{}, error) {
	totalMes, err := s.repo.CalcularTotalOrcamentosMes(lojaID)
	if err != nil {
		return nil, err
	}

	aprovados, err := s.repo.ContarOrcamentosPorStatus(lojaID, models.OrcamentoAprovado)
	if err != nil {
		return nil, err
	}

	pendentes, err := s.repo.ContarOrcamentosPorStatus(lojaID, models.OrcamentoPendente)
	if err != nil {
		return nil, err
	}

	rejeitados, err := s.repo.ContarOrcamentosPorStatus(lojaID, models.OrcamentoRejeitado)
	if err != nil {
		return nil, err
	}

	total := aprovados + pendentes + rejeitados
	taxaAprovacao := 0.0
	if total > 0 {
		taxaAprovacao = float64(aprovados) / float64(total) * 100
	}

	return map[string]interface{}{
		"total":          total,
		"aprovados":      aprovados,
		"pendentes":      pendentes,
		"rejeitados":     rejeitados,
		"taxa_aprovacao": taxaAprovacao,
		"total_mes":      totalMes,
	}, nil
}