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

// ============================================
// CRUD BÁSICO
// ============================================

// CriarAtendimento cria um novo atendimento
func (r *AtendimentoRepository) CriarAtendimento(atendimento *models.Atendimento) error {
	query := `
		INSERT INTO atendimentos (cliente_id, loja_id, status, iniciado_em)
		VALUES (?, ?, ?, NOW())
	`

	result, err := r.db.Exec(query, atendimento.ClienteID, atendimento.LojaID, models.StatusAguardando)
	if err != nil {
		return fmt.Errorf("erro ao criar atendimento: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do atendimento: %w", err)
	}

	atendimento.ID = int(id)
	atendimento.Status = models.StatusAguardando
	atendimento.IniciadoEm = time.Now()
	return nil
}

// BuscarAtendimentoPorID busca um atendimento pelo ID
func (r *AtendimentoRepository) BuscarAtendimentoPorID(id int) (*models.Atendimento, error) {
	query := `
		SELECT a.id, a.cliente_id, a.loja_id, a.status, a.iniciado_em,
		       a.finalizado_em, a.atendente_id, a.tempo_espera, a.tempo_atendimento,
		       c.id, c.nome, c.telefone, c.email
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.id = ?
	`

	var a models.Atendimento
	var c models.Cliente
	var finalizadoEm sql.NullTime
	var atendenteID sql.NullInt64
	var tempoEspera, tempoAtendimento sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.ClienteID, &a.LojaID, &a.Status,
		&a.IniciadoEm, &finalizadoEm, &atendenteID,
		&tempoEspera, &tempoAtendimento,
		&c.ID, &c.Nome, &c.Telefone, &c.Email,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar atendimento: %w", err)
	}

	if finalizadoEm.Valid {
		a.FinalizadoEm = &finalizadoEm.Time
	}
	if atendenteID.Valid {
		id := int(atendenteID.Int64)
		a.AtendenteID = &id
	}
	if tempoEspera.Valid {
		// tempo_espera não está mapeado no modelo atual; ignorando
	}
	if tempoAtendimento.Valid {
		// tempo_atendimento não está mapeado no modelo atual
	}

	a.Cliente = &c
	return &a, nil
}

