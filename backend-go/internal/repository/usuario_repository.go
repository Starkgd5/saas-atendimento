package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type UsuarioRepository struct {
	db *sql.DB
}

func NewUsuarioRepository(db *sql.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

// ============================================
// CRUD BÁSICO
// ============================================

// BuscarUsuarioPorID busca um usuário pelo ID
func (r *UsuarioRepository) BuscarUsuarioPorID(id int) (*models.Usuario, error) {
	query := `
		SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
		FROM usuarios
		WHERE id = ?
	`

	var usuario models.Usuario
	var lojaID sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&usuario.ID,
		&lojaID,
		&usuario.Nome,
		&usuario.Email,
		&usuario.SenhaHash,
		&usuario.Role,
		&usuario.Ativo,
		&usuario.CreatedAt,
		&usuario.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário: %w", err)
	}

	if lojaID.Valid {
		id := int(lojaID.Int64)
		usuario.LojaID = &id
	}

	return &usuario, nil
}

// BuscarUsuarioPorEmail busca um usuário pelo email
func (r *UsuarioRepository) BuscarUsuarioPorEmail(email string) (*models.Usuario, error) {
	query := `
		SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
		FROM usuarios
		WHERE email = ?
	`

	var usuario models.Usuario
	var lojaID sql.NullInt64

	err := r.db.QueryRow(query, email).Scan(
		&usuario.ID,
		&lojaID,
		&usuario.Nome,
		&usuario.Email,
		&usuario.SenhaHash,
		&usuario.Role,
		&usuario.Ativo,
		&usuario.CreatedAt,
		&usuario.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar usuário por email: %w", err)
	}

	if lojaID.Valid {
		id := int(lojaID.Int64)
		usuario.LojaID = &id
	}

	return &usuario, nil
}

// ListarUsuarios lista todos os usuários
func (r *UsuarioRepository) ListarUsuarios(lojaID *int, limit, offset int) ([]*models.Usuario, int, error) {
	if limit == 0 {
		limit = 50
	}

	var args []interface{}
	var countQuery string
	var query string

	if lojaID != nil {
		countQuery = `SELECT COUNT(*) FROM usuarios WHERE loja_id = ?`
		query = `
			SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
			FROM usuarios
			WHERE loja_id = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, *lojaID)
	} else {
		countQuery = `SELECT COUNT(*) FROM usuarios`
		query = `
			SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
			FROM usuarios
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
	}

	// Contar total
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar usuários: %w", err)
	}

	// Buscar usuários
	args = append(args, limit, offset)
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar usuários: %w", err)
	}
	defer rows.Close()

	var usuarios []*models.Usuario
	for rows.Next() {
		var u models.Usuario
		var lojaIDNull sql.NullInt64

		err := rows.Scan(
			&u.ID,
			&lojaIDNull,
			&u.Nome,
			&u.Email,
			&u.SenhaHash,
			&u.Role,
			&u.Ativo,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear usuário: %w", err)
		}

		if lojaIDNull.Valid {
			id := int(lojaIDNull.Int64)
			u.LojaID = &id
		}

		usuarios = append(usuarios, &u)
	}

	return usuarios, total, nil
}

// ListarUsuariosPorRole lista usuários por role
func (r *UsuarioRepository) ListarUsuariosPorRole(role string) ([]*models.Usuario, error) {
	query := `
		SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
		FROM usuarios
		WHERE role = ?
		ORDER BY nome ASC
	`

	rows, err := r.db.Query(query, role)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar usuários por role: %w", err)
	}
	defer rows.Close()

	var usuarios []*models.Usuario
	for rows.Next() {
		var u models.Usuario
		var lojaID sql.NullInt64

		err := rows.Scan(
			&u.ID,
			&lojaID,
			&u.Nome,
			&u.Email,
			&u.SenhaHash,
			&u.Role,
			&u.Ativo,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear usuário: %w", err)
		}

		if lojaID.Valid {
			id := int(lojaID.Int64)
			u.LojaID = &id
		}

		usuarios = append(usuarios, &u)
	}

	return usuarios, nil
}

