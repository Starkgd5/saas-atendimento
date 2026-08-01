package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type OrcamentoRepository struct {
	db *sql.DB
}

func NewOrcamentoRepository(db *sql.DB) *OrcamentoRepository {
	return &OrcamentoRepository{db: db}
}

// ============================================
// CRUD BÁSICO
// ============================================

// BuscarOrcamentoPorID busca um orçamento pelo ID
func (r *OrcamentoRepository) BuscarOrcamentoPorID(id int, lojaID int) (*models.Orcamento, error) {
	query := `
		SELECT id, cliente_id, loja_id, atendimento_id, status, total,
		       desconto, total_com_desconto, observacao, expirado_em,
		       created_at, updated_at
		FROM orcamentos
		WHERE id = ? AND loja_id = ?
	`

	var o models.Orcamento
	var atendimentoID sql.NullInt64
	var expiradoEmTime sql.NullTime

	err := r.db.QueryRow(query, id, lojaID).Scan(
		&o.ID,
		&o.ClienteID,
		&o.LojaID,
		&atendimentoID,
		&o.Status,
		&o.Total,
		&o.Desconto,
		&o.TotalComDesconto,
		&o.Observacao,
		&expiradoEmTime,
		&o.CreatedAt,
		&o.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar orçamento: %w", err)
	}

	if atendimentoID.Valid {
		id := int(atendimentoID.Int64)
		o.AtendimentoID = id
	}
	if expiradoEmTime.Valid {
		o.ExpiradoEm = &expiradoEmTime.Time
	}

	return &o, nil
}

// ListarOrcamentos lista orçamentos com paginação
func (r *OrcamentoRepository) ListarOrcamentos(lojaID int, status string, clienteID *int, limit, offset int) ([]*models.Orcamento, int, error) {
	if limit == 0 {
		limit = 50
	}

	args := []interface{}{lojaID}
	whereClause := "WHERE loja_id = ?"

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if clienteID != nil {
		whereClause += " AND cliente_id = ?"
		args = append(args, *clienteID)
	}

	// Contar total
	countQuery := `SELECT COUNT(*) FROM orcamentos ` + whereClause
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar orçamentos: %w", err)
	}

	// Buscar orçamentos
	query := `
		SELECT id, cliente_id, loja_id, atendimento_id, status, total,
		       desconto, total_com_desconto, observacao, expirado_em,
		       created_at, updated_at
		FROM orcamentos ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar orçamentos: %w", err)
	}
	defer rows.Close()

	var orcamentos []*models.Orcamento
	for rows.Next() {
		var o models.Orcamento
		var atendimentoID sql.NullInt64
		var expiradoEmTime sql.NullTime

		err := rows.Scan(
			&o.ID,
			&o.ClienteID,
			&o.LojaID,
			&atendimentoID,
			&o.Status,
			&o.Total,
			&o.Desconto,
			&o.TotalComDesconto,
			&o.Observacao,
			&expiradoEmTime,
			&o.CreatedAt,
			&o.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear orçamento: %w", err)
		}

		if atendimentoID.Valid {
			id := int(atendimentoID.Int64)
			o.AtendimentoID = id
		}
		if expiradoEmTime.Valid {
			o.ExpiradoEm = &expiradoEmTime.Time
		}

		orcamentos = append(orcamentos, &o)
	}

	return orcamentos, total, nil
}

// BuscarOrcamentosPorCliente busca orçamentos de um cliente
func (r *OrcamentoRepository) BuscarOrcamentosPorCliente(clienteID int, limit int) ([]*models.Orcamento, error) {
	if limit == 0 {
		limit = 20
	}

	query := `
		SELECT id, cliente_id, loja_id, atendimento_id, status, total,
		       desconto, total_com_desconto, observacao, expirado_em,
		       created_at, updated_at
		FROM orcamentos
		WHERE cliente_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, clienteID, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar orçamentos do cliente: %w", err)
	}
	defer rows.Close()

	var orcamentos []*models.Orcamento
	for rows.Next() {
		var o models.Orcamento
		var atendimentoID sql.NullInt64
		var expiradoEmTime sql.NullTime

		err := rows.Scan(
			&o.ID,
			&o.ClienteID,
			&o.LojaID,
			&atendimentoID,
			&o.Status,
			&o.Total,
			&o.Desconto,
			&o.TotalComDesconto,
			&o.Observacao,
			&expiradoEmTime,
			&o.CreatedAt,
			&o.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear orçamento: %w", err)
		}

		if atendimentoID.Valid {
			id := int(atendimentoID.Int64)
			o.AtendimentoID = id
		}
		if expiradoEmTime.Valid {
			o.ExpiradoEm = &expiradoEmTime.Time
		}

		orcamentos = append(orcamentos, &o)
	}

	return orcamentos, nil
}

