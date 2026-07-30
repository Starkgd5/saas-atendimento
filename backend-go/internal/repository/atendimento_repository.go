package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type AtendimentoRepository struct {
	db *sql.DB
}

func NewAtendimentoRepository(db *sql.DB) *AtendimentoRepository {
	return &AtendimentoRepository{db: db}
}

// CriarAtendimento cria um novo atendimento
func (r *AtendimentoRepository) CriarAtendimento(atendimento *models.Atendimento) error {
	query := `
		INSERT INTO atendimentos (cliente_id, loja_id, status, iniciado_em)
		VALUES (?, ?, ?, NOW())
	`

	result, err := r.db.Exec(query, atendimento.ClienteID, atendimento.LojaID, "aguardando")
	if err != nil {
		return fmt.Errorf("erro ao criar atendimento: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do atendimento: %w", err)
	}

	atendimento.ID = int(id)
	atendimento.Status = "aguardando"
	atendimento.IniciadoEm = time.Now()
	return nil
}

// BuscarAtendimentosAtivos busca atendimentos em andamento ou aguardando
func (r *AtendimentoRepository) BuscarAtendimentosAtivos(lojaID int) ([]*models.Atendimento, error) {
	query := `
		SELECT a.id, a.cliente_id, a.loja_id, a.status, a.iniciado_em, 
		       a.finalizado_em, a.atendente_id,
		       c.id, c.nome, c.telefone, c.email
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.loja_id = ? AND a.status IN ('aguardando', 'em_atendimento')
		ORDER BY a.iniciado_em ASC
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos: %w", err)
	}
	defer rows.Close()

	var atendimentos []*models.Atendimento
	for rows.Next() {
		var a models.Atendimento
		var c models.Cliente
		var finalizadoEm sql.NullTime
		var atendenteID sql.NullInt64

		err := rows.Scan(
			&a.ID, &a.ClienteID, &a.LojaID, &a.Status,
			&a.IniciadoEm, &finalizadoEm, &atendenteID,
			&c.ID, &c.Nome, &c.Telefone, &c.Email,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear atendimento: %w", err)
		}

		if finalizadoEm.Valid {
			a.FinalizadoEm = &finalizadoEm.Time
		}
		if atendenteID.Valid {
			id := int(atendenteID.Int64)
			a.AtendenteID = &id
		}

		a.Cliente = &c
		atendimentos = append(atendimentos, &a)
	}

	return atendimentos, nil
}

// AtualizarStatusAtendimento atualiza o status de um atendimento
func (r *AtendimentoRepository) AtualizarStatusAtendimento(atendimentoID int, status string, atendenteID *int) error {
	query := `UPDATE atendimentos SET status = ?`
	args := []interface{}{status}

	if atendenteID != nil {
		query += `, atendente_id = ?`
		args = append(args, *atendenteID)
	}

	if status == "finalizado" {
		query += `, finalizado_em = NOW()`
	}

	query += ` WHERE id = ?`
	args = append(args, atendimentoID)

	_, err := r.db.Exec(query, args...)
	return err
}