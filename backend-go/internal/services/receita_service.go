package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type ReceitaService struct {
	repo        *repository.ReceitaRepository
	clienteRepo *repository.ClienteRepository
}

func NewReceitaService(repo *repository.ReceitaRepository, clienteRepo *repository.ClienteRepository) *ReceitaService {
	return &ReceitaService{
		repo:        repo,
		clienteRepo: clienteRepo,
	}
}

func (s *ReceitaService) CriarReceita(ctx context.Context, receita *models.ReceitaMedica) error {
	if receita.ClienteID == 0 {
		return fmt.Errorf("cliente é obrigatório")
	}
	if receita.NumeroReceita == "" {
		return fmt.Errorf("número da receita é obrigatório")
	}
	if receita.MedicoNome == "" {
		return fmt.Errorf("nome do médico é obrigatório")
	}
	if receita.MedicoCRM == "" {
		return fmt.Errorf("CRM do médico é obrigatório")
	}

	cliente, err := s.clienteRepo.BuscarClientePorID(receita.ClienteID)
	if err != nil {
		return err
	}
	if cliente == nil {
		return fmt.Errorf("cliente não encontrado")
	}

	receita.Status = "Pendente"
	return s.repo.CriarReceita(receita)
}

func (s *ReceitaService) ValidarReceita(ctx context.Context, receitaID int, farmaceuticoID int, observacao string) error {
	receita, err := s.repo.BuscarReceitaPorID(receitaID)
	if err != nil {
		return err
	}
	if receita == nil {
		return fmt.Errorf("receita não encontrada")
	}

	if receita.Status != "Pendente" {
		return fmt.Errorf("receita já foi processada")
	}

	agora := time.Now()
	receita.Status = "Validada"
	receita.DataValidacao = &agora
	receita.FarmaceuticoID = &farmaceuticoID
	receita.Observacao = observacao

	return s.repo.AtualizarStatus(receitaID, "Validada", observacao)
}

func (s *ReceitaService) RejeitarReceita(ctx context.Context, receitaID int, observacao string) error {
	receita, err := s.repo.BuscarReceitaPorID(receitaID)
	if err != nil {
		return err
	}
	if receita == nil {
		return fmt.Errorf("receita não encontrada")
	}

	if receita.Status != "Pendente" {
		return fmt.Errorf("receita já foi processada")
	}

	return s.repo.AtualizarStatus(receitaID, "Rejeitada", observacao)
}

func (s *ReceitaService) ListarReceitas(ctx context.Context, lojaID int, status string, limit, offset int) ([]*models.ReceitaMedica, error) {
	return s.repo.ListarReceitas(lojaID, status, limit, offset)
}

func (s *ReceitaService) BuscarReceita(ctx context.Context, receitaID int) (*models.ReceitaMedica, error) {
	return s.repo.BuscarReceitaPorID(receitaID)
}