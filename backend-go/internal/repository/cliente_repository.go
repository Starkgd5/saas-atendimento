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
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	if ultimoAtendimento.Valid {
		cliente.UltimoAtendimento = &ultimoAtendimento.Time
	}

	return &cliente, nil
}

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

// AtualizarUltimoAtendimento atualiza a data do último atendimento
func (r *ClienteRepository) AtualizarUltimoAtendimento(clienteID int) error {
	query := `UPDATE clientes SET ultimo_atendimento = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, clienteID)
	return err
}