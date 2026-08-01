package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type ReclamacaoRepository struct {
	db *sql.DB
}

func NewReclamacaoRepository(db *sql.DB) *ReclamacaoRepository {
	return &ReclamacaoRepository{db: db}
}

// ============================================
// CRUD BÁSICO
// ============================================

// BuscarReclamacaoPorID busca uma reclamação pelo ID
func (r *ReclamacaoRepository) BuscarReclamacaoPorID(id int, lojaID int) (*models.Reclamacao, error) {
	query := `
		SELECT id, cliente_id, loja_id, atendimento_id, mensagem, status,
		       prioridade, categoria, resposta, resolvido_em,
		       created_at, updated_at
		FROM reclamacoes
		WHERE id = ? AND loja_id = ?
	`

	var rec models.Reclamacao
	var atendimentoID sql.NullInt64
	var resolvidoEmTime sql.NullTime

	err := r.db.QueryRow(query, id, lojaID).Scan(
		&rec.ID,
		&rec.ClienteID,
		&rec.LojaID,
		&atendimentoID,
		&rec.Mensagem,
		&rec.Status,
		&rec.Prioridade,
		&rec.Categoria,
		&rec.Resposta,
		&resolvidoEmTime,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar reclamação: %w", err)
	}

	if atendimentoID.Valid {
		id := int(atendimentoID.Int64)
		rec.AtendimentoID = &id
	}
	if resolvidoEmTime.Valid {
		rec.ResolvidoEm = &resolvidoEmTime.Time
	}

	return &rec, nil
}

// ListarReclamacoes lista reclamações com paginação
func (r *ReclamacaoRepository) ListarReclamacoes(lojaID int, status string, prioridade string, limit, offset int) ([]*models.Reclamacao, int, error) {
	if limit == 0 {
		limit = 50
	}

	args := []interface{}{lojaID}
	whereClause := "WHERE loja_id = ?"

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if prioridade != "" {
		whereClause += " AND prioridade = ?"
		args = append(args, prioridade)
	}

	// Contar total
	countQuery := `SELECT COUNT(*) FROM reclamacoes ` + whereClause
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar reclamações: %w", err)
	}

	// Buscar reclamações
	query := `
		SELECT id, cliente_id, loja_id, atendimento_id, mensagem, status,
		       prioridade, categoria, resposta, resolvido_em,
		       created_at, updated_at
		FROM reclamacoes ` + whereClause + `
		ORDER BY 
			CASE prioridade 
				WHEN 'critica' THEN 1 
				WHEN 'alta' THEN 2 
				WHEN 'media' THEN 3 
				WHEN 'baixa' THEN 4 
			END,
			created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar reclamações: %w", err)
	}
	defer rows.Close()

	var reclamacoes []*models.Reclamacao
	for rows.Next() {
		var rec models.Reclamacao
		var atendimentoID sql.NullInt64
		var resolvidoEmTime sql.NullTime

		err := rows.Scan(
			&rec.ID,
			&rec.ClienteID,
			&rec.LojaID,
			&atendimentoID,
			&rec.Mensagem,
			&rec.Status,
			&rec.Prioridade,
			&rec.Categoria,
			&rec.Resposta,
			&resolvidoEmTime,
			&rec.CreatedAt,
			&rec.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear reclamação: %w", err)
		}

		if atendimentoID.Valid {
			id := int(atendimentoID.Int64)
			rec.AtendimentoID = &id
		}
		if resolvidoEmTime.Valid {
			rec.ResolvidoEm = &resolvidoEmTime.Time
		}

		reclamacoes = append(reclamacoes, &rec)
	}

	return reclamacoes, total, nil
}

// ListarReclamacoesPendentes lista reclamações pendentes
func (r *ReclamacaoRepository) ListarReclamacoesPendentes(lojaID int) ([]*models.Reclamacao, error) {
	reclamacoes, _, err := r.ListarReclamacoes(lojaID, models.ReclamacaoPendente, "", 100, 0)
	return reclamacoes, err
}

// ============================================
// CRIAÇÃO E ATUALIZAÇÃO
// ============================================

// CriarReclamacao cria uma nova reclamação
func (r *ReclamacaoRepository) CriarReclamacao(reclamacao *models.Reclamacao) error {
	query := `
		INSERT INTO reclamacoes (cliente_id, loja_id, atendimento_id, mensagem,
		                         status, prioridade, categoria)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	var atendimentoID interface{}
	if reclamacao.AtendimentoID != nil {
		atendimentoID = *reclamacao.AtendimentoID
	} else {
		atendimentoID = nil
	}

	result, err := r.db.Exec(query,
		reclamacao.ClienteID,
		reclamacao.LojaID,
		atendimentoID,
		reclamacao.Mensagem,
		reclamacao.Status,
		reclamacao.Prioridade,
		reclamacao.Categoria,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar reclamação: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID da reclamação: %w", err)
	}

	reclamacao.ID = int(id)
	reclamacao.CreatedAt = time.Now()
	reclamacao.UpdatedAt = time.Now()
	return nil
}

// AtualizarReclamacao atualiza uma reclamação existente
func (r *ReclamacaoRepository) AtualizarReclamacao(reclamacao *models.Reclamacao) error {
	query := `
		UPDATE reclamacoes 
		SET mensagem = ?, status = ?, prioridade = ?, categoria = ?,
		    resposta = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	result, err := r.db.Exec(query,
		reclamacao.Mensagem,
		reclamacao.Status,
		reclamacao.Prioridade,
		reclamacao.Categoria,
		reclamacao.Resposta,
		reclamacao.ID,
		reclamacao.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar reclamação: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("reclamação não encontrada")
	}

	return nil
}

// ResolverReclamacao resolve uma reclamação
func (r *ReclamacaoRepository) ResolverReclamacao(id int, lojaID int, resposta string) error {
	query := `
		UPDATE reclamacoes 
		SET status = ?, resposta = ?, resolvido_em = NOW(), updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`
	result, err := r.db.Exec(query, models.ReclamacaoResolvido, resposta, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao resolver reclamação: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("reclamação não encontrada")
	}

	return nil
}

// IgnorarReclamacao ignora uma reclamação
func (r *ReclamacaoRepository) IgnorarReclamacao(id int, lojaID int) error {
	query := `
		UPDATE reclamacoes 
		SET status = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`
	result, err := r.db.Exec(query, models.ReclamacaoIgnorado, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao ignorar reclamação: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("reclamação não encontrada")
	}

	return nil
}

// ============================================
// MÉTRICAS
// ============================================

// ContarReclamacoesPorStatus conta reclamações por status
func (r *ReclamacaoRepository) ContarReclamacoesPorStatus(lojaID int, status string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM reclamacoes WHERE loja_id = ? AND status = ?`
	err := r.db.QueryRow(query, lojaID, status).Scan(&count)
	return count, err
}

// ContarReclamacoesPendentes conta reclamações pendentes
func (r *ReclamacaoRepository) ContarReclamacoesPendentes(lojaID int) (int, error) {
	return r.ContarReclamacoesPorStatus(lojaID, models.ReclamacaoPendente)
}

// ContarReclamacoesPorPrioridade conta reclamações por prioridade
func (r *ReclamacaoRepository) ContarReclamacoesPorPrioridade(lojaID int, prioridade string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM reclamacoes WHERE loja_id = ? AND prioridade = ?`
	err := r.db.QueryRow(query, lojaID, prioridade).Scan(&count)
	return count, err
}
