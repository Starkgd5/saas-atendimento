package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type ReceitaRepository struct {
	db *sql.DB
}

func NewReceitaRepository(db *sql.DB) *ReceitaRepository {
	return &ReceitaRepository{db: db}
}

func (r *ReceitaRepository) CriarReceita(receita *models.ReceitaMedica) error {
	query := `
		INSERT INTO receitas_medicas (
			cliente_id, loja_id, venda_id, numero_receita, data_emissao,
			data_validade, medico_nome, medico_crm, medico_uf, cid, status, observacao, arquivo_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var vendaID interface{}
	if receita.VendaID != nil {
		vendaID = *receita.VendaID
	} else {
		vendaID = nil
	}

	result, err := r.db.Exec(query,
		receita.ClienteID, receita.LojaID, vendaID,
		receita.NumeroReceita, receita.DataEmissao,
		receita.DataValidade, receita.MedicoNome, receita.MedicoCRM,
		receita.MedicoUF, receita.Cid, "Pendente",
		receita.Observacao, receita.ArquivoURL,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar receita: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID da receita: %w", err)
	}
	receita.ID = int(id)
	receita.CreatedAt = time.Now()
	receita.UpdatedAt = time.Now()

	return nil
}

func (r *ReceitaRepository) BuscarReceitaPorID(id int) (*models.ReceitaMedica, error) {
	query := `
		SELECT id, cliente_id, loja_id, venda_id, numero_receita,
			data_emissao, data_validade, medico_nome, medico_crm,
			medico_uf, cid, status, data_validacao, farmaceutico_id,
			observacao, arquivo_url, created_at, updated_at
		FROM receitas_medicas
		WHERE id = ?
	`

	var receita models.ReceitaMedica
	var vendaID, farmaceuticoID sql.NullInt64
	var dataValidacao sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&receita.ID, &receita.ClienteID, &receita.LojaID, &vendaID,
		&receita.NumeroReceita, &receita.DataEmissao, &receita.DataValidade,
		&receita.MedicoNome, &receita.MedicoCRM, &receita.MedicoUF,
		&receita.Cid, &receita.Status, &dataValidacao, &farmaceuticoID,
		&receita.Observacao, &receita.ArquivoURL,
		&receita.CreatedAt, &receita.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar receita: %w", err)
	}

	if vendaID.Valid {
		id := int(vendaID.Int64)
		receita.VendaID = &id
	}
	if farmaceuticoID.Valid {
		id := int(farmaceuticoID.Int64)
		receita.FarmaceuticoID = &id
	}
	if dataValidacao.Valid {
		receita.DataValidacao = &dataValidacao.Time
	}

	return &receita, nil
}

func (r *ReceitaRepository) ListarReceitas(lojaID int, status string, limit, offset int) ([]*models.ReceitaMedica, error) {
	if limit == 0 {
		limit = 50
	}

	args := []interface{}{lojaID}
	whereClause := "WHERE loja_id = ?"

	if status != "" && status != "todos" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	query := `
		SELECT id, cliente_id, loja_id, numero_receita,
			data_emissao, data_validade, medico_nome, medico_crm,
			medico_uf, status, observacao, created_at
		FROM receitas_medicas ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar receitas: %w", err)
	}
	defer rows.Close()

	var receitas []*models.ReceitaMedica
	for rows.Next() {
		var r models.ReceitaMedica
		err := rows.Scan(
			&r.ID, &r.ClienteID, &r.LojaID, &r.NumeroReceita,
			&r.DataEmissao, &r.DataValidade, &r.MedicoNome, &r.MedicoCRM,
			&r.MedicoUF, &r.Status, &r.Observacao, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear receita: %w", err)
		}
		receitas = append(receitas, &r)
	}

	return receitas, nil
}

func (r *ReceitaRepository) AtualizarStatus(id int, status string, observacao string) error {
	_, err := r.db.Exec(`
		UPDATE receitas_medicas
		SET status = ?, observacao = CONCAT(observacao, '\n', ?), updated_at = NOW()
		WHERE id = ?
	`, status, observacao, id)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status da receita: %w", err)
	}
	return nil
}