// ListarUsuariosAtivos lista usuários ativos
func (r *UsuarioRepository) ListarUsuariosAtivos() ([]*models.Usuario, error) {
	query := `
		SELECT id, loja_id, nome, email, senha_hash, role, ativo, created_at, updated_at
		FROM usuarios
		WHERE ativo = 1
		ORDER BY nome ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar usuários ativos: %w", err)
	}
	defer rows.Close()

	var usuarios []*models.Usuario
	for rows.Next() {
		var u models.Usuario
		var lojaID sql.NullInt64

		err := rows.Scan(
			&u.ID,
			&lojaID,
			&u.Nome,
			&u.Email,
			&u.SenhaHash,
			&u.Role,
			&u.Ativo,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear usuário: %w", err)
		}

		if lojaID.Valid {
			id := int(lojaID.Int64)
			u.LojaID = &id
		}

		usuarios = append(usuarios, &u)
	}

	return usuarios, nil
}

// ============================================
// CRIAÇÃO E ATUALIZAÇÃO
// ============================================

// CriarUsuario cria um novo usuário
func (r *UsuarioRepository) CriarUsuario(usuario *models.Usuario) error {
	query := `
		INSERT INTO usuarios (loja_id, nome, email, senha_hash, role, ativo)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var lojaID interface{}
	if usuario.LojaID != nil {
		lojaID = *usuario.LojaID
	} else {
		lojaID = nil
	}

	result, err := r.db.Exec(query, lojaID, usuario.Nome, usuario.Email, usuario.SenhaHash, usuario.Role, usuario.Ativo)
	if err != nil {
		return fmt.Errorf("erro ao criar usuário: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do usuário: %w", err)
	}

	usuario.ID = int(id)
	usuario.CreatedAt = time.Now()
	usuario.UpdatedAt = time.Now()
	return nil
}

// AtualizarUsuario atualiza um usuário existente
func (r *UsuarioRepository) AtualizarUsuario(usuario *models.Usuario) error {
	query := `
		UPDATE usuarios 
		SET nome = ?, email = ?, role = ?, ativo = ?, loja_id = ?, updated_at = NOW()
	`
	args := []interface{}{usuario.Nome, usuario.Email, usuario.Role, usuario.Ativo}

	if usuario.LojaID != nil {
		query += `, loja_id = ?`
		args = append(args, *usuario.LojaID)
	} else {
		query += `, loja_id = NULL`
	}

	if usuario.SenhaHash != "" {
		query += `, senha_hash = ?`
		args = append(args, usuario.SenhaHash)
	}

	query += ` WHERE id = ?`
	args = append(args, usuario.ID)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("erro ao atualizar usuário: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("usuário não encontrado")
	}

	return nil
}

// AtualizarSenha atualiza a senha de um usuário
func (r *UsuarioRepository) AtualizarSenha(id int, senhaHash string) error {
	_, err := r.db.Exec(`
		UPDATE usuarios SET senha_hash = ?, updated_at = NOW()
		WHERE id = ?
	`, senhaHash, id)
	return err
}

// AtivarUsuario ativa um usuário
func (r *UsuarioRepository) AtivarUsuario(id int) error {
	_, err := r.db.Exec(`
		UPDATE usuarios SET ativo = 1, updated_at = NOW()
		WHERE id = ?
	`, id)
	return err
}

// DesativarUsuario desativa um usuário
func (r *UsuarioRepository) DesativarUsuario(id int) error {
	_, err := r.db.Exec(`
		UPDATE usuarios SET ativo = 0, updated_at = NOW()
		WHERE id = ?
	`, id)
	return err
}

// ============================================
// EXCLUSÃO
// ============================================

// DeletarUsuario deleta um usuário
func (r *UsuarioRepository) DeletarUsuario(id int) error {
	result, err := r.db.Exec(`DELETE FROM usuarios WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("erro ao deletar usuário: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("usuário não encontrado")
	}

	return nil
}

// ============================================
// VALIDAÇÕES
// ============================================

// EmailExiste verifica se um email já está cadastrado
func (r *UsuarioRepository) EmailExiste(email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM usuarios WHERE email = ?)`, email).Scan(&exists)
	return exists, err
}

// EmailExisteExcluindoID verifica se um email já está cadastrado (excluindo um ID)
func (r *UsuarioRepository) EmailExisteExcluindoID(email string, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM usuarios WHERE email = ? AND id != ?)`, email, id).Scan(&exists)
	return exists, err
}