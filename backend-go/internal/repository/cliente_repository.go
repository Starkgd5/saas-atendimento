package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type ClienteRepository struct {
	db *sql.DB
}

func NewClienteRepository(db *sql.DB) *ClienteRepository {
	return &ClienteRepository{db: db}
}

// ============================================
// CRUD BÁSICO
// ============================================

// BuscarClientePorID busca um cliente pelo ID
func (r *ClienteRepository) BuscarClientePorID(id int) (*models.Cliente, error) {
	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE id = ?
	`

	var cliente models.Cliente
	var ultimoAtendimento sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&cliente.ID,
		&cliente.LojaID,
		&cliente.Nome,
		&cliente.Telefone,
		&cliente.Email,
		&ultimoAtendimento,
		&cliente.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente por ID: %w", err)
	}

	if ultimoAtendimento.Valid {
		cliente.UltimoAtendimento = &ultimoAtendimento.Time
	}

	return &cliente, nil
}

// BuscarClientePorTelefone busca um cliente pelo telefone
func (r *ClienteRepository) BuscarClientePorTelefone(telefone string) (*models.Cliente, error) {
	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE telefone = ?
	`

	var cliente models.Cliente
	var ultimoAtendimento sql.NullTime

	err := r.db.QueryRow(query, telefone).Scan(
		&cliente.ID,
		&cliente.LojaID,
		&cliente.Nome,
		&cliente.Telefone,
		&cliente.Email,
		&ultimoAtendimento,
		&cliente.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente por telefone: %w", err)
	}

	if ultimoAtendimento.Valid {
		cliente.UltimoAtendimento = &ultimoAtendimento.Time
	}

	return &cliente, nil
}

// BuscarClientePorEmail busca um cliente pelo email
func (r *ClienteRepository) BuscarClientePorEmail(email string) (*models.Cliente, error) {
	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE email = ?
	`

	var cliente models.Cliente
	var ultimoAtendimento sql.NullTime

	err := r.db.QueryRow(query, email).Scan(
		&cliente.ID,
		&cliente.LojaID,
		&cliente.Nome,
		&cliente.Telefone,
		&cliente.Email,
		&ultimoAtendimento,
		&cliente.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente por email: %w", err)
	}

	if ultimoAtendimento.Valid {
		cliente.UltimoAtendimento = &ultimoAtendimento.Time
	}

	return &cliente, nil
}

// ListarClientes lista clientes com paginação
func (r *ClienteRepository) ListarClientes(lojaID int, limit, offset int) ([]*models.Cliente, int, error) {
	if limit == 0 {
		limit = 50
	}

	// Buscar total
	var total int
	countQuery := `SELECT COUNT(*) FROM clientes WHERE loja_id = ?`
	err := r.db.QueryRow(countQuery, lojaID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar clientes: %w", err)
	}

	// Buscar clientes
	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE loja_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, lojaID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar clientes: %w", err)
	}
	defer rows.Close()

	var clientes []*models.Cliente
	for rows.Next() {
		var c models.Cliente
		var ultimoAtendimento sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.LojaID,
			&c.Nome,
			&c.Telefone,
			&c.Email,
			&ultimoAtendimento,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear cliente: %w", err)
		}

		if ultimoAtendimento.Valid {
			c.UltimoAtendimento = &ultimoAtendimento.Time
		}

		clientes = append(clientes, &c)
	}

	return clientes, total, nil
}

// BuscarClientesPorNome busca clientes pelo nome (like)
func (r *ClienteRepository) BuscarClientesPorNome(lojaID int, nome string, limit int) ([]*models.Cliente, error) {
	if limit == 0 {
		limit = 20
	}

	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE loja_id = ? AND nome LIKE ?
		ORDER BY nome ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, lojaID, "%"+nome+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes por nome: %w", err)
	}
	defer rows.Close()

	var clientes []*models.Cliente
	for rows.Next() {
		var c models.Cliente
		var ultimoAtendimento sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.LojaID,
			&c.Nome,
			&c.Telefone,
			&c.Email,
			&ultimoAtendimento,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear cliente: %w", err)
		}

		if ultimoAtendimento.Valid {
			c.UltimoAtendimento = &ultimoAtendimento.Time
		}

		clientes = append(clientes, &c)
	}

	return clientes, nil
}

// BuscarClientesRecentes busca clientes recentes
func (r *ClienteRepository) BuscarClientesRecentes(lojaID int, limit int) ([]*models.Cliente, error) {
	if limit == 0 {
		limit = 10
	}

	query := `
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE loja_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, lojaID, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar clientes recentes: %w", err)
	}
	defer rows.Close()

	var clientes []*models.Cliente
	for rows.Next() {
		var c models.Cliente
		var ultimoAtendimento sql.NullTime

		err := rows.Scan(
			&c.ID,
			&c.LojaID,
			&c.Nome,
			&c.Telefone,
			&c.Email,
			&ultimoAtendimento,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear cliente: %w", err)
		}

		if ultimoAtendimento.Valid {
			c.UltimoAtendimento = &ultimoAtendimento.Time
		}

		clientes = append(clientes, &c)
	}

	return clientes, nil
}