// ============================================
// CRIAÇÃO E ATUALIZAÇÃO
// ============================================

// CriarOrcamento cria um novo orçamento
func (r *OrcamentoRepository) CriarOrcamento(orcamento *models.Orcamento) error {
	query := `
		INSERT INTO orcamentos (cliente_id, loja_id, atendimento_id, status, total,
		                       desconto, total_com_desconto, observacao)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var atendimentoID interface{}
	// AtendimentoID is an int (not a pointer). Use nil for zero value (no atendimento).
	if orcamento.AtendimentoID != 0 {
		atendimentoID = orcamento.AtendimentoID
	} else {
		atendimentoID = nil
	}

	result, err := r.db.Exec(query,
		orcamento.ClienteID,
		orcamento.LojaID,
		atendimentoID,
		orcamento.Status,
		orcamento.Total,
		orcamento.Desconto,
		orcamento.TotalComDesconto,
		orcamento.Observacao,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar orçamento: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do orçamento: %w", err)
	}

	orcamento.ID = int(id)
	orcamento.CreatedAt = time.Now()
	orcamento.UpdatedAt = time.Now()
	return nil
}

// AtualizarOrcamento atualiza um orçamento existente
func (r *OrcamentoRepository) AtualizarOrcamento(orcamento *models.Orcamento) error {
	query := `
		UPDATE orcamentos 
		SET status = ?, total = ?, desconto = ?, total_com_desconto = ?,
		    observacao = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	result, err := r.db.Exec(query,
		orcamento.Status,
		orcamento.Total,
		orcamento.Desconto,
		orcamento.TotalComDesconto,
		orcamento.Observacao,
		orcamento.ID,
		orcamento.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar orçamento: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("orçamento não encontrado")
	}

	return nil
}

// AtualizarStatusOrcamento atualiza o status de um orçamento
func (r *OrcamentoRepository) AtualizarStatusOrcamento(id int, lojaID int, status string) error {
	query := `UPDATE orcamentos SET status = ?, updated_at = NOW() WHERE id = ? AND loja_id = ?`
	result, err := r.db.Exec(query, status, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status do orçamento: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("orçamento não encontrado")
	}

	return nil
}

// AprovarOrcamento aprova um orçamento
func (r *OrcamentoRepository) AprovarOrcamento(id int, lojaID int) error {
	return r.AtualizarStatusOrcamento(id, lojaID, models.OrcamentoAprovado)
}

// RejeitarOrcamento rejeita um orçamento
func (r *OrcamentoRepository) RejeitarOrcamento(id int, lojaID int) error {
	return r.AtualizarStatusOrcamento(id, lojaID, models.OrcamentoRejeitado)
}

// ============================================
// MÉTRICAS
// ============================================

// CalcularTotalOrcamentosMes calcula o total de orçamentos do mês
func (r *OrcamentoRepository) CalcularTotalOrcamentosMes(lojaID int) (float64, error) {
	var total float64
	query := `
		SELECT COALESCE(SUM(total), 0)
		FROM orcamentos
		WHERE loja_id = ? AND status = ? 
		AND MONTH(created_at) = MONTH(CURDATE()) 
		AND YEAR(created_at) = YEAR(CURDATE())
	`
	err := r.db.QueryRow(query, lojaID, models.OrcamentoAprovado).Scan(&total)
	return total, err
}

// ContarOrcamentosPorStatus conta orçamentos por status
func (r *OrcamentoRepository) ContarOrcamentosPorStatus(lojaID int, status string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM orcamentos WHERE loja_id = ? AND status = ?`
	err := r.db.QueryRow(query, lojaID, status).Scan(&count)
	return count, err
}

// ContarOrcamentosHoje conta orçamentos de hoje
func (r *OrcamentoRepository) ContarOrcamentosHoje(lojaID int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM orcamentos 
		WHERE loja_id = ? AND DATE(created_at) = CURDATE()
	`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}