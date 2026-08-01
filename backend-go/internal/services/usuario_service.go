package services

import (
	"context"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type UsuarioService struct {
	repo       *repository.UsuarioRepository
	jwtService *JWTService
}

func NewUsuarioService(repo *repository.UsuarioRepository, jwtService *JWTService) *UsuarioService {
	return &UsuarioService{
		repo:       repo,
		jwtService: jwtService,
	}
}

// CriarUsuario cria um novo usuário
func (s *UsuarioService) CriarUsuario(ctx context.Context, usuario *models.Usuario) error {
	if usuario.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}
	if usuario.Email == "" {
		return fmt.Errorf("email é obrigatório")
	}
	if usuario.SenhaHash == "" {
		return fmt.Errorf("senha é obrigatória")
	}

	// Verificar se email já existe
	exists, err := s.repo.EmailExiste(usuario.Email)
	if err != nil {
		return fmt.Errorf("erro ao verificar email: %w", err)
	}
	if exists {
		return fmt.Errorf("email %s já está cadastrado", usuario.Email)
	}

	if usuario.Role == "" {
		usuario.Role = models.RoleAtendente
	}
	usuario.Ativo = true

	return s.repo.CriarUsuario(usuario)
}

// BuscarUsuarioPorID busca um usuário pelo ID
func (s *UsuarioService) BuscarUsuarioPorID(ctx context.Context, id int) (*models.Usuario, error) {
	if id <= 0 {
		return nil, fmt.Errorf("ID inválido")
	}
	return s.repo.BuscarUsuarioPorID(id)
}

// BuscarUsuarioPorEmail busca um usuário pelo email
func (s *UsuarioService) BuscarUsuarioPorEmail(ctx context.Context, email string) (*models.Usuario, error) {
	if email == "" {
		return nil, fmt.Errorf("email é obrigatório")
	}
	return s.repo.BuscarUsuarioPorEmail(email)
}

// ListarUsuarios lista usuários com paginação
func (s *UsuarioService) ListarUsuarios(ctx context.Context, lojaID *int, limit, offset int) ([]*models.Usuario, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListarUsuarios(lojaID, limit, offset)
}

// AtualizarUsuario atualiza um usuário
func (s *UsuarioService) AtualizarUsuario(ctx context.Context, usuario *models.Usuario) error {
	if usuario.ID <= 0 {
		return fmt.Errorf("ID inválido")
	}
	if usuario.Nome == "" {
		return fmt.Errorf("nome é obrigatório")
	}

	existing, err := s.repo.BuscarUsuarioPorID(usuario.ID)
	if err != nil {
		return fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("usuário não encontrado")
	}

	// Verificar email duplicado
	if usuario.Email != existing.Email {
		exists, err := s.repo.EmailExisteExcluindoID(usuario.Email, usuario.ID)
		if err != nil {
			return fmt.Errorf("erro ao verificar email: %w", err)
		}
		if exists {
			return fmt.Errorf("email %s já está cadastrado", usuario.Email)
		}
	}

	return s.repo.AtualizarUsuario(usuario)
}

// DeletarUsuario deleta um usuário
func (s *UsuarioService) DeletarUsuario(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}
	return s.repo.DeletarUsuario(id)
}

// AtivarUsuario ativa um usuário
func (s *UsuarioService) AtivarUsuario(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}
	return s.repo.AtivarUsuario(id)
}

// DesativarUsuario desativa um usuário
func (s *UsuarioService) DesativarUsuario(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("ID inválido")
	}
	return s.repo.DesativarUsuario(id)
}

// Login realiza login de um usuário
func (s *UsuarioService) Login(ctx context.Context, email, senha string) (string, *models.Usuario, error) {
	usuario, err := s.repo.BuscarUsuarioPorEmail(email)
	if err != nil {
		return "", nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if usuario == nil {
		return "", nil, fmt.Errorf("usuário não encontrado")
	}
	if !usuario.Ativo {
		return "", nil, fmt.Errorf("usuário inativo")
	}

	// Verificar senha (simplificado - usar bcrypt em produção)
	if usuario.SenhaHash != senha {
		return "", nil, fmt.Errorf("senha incorreta")
	}

	// Gerar token
	lojaID := 0
	if usuario.LojaID != nil {
		lojaID = *usuario.LojaID
	}

	token, err := s.jwtService.GenerateTokenWithDetails(
		usuario.ID,
		lojaID,
		usuario.Role,
		usuario.Email,
		usuario.Nome,
	)
	if err != nil {
		return "", nil, fmt.Errorf("erro ao gerar token: %w", err)
	}

	return token, usuario, nil
}

// ValidarToken valida um token JWT
func (s *UsuarioService) ValidarToken(ctx context.Context, token string) (*models.Usuario, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("token inválido: %w", err)
	}

	usuario, err := s.repo.BuscarUsuarioPorID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}
	if usuario == nil {
		return nil, fmt.Errorf("usuário não encontrado")
	}
	if !usuario.Ativo {
		return nil, fmt.Errorf("usuário inativo")
	}

	return usuario, nil
}