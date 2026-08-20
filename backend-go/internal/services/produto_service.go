package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type ProdutoService struct {
	repo *repository.ProdutoRepository
}

func NewProdutoService(repo *repository.ProdutoRepository) *ProdutoService {
	return &ProdutoService{repo: repo}
}

// CriarProduto cria um novo produto
func (s *ProdutoService) CriarProduto(ctx context.Context, produto *models.Produto) error {
	if produto.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}
	if produto.Categoria == "" {
		return fmt.Errorf("categoria é obrigatória")
	}
	if produto.PrecoVenda < 0 {
		return fmt.Errorf("preço não pode ser negativo")
	}
	if produto.LojaID == 0 {
		return fmt.Errorf("loja é obrigatória")
	}

	// Verificar se nome já existe
	exists, err := s.repo.NomeExiste(produto.Nome, produto.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao verificar nome: %w", err)
	}
	if exists {
		return fmt.Errorf("produto %s já existe", produto.Nome)
	}

	produto.Ativo = true
	return s.repo.CriarProduto(produto)
}

// BuscarProdutoPorID busca um produto pelo ID
func (s *ProdutoService) BuscarProdutoPorID(ctx context.Context, id, lojaID int) (*models.Produto, error) {
	if id <= 0 {
		return nil, fmt.Errorf("ID inválido")
	}
	return s.repo.BuscarProdutoPorID(id, lojaID)
}

// ListarProdutos lista produtos com paginação
func (s *ProdutoService) ListarProdutos(ctx context.Context, lojaID int, categoria string, ativo *bool, limit, offset int) ([]*models.Produto, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListarProdutos(lojaID, categoria, ativo, limit, offset)
}

// ListarProdutosEmFalta lista produtos com estoque baixo
func (s *ProdutoService) ListarProdutosEmFalta(ctx context.Context, lojaID int) ([]*models.Produto, error) {
	return s.repo.ListarProdutosEmFalta(lojaID)
}

// ListarCategorias lista categorias disponíveis
func (s *ProdutoService) ListarCategorias(ctx context.Context, lojaID int) ([]string, error) {
	return s.repo.ListarCategorias(lojaID)
}

// AtualizarProduto atualiza um produto
func (s *ProdutoService) AtualizarProduto(ctx context.Context, produto *models.Produto) error {
	if produto.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}
	if produto.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}
	if produto.PrecoVenda < 0 {
		return fmt.Errorf("preço não pode ser negativo")
	}

	// Verificar se produto existe
	existing, err := s.repo.BuscarProdutoPorID(produto.ID, produto.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar produto: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("produto não encontrado")
	}

	// Verificar nome duplicado
	if produto.Nome != existing.Nome {
		exists, err := s.repo.NomeExisteExcluindoID(produto.Nome, produto.LojaID, produto.ID)
		if err != nil {
			return fmt.Errorf("erro ao verificar nome: %w", err)
		}
		if exists {
			return fmt.Errorf("produto %s já existe", produto.Nome)
		}
	}

	return s.repo.AtualizarProduto(produto)
}

// DeletarProduto deleta um produto
// func (s *ProdutoService) DeletarProduto(ctx context.Context, id, lojaID int) error {
// 	if id <= 0 {
// 		return fmt.Errorf("ID inválido")
// 	}
// 	return s.repo.DeletarProduto(id, lojaID)
// }
