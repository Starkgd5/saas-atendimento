package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type AtendimentoService struct {
	repo          *repository.AtendimentoRepository
	clienteRepo   *repository.ClienteRepository
	orcamentoRepo *repository.OrcamentoRepository
	filaService   *FilaService
	db            *sql.DB
}

func NewAtendimentoService(
	repo *repository.AtendimentoRepository,
	clienteRepo *repository.ClienteRepository,
	orcamentoRepo *repository.OrcamentoRepository,
	filaService *FilaService,
	db *sql.DB,
) *AtendimentoService {
	return &AtendimentoService{
		repo:          repo,
		clienteRepo:   clienteRepo,
		orcamentoRepo: orcamentoRepo,
		filaService:   filaService,
		db:            db,
	}
}

// ============================================
// CRIAÇÃO E GESTÃO
// ============================================

// CriarAtendimento cria um novo atendimento e adiciona à fila
func (s *AtendimentoService) CriarAtendimento(ctx context.Context, clienteID, lojaID int) (*models.Atendimento, error) {
	// Verificar se cliente existe
	cliente, err := s.clienteRepo.BuscarClientePorID(clienteID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	if cliente == nil {
		return nil, fmt.Errorf("cliente não encontrado")
	}

	// Verificar se cliente já tem atendimento ativo
	atendimentos, err := s.repo.BuscarAtendimentosAtivos(lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos ativos: %w", err)
	}
	for _, a := range atendimentos {
		if a.ClienteID == clienteID {
			return nil, fmt.Errorf("cliente já tem atendimento ativo")
		}
	}

	// Criar atendimento
	atendimento := &models.Atendimento{
		ClienteID:  clienteID,
		LojaID:     lojaID,
		Status:     models.StatusAguardando,
		IniciadoEm: time.Now(),
	}

	if err := s.repo.CriarAtendimento(atendimento); err != nil {
		return nil, fmt.Errorf("erro ao criar atendimento: %w", err)
	}

	// Adicionar à fila
	if err := s.filaService.AdicionarClienteFila(ctx, clienteID, lojaID); err != nil {
		// Não falha a criação do atendimento, apenas loga
		_ = err
	}

	return atendimento, nil
}

// IniciarAtendimento inicia um atendimento (puxa da fila)
func (s *AtendimentoService) IniciarAtendimento(ctx context.Context, atendimentoID, atendenteID int) error {
	// Buscar atendimento
	atendimento, err := s.repo.BuscarAtendimentoPorID(atendimentoID)
	if err != nil {
		return fmt.Errorf("erro ao buscar atendimento: %w", err)
	}
	if atendimento == nil {
		return fmt.Errorf("atendimento não encontrado")
	}

	if atendimento.Status != models.StatusAguardando {
		return fmt.Errorf("atendimento não está aguardando")
	}

	// Iniciar
	if err := s.repo.IniciarAtendimento(atendimentoID, atendenteID); err != nil {
		return fmt.Errorf("erro ao iniciar atendimento: %w", err)
	}

	// Atualizar último atendimento do cliente
	if err := s.clienteRepo.AtualizarUltimoAtendimento(atendimento.ClienteID); err != nil {
		_ = err // Não falha a operação
	}

	return nil
}

// FinalizarAtendimento finaliza um atendimento
func (s *AtendimentoService) FinalizarAtendimento(ctx context.Context, atendimentoID int) error {
	// Buscar atendimento
	atendimento, err := s.repo.BuscarAtendimentoPorID(atendimentoID)
	if err != nil {
		return fmt.Errorf("erro ao buscar atendimento: %w", err)
	}
	if atendimento == nil {
		return fmt.Errorf("atendimento não encontrado")
	}

	if atendimento.Status != models.StatusEmAtendimento {
		return fmt.Errorf("atendimento não está em andamento")
	}

	// Finalizar no banco
	if err := s.repo.FinalizarAtendimento(atendimentoID); err != nil {
		return fmt.Errorf("erro ao finalizar atendimento: %w", err)
	}

	// Finalizar na fila
	if err := s.filaService.FinalizarAtendimento(ctx, atendimento.ClienteID); err != nil {
		_ = err // Não falha a operação
	}

	// Tentar puxar próximo cliente automaticamente
	go func() {
		ctx := context.Background()
		_, _ = s.filaService.ProximoClienteAtendimento(ctx)
	}()

	return nil
}

// AbandonarAtendimento abandona um atendimento
func (s *AtendimentoService) AbandonarAtendimento(ctx context.Context, atendimentoID int) error {
	// Buscar atendimento
	atendimento, err := s.repo.BuscarAtendimentoPorID(atendimentoID)
	if err != nil {
		return fmt.Errorf("erro ao buscar atendimento: %w", err)
	}
	if atendimento == nil {
		return fmt.Errorf("atendimento não encontrado")
	}

	if atendimento.Status != models.StatusAguardando && atendimento.Status != models.StatusEmAtendimento {
		return fmt.Errorf("atendimento não pode ser abandonado")
	}

	// Atualizar status
	if err := s.repo.AtualizarStatusAtendimento(atendimentoID, models.StatusAbandonado, atendimento.Atendente.ID); err != nil {
		return fmt.Errorf("erro ao abandonar atendimento: %w", err)
	}

	// Remover da fila
	if err := s.filaService.RemoverClienteFila(ctx, atendimento.ClienteID, "abandono"); err != nil {
		_ = err
	}

	return nil
}

// ============================================
// MENSAGENS
// ============================================

// EnviarMensagem envia uma mensagem no atendimento
func (s *AtendimentoService) EnviarMensagem(ctx context.Context, atendimentoID int, remetente, conteudo, tipo string, arquivoURL string) (*models.Mensagem, error) {
	// Verificar se atendimento existe
	atendimento, err := s.repo.BuscarAtendimentoPorID(atendimentoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimento: %w", err)
	}
	if atendimento == nil {
		return nil, fmt.Errorf("atendimento não encontrado")
	}

	if atendimento.Status == models.StatusFinalizado || atendimento.Status == models.StatusAbandonado {
		return nil, fmt.Errorf("atendimento já foi finalizado")
	}

	mensagem := &models.Mensagem{
		AtendimentoID: atendimentoID,
		Remetente:     remetente,
		Conteudo:      conteudo,
		Tipo:          tipo,
		ArquivoURL:    arquivoURL,
		Lida:          false,
	}

	if err := s.repo.EnviarMensagem(mensagem); err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	return mensagem, nil
}

// ListarMensagens lista mensagens de um atendimento
func (s *AtendimentoService) ListarMensagens(ctx context.Context, atendimentoID int) ([]*models.Mensagem, error) {
	return s.repo.ListarMensagens(atendimentoID)
}

// ============================================
// CONSULTAS
// ============================================

// BuscarAtendimento busca um atendimento por ID
func (s *AtendimentoService) BuscarAtendimento(ctx context.Context, atendimentoID int) (*models.Atendimento, error) {
	return s.repo.BuscarAtendimentoPorID(atendimentoID)
}

// ListarAtendimentos lista atendimentos com paginação
func (s *AtendimentoService) ListarAtendimentos(ctx context.Context, lojaID int, limit, offset int) ([]*models.AtendimentoResumo, error) {
	return s.repo.ListarAtendimentos(lojaID, limit, offset)
}

// GetAtendimentoStats retorna estatísticas de atendimentos
func (s *AtendimentoService) GetAtendimentoStats(ctx context.Context, lojaID int) (*models.AtendimentoStats, error) {
	return s.repo.GetAtendimentoStats(lojaID)
}

// GetAtendimentosAtivos retorna atendimentos ativos
func (s *AtendimentoService) GetAtendimentosAtivos(ctx context.Context, lojaID int) ([]*models.Atendimento, error) {
	return s.repo.BuscarAtendimentosAtivos(lojaID)
}

// ============================================
// ORÇAMENTOS NO ATENDIMENTO
// ============================================

// CriarOrcamentoNoAtendimento cria um orçamento a partir de um atendimento
func (s *AtendimentoService) CriarOrcamentoNoAtendimento(ctx context.Context, atendimentoID int, total, desconto float64, observacao string) (*models.Orcamento, error) {
	// Buscar atendimento
	atendimento, err := s.repo.BuscarAtendimentoPorID(atendimentoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimento: %w", err)
	}
	if atendimento == nil {
		return nil, fmt.Errorf("atendimento não encontrado")
	}

	if atendimento.Status == models.StatusFinalizado || atendimento.Status == models.StatusAbandonado {
		return nil, fmt.Errorf("atendimento já foi finalizado")
	}

	orcamento := &models.Orcamento{
		ClienteID:     atendimento.ClienteID,
		LojaID:        atendimento.LojaID,
		AtendimentoID: atendimentoID,
		Total:         total,
		Desconto:      desconto,
		TotalComDesconto: total - desconto,
		Observacao:    observacao,
		Status:        "pendente",
	}

	if err := s.orcamentoRepo.CriarOrcamento(orcamento); err != nil {
		return nil, fmt.Errorf("erro ao criar orçamento: %w", err)
	}

	return orcamento, nil
}