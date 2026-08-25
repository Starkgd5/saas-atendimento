package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type FornecedorService struct {
	repo *repository.FornecedorRepository
}

func NewFornecedorService(repo *repository.FornecedorRepository) *FornecedorService {
	return &FornecedorService{repo: repo}
}

func (s *FornecedorService) CriarFornecedor(ctx context.Context, fornecedor *models.Fornecedor) error {
	if fornecedor.RazaoSocial == "" {
		return fmt.Errorf("razão social é obrigatória")
	}
	if fornecedor.CNPJ == "" {
		return fmt.Errorf("CNPJ é obrigatório")
	}
	fornecedor.Ativo = true
	return s.repo.CriarFornecedor(fornecedor)
}

func (s *FornecedorService) BuscarFornecedor(ctx context.Context, id, lojaID int) (*models.Fornecedor, error) {
	return s.repo.BuscarFornecedorPorID(id, lojaID)
}

func (s *FornecedorService) ListarFornecedores(ctx context.Context, lojaID int) ([]*models.Fornecedor, error) {
	return s.repo.ListarFornecedores(lojaID)
}

func (s *FornecedorService) AtualizarFornecedor(ctx context.Context, fornecedor *models.Fornecedor) error {
	if fornecedor.ID == 0 {
		return fmt.Errorf("ID do fornecedor é obrigatório")
	}
	return s.repo.AtualizarFornecedor(fornecedor)
}

func (s *FornecedorService) DeletarFornecedor(ctx context.Context, id, lojaID int) error {
	return s.repo.DeletarFornecedor(id, lojaID)
}