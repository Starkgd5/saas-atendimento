package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type ClienteService struct {
	repo *repository.ClienteRepository
}

func NewClienteService(repo *repository.ClienteRepository) *ClienteService {
	return &ClienteService{repo: repo}
}

// CriarCliente cria um novo cliente
func (s *ClienteService) CriarCliente(ctx context.Context, cliente *models.Cliente) error {
	// Validar dados
	if cliente.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}
	if cliente.Telefone == "" {
		return fmt.Errorf("telefone é obrigatório")
	}
	if cliente.LojaID == 0 {
		return fmt.Errorf("loja é obrigatória")
	}

	// Verificar se telefone já existe
	exists, err := s.repo.TelefoneExiste(cliente.Telefone)
	if err != nil {
		return fmt.Errorf("erro ao verificar telefone: %w", err)
	}
	if exists {
		return fmt.Errorf("telefone %s já está cadastrado", cliente.Telefone)
	}

	// Verificar email se informado
	if cliente.Email != "" {
		exists, err := s.repo.EmailExiste(cliente.Email)
		if err != nil {
			return fmt.Errorf("erro ao verificar email: %w", err)
		}
		if exists {
			return fmt.Errorf("email %s já está cadastrado", cliente.Email)
		}
	}

	return s.repo.CriarCliente(cliente)
}

// BuscarClientePorID busca um cliente pelo ID
func (s *ClienteService) BuscarClientePorID(ctx context.Context, id int) (*models.Cliente, error) {
	if id <= 0 {
		return nil, fmt.Errorf("ID inválido")
	}
	return s.repo.BuscarClientePorID(id)
}

// BuscarClientePorTelefone busca um cliente pelo telefone
func (s *ClienteService) BuscarClientePorTelefone(ctx context.Context, telefone string) (*models.Cliente, error) {
	if telefone == "" {
		return nil, fmt.Errorf("telefone é obrigatório")
	}
	return s.repo.BuscarClientePorTelefone(telefone)
}

// ListarClientes lista clientes com paginação
func (s *ClienteService) ListarClientes(ctx context.Context, lojaID int, limit, offset int) ([]*models.Cliente, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListarClientes(lojaID, limit, offset)
}

// BuscarClientesPorNome busca clientes por nome
func (s *ClienteService) BuscarClientesPorNome(ctx context.Context, lojaID int, nome string) ([]*models.Cliente, error) {
	if nome == "" {
		return nil, fmt.Errorf("nome é obrigatório")
	}
	return s.repo.BuscarClientesPorNome(lojaID, nome, 20)
}

// AtualizarCliente atualiza um cliente
func (s *ClienteService) AtualizarCliente(ctx context.Context, cliente *models.Cliente) error {
	if cliente.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}
	if cliente.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}

	// Verificar se cliente existe
	existing, err := s.repo.BuscarClientePorID(cliente.ID)
	if err != nil {
		return fmt.Errorf("erro ao buscar cliente: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("cliente não encontrado")
	}

	// Verificar telefone duplicado
	if cliente.Telefone != existing.Telefone {
		exists, err := s.repo.TelefoneExisteExcluindoID(cliente.Telefone, cliente.ID)
		if err != nil {
			return fmt.Errorf("erro ao verificar telefone: %w", err)
		}
		if exists {
			return fmt.Errorf("telefone %s já está cadastrado", cliente.Telefone)
		}
	}

	return s.repo.AtualizarCliente(cliente)
}

// DeletarCliente deleta um cliente
func (s *ClienteService) DeletarCliente(ctx context.Context, id, lojaID int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}
	return s.repo.DeletarCliente(id, lojaID)
}

// AtualizarUltimoAtendimento atualiza a data do último atendimento
func (s *ClienteService) AtualizarUltimoAtendimento(ctx context.Context, clienteID int) error {
	if clienteID <= 0 {
		return fmt.Errorf("ID inválido")
	}
	return s.repo.AtualizarUltimoAtendimento(clienteID)
}