// ============================================
// CRIAÇÃO E ATUALIZAÇÃO
// ============================================

// CriarCliente cria um novo cliente
func (r *ClienteRepository) CriarCliente(cliente *models.Cliente) error {
	query := `
		INSERT INTO clientes (loja_id, nome, telefone, email)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.Exec(query, cliente.LojaID, cliente.Nome, cliente.Telefone, cliente.Email)
	if err != nil {
		return fmt.Errorf("erro ao criar cliente: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do cliente: %w", err)
	}

	cliente.ID = int(id)
	cliente.CreatedAt = time.Now()
	return nil
}

// AtualizarCliente atualiza um cliente existente
func (r *ClienteRepository) AtualizarCliente(cliente *models.Cliente) error {
	query := `
		UPDATE clientes 
		SET nome = ?, telefone = ?, email = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	result, err := r.db.Exec(query, cliente.Nome, cliente.Telefone, cliente.Email, cliente.ID, cliente.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar cliente: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("cliente não encontrado")
	}

	return nil
}

// AtualizarUltimoAtendimento atualiza a data do último atendimento
func (r *ClienteRepository) AtualizarUltimoAtendimento(clienteID int) error {
	query := `UPDATE clientes SET ultimo_atendimento = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, clienteID)
	return err
}

// AtualizarTelefone atualiza o telefone de um cliente
func (r *ClienteRepository) AtualizarTelefone(clienteID int, telefone string) error {
	query := `UPDATE clientes SET telefone = ? WHERE id = ?`
	_, err := r.db.Exec(query, telefone, clienteID)
	return err
}

// AtualizarEmail atualiza o email de um cliente
func (r *ClienteRepository) AtualizarEmail(clienteID int, email string) error {
	query := `UPDATE clientes SET email = ? WHERE id = ?`
	_, err := r.db.Exec(query, email, clienteID)
	return err
}

// ============================================
// EXCLUSÃO
// ============================================

// DeletarCliente deleta um cliente
func (r *ClienteRepository) DeletarCliente(id int, lojaID int) error {
	query := `DELETE FROM clientes WHERE id = ? AND loja_id = ?`
	result, err := r.db.Exec(query, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao deletar cliente: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("cliente não encontrado")
	}

	return nil
}

// ============================================
// MÉTRICAS
// ============================================

// ContarClientes conta o total de clientes de uma loja
func (r *ClienteRepository) ContarClientes(lojaID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM clientes WHERE loja_id = ?`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}

// ContarClientesAtivos conta clientes que tiveram atendimento nos últimos 30 dias
func (r *ClienteRepository) ContarClientesAtivos(lojaID int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM clientes 
		WHERE loja_id = ? AND (ultimo_atendimento >= DATE_SUB(NOW(), INTERVAL 30 DAY) OR ultimo_atendimento IS NOT NULL)
	`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}

// ContarNovosClientesMes conta novos clientes do mês
func (r *ClienteRepository) ContarNovosClientesMes(lojaID int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM clientes 
		WHERE loja_id = ? AND MONTH(created_at) = MONTH(CURDATE()) 
		AND YEAR(created_at) = YEAR(CURDATE())
	`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}

// ============================================
// VALIDAÇÕES
// ============================================

// TelefoneExiste verifica se um telefone já está cadastrado
func (r *ClienteRepository) TelefoneExiste(telefone string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM clientes WHERE telefone = ?)`
	err := r.db.QueryRow(query, telefone).Scan(&exists)
	return exists, err
}

// TelefoneExisteExcluindoID verifica se um telefone já está cadastrado (excluindo um ID)
func (r *ClienteRepository) TelefoneExisteExcluindoID(telefone string, clienteID int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM clientes WHERE telefone = ? AND id != ?)`
	err := r.db.QueryRow(query, telefone, clienteID).Scan(&exists)
	return exists, err
}

// EmailExiste verifica se um email já está cadastrado
func (r *ClienteRepository) EmailExiste(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM clientes WHERE email = ?)`
	err := r.db.QueryRow(query, email).Scan(&exists)
	return exists, err
}