// BuscarAtendimentosAtivos busca atendimentos em andamento ou aguardando
func (r *AtendimentoRepository) BuscarAtendimentosAtivos(lojaID int) ([]*models.Atendimento, error) {
	query := `
		SELECT a.id, a.cliente_id, a.loja_id, a.status, a.iniciado_em,
		       a.finalizado_em, a.atendente_id,
		       c.id, c.nome, c.telefone, c.email
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.loja_id = ? AND a.status IN (?, ?)
		ORDER BY a.iniciado_em ASC
	`

	rows, err := r.db.Query(query, lojaID, models.StatusAguardando, models.StatusEmAtendimento)
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

// BuscarAtendimentosPorCliente busca atendimentos de um cliente específico
func (r *AtendimentoRepository) BuscarAtendimentosPorCliente(clienteID int, limit int) ([]*models.Atendimento, error) {
	if limit == 0 {
		limit = 10
	}

	query := `
		SELECT id, cliente_id, loja_id, status, iniciado_em,
		       finalizado_em, atendente_id
		FROM atendimentos
		WHERE cliente_id = ?
		ORDER BY iniciado_em DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, clienteID, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos do cliente: %w", err)
	}
	defer rows.Close()

	var atendimentos []*models.Atendimento
	for rows.Next() {
		var a models.Atendimento
		var finalizadoEm sql.NullTime
		var atendenteID sql.NullInt64

		err := rows.Scan(
			&a.ID, &a.ClienteID, &a.LojaID, &a.Status,
			&a.IniciadoEm, &finalizadoEm, &atendenteID,
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

		atendimentos = append(atendimentos, &a)
	}

	return atendimentos, nil
}

// BuscarAtendimentosPorStatus busca atendimentos por status
func (r *AtendimentoRepository) BuscarAtendimentosPorStatus(lojaID int, status string) ([]*models.Atendimento, error) {
	query := `
		SELECT a.id, a.cliente_id, a.loja_id, a.status, a.iniciado_em,
		       a.finalizado_em, a.atendente_id,
		       c.id, c.nome, c.telefone, c.email
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.loja_id = ? AND a.status = ?
		ORDER BY a.iniciado_em DESC
	`

	rows, err := r.db.Query(query, lojaID, status)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos por status: %w", err)
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

// ============================================
// ATUALIZAÇÕES
// ============================================

// AtualizarStatusAtendimento atualiza o status de um atendimento
func (r *AtendimentoRepository) AtualizarStatusAtendimento(atendimentoID int, status string, atendenteID *int) error {
	query := `UPDATE atendimentos SET status = ?`
	args := []interface{}{status}

	if atendenteID != nil {
		query += `, atendente_id = ?`
		args = append(args, *atendenteID)
	}

	if status == models.StatusFinalizado {
		query += `, finalizado_em = NOW()`
	}

	query += ` WHERE id = ?`
	args = append(args, atendimentoID)

	_, err := r.db.Exec(query, args...)
	return err
}

// IniciarAtendimento marca um atendimento como em andamento
func (r *AtendimentoRepository) IniciarAtendimento(atendimentoID int, atendenteID int) error {
	_, err := r.db.Exec(`
		UPDATE atendimentos 
		SET status = ?, atendente_id = ?
		WHERE id = ?
	`, models.StatusEmAtendimento, atendenteID, atendimentoID)
	return err
}

// FinalizarAtendimento finaliza um atendimento
func (r *AtendimentoRepository) FinalizarAtendimento(atendimentoID int) error {
	_, err := r.db.Exec(`
		UPDATE atendimentos 
		SET status = ?, finalizado_em = NOW()
		WHERE id = ?
	`, models.StatusFinalizado, atendimentoID)
	return err
}

// AbandonarAtendimento marca um atendimento como abandonado
func (r *AtendimentoRepository) AbandonarAtendimento(atendimentoID int) error {
	_, err := r.db.Exec(`
		UPDATE atendimentos 
		SET status = ?, finalizado_em = NOW()
		WHERE id = ?
	`, models.StatusAbandonado, atendimentoID)
	return err
}

// ============================================
// MÉTRICAS
// ============================================

// ContarAtendimentosPorStatus conta atendimentos por status
func (r *AtendimentoRepository) ContarAtendimentosPorStatus(lojaID int, status string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = ?`
	err := r.db.QueryRow(query, lojaID, status).Scan(&count)
	return count, err
}

// CalcularTempoMedioAtendimento calcula o tempo médio de atendimento
func (r *AtendimentoRepository) CalcularTempoMedioAtendimento(lojaID int) (float64, error) {
	var tempoMedio sql.NullFloat64
	query := `
		SELECT AVG(TIMESTAMPDIFF(SECOND, iniciado_em, finalizado_em))
		FROM atendimentos
		WHERE loja_id = ? AND status = ? AND finalizado_em IS NOT NULL
	`
	err := r.db.QueryRow(query, lojaID, models.StatusFinalizado).Scan(&tempoMedio)
	if err != nil {
		return 0, err
	}
	if !tempoMedio.Valid {
		return 0, nil
	}
	return tempoMedio.Float64, nil
}

// CalcularTempoMedioEspera calcula o tempo médio de espera
func (r *AtendimentoRepository) CalcularTempoMedioEspera(lojaID int) (int, error) {
	var tempoMedio sql.NullInt64
	query := `
		SELECT AVG(tempo_espera)
		FROM atendimentos
		WHERE loja_id = ? AND status IN (?, ?) AND tempo_espera IS NOT NULL
	`
	err := r.db.QueryRow(query, lojaID, models.StatusEmAtendimento, models.StatusFinalizado).Scan(&tempoMedio)
	if err != nil {
		return 0, err
	}
	if !tempoMedio.Valid {
		return 0, nil
	}
	return int(tempoMedio.Int64), nil
}

// ContarAtendimentosHoje conta atendimentos de hoje
func (r *AtendimentoRepository) ContarAtendimentosHoje(lojaID int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
	`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}

// ContarAtendimentosMes conta atendimentos do mês
func (r *AtendimentoRepository) ContarAtendimentosMes(lojaID int) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND MONTH(iniciado_em) = MONTH(CURDATE()) 
		AND YEAR(iniciado_em) = YEAR(CURDATE())
	`
	err := r.db.QueryRow(query, lojaID).Scan(&count)
	return count, err
}

// ============================================
// RELATÓRIOS
// ============================================

// ObterMetricasDiarias obtém métricas diárias de atendimento
func (r *AtendimentoRepository) ObterMetricasDiarias(lojaID int, dias int) ([]map[string]interface{}, error) {
	if dias == 0 {
		dias = 7
	}

	query := `
		SELECT 
			DATE(iniciado_em) as data,
			COUNT(*) as total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as finalizados,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as abandonados,
			AVG(TIMESTAMPDIFF(SECOND, iniciado_em, finalizado_em)) as tempo_medio
		FROM atendimentos
		WHERE loja_id = ? AND iniciado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY DATE(iniciado_em)
		ORDER BY data DESC
	`

	rows, err := r.db.Query(query, models.StatusFinalizado, models.StatusAbandonado, lojaID, dias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []map[string]interface{}
	for rows.Next() {
		var data string
		var total, finalizados, abandonados int
		var tempoMedio sql.NullFloat64

		if err := rows.Scan(&data, &total, &finalizados, &abandonados, &tempoMedio); err != nil {
			continue
		}

		resultado := map[string]interface{}{
			"data":         data,
			"total":        total,
			"finalizados":  finalizados,
			"abandonados":  abandonados,
			"taxa_sucesso": float64(finalizados) / float64(total+1) * 100,
		}
		if tempoMedio.Valid {
			resultado["tempo_medio"] = tempoMedio.Float64
		}
		resultados = append(resultados, resultado)
	}

	return resultados, nil
}
