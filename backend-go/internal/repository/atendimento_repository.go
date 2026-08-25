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
	atendimento.CreatedAt = time.Now()
	atendimento.UpdatedAt = time.Now()
	return nil
}

// BuscarAtendimentoPorID busca um atendimento pelo ID
func (r *AtendimentoRepository) BuscarAtendimentoPorID(id int) (*models.Atendimento, error) {
	query := `
		SELECT a.id, a.cliente_id, a.loja_id, a.status, a.iniciado_em,
		       a.finalizado_em, a.atendente_id, a.tempo_espera, a.tempo_atendimento,
		       a.created_at, a.updated_at,
		       c.id, c.nome, c.telefone, c.email,
		       u.id, u.nome as atendente_nome
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		LEFT JOIN usuarios u ON a.atendente_id = u.id
		WHERE a.id = ?
	`

	var a models.Atendimento
	var c models.Cliente
	var u models.Usuario
	var atendenteID, tempoEspera, tempoAtendimento sql.NullInt64
	var finalizadoEmTime sql.NullTime
	var atendenteNome sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&a.ID, &a.ClienteID, &a.LojaID, &a.Status,
		&a.IniciadoEm, &finalizadoEmTime, &atendenteID,
		&tempoEspera, &tempoAtendimento,
		&a.CreatedAt, &a.UpdatedAt,
		&c.ID, &c.Nome, &c.Telefone, &c.Email,
		&u.ID, &atendenteNome,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar atendimento: %w", err)
	}

	if finalizadoEmTime.Valid {
		a.FinalizadoEm = &finalizadoEmTime.Time
	}
	if atendenteID.Valid {
		id := int(atendenteID.Int64)
		a.AtendenteID = &id
	}
	if tempoEspera.Valid {
		val := int(tempoEspera.Int64)
		a.TempoEspera = &val
	}
	if tempoAtendimento.Valid {
		val := int(tempoAtendimento.Int64)
		a.TempoAtendimento = &val
	}
	if atendenteNome.Valid {
		u.Nome = atendenteNome.String
		a.Atendente = &u
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
		return nil, fmt.Errorf("erro ao buscar atendimentos ativos: %w", err)
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

// ListarAtendimentos lista atendimentos com paginação
func (r *AtendimentoRepository) ListarAtendimentos(lojaID int, limit, offset int) ([]*models.AtendimentoResumo, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT a.id, a.cliente_id, c.nome, c.telefone, a.status,
		       a.iniciado_em, 
		       TIMESTAMPDIFF(SECOND, a.iniciado_em, NOW()) as tempo_decorrido,
		       (SELECT m.conteudo FROM mensagens m 
		        WHERE m.atendimento_id = a.id 
		        ORDER BY m.enviado_em DESC LIMIT 1) as ultima_mensagem,
		       (SELECT m.enviado_em FROM mensagens m 
		        WHERE m.atendimento_id = a.id 
		        ORDER BY m.enviado_em DESC LIMIT 1) as ultima_mensagem_em
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.loja_id = ?
		ORDER BY a.iniciado_em DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, lojaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar atendimentos: %w", err)
	}
	defer rows.Close()

	var atendimentos []*models.AtendimentoResumo
	for rows.Next() {
		var a models.AtendimentoResumo
		var ultimaMensagem sql.NullString
		var ultimaMensagemEm sql.NullTime

		err := rows.Scan(
			&a.ID, &a.ClienteID, &a.ClienteNome, &a.ClienteTelefone,
			&a.Status, &a.IniciadoEm, &a.TempoDecorrido,
			&ultimaMensagem, &ultimaMensagemEm,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear atendimento: %w", err)
		}

		if ultimaMensagem.Valid {
			a.UltimaMensagem = ultimaMensagem.String
		}
		if ultimaMensagemEm.Valid {
			a.UltimaMensagemEm = &ultimaMensagemEm.Time
		}

		atendimentos = append(atendimentos, &a)
	}

	return atendimentos, nil
}

// ============================================
// ATUALIZAÇÃO DE STATUS
// ============================================

// AtualizarStatusAtendimento atualiza o status de um atendimento
func (r *AtendimentoRepository) AtualizarStatusAtendimento(atendimentoID int, status string, atendenteID int) error {
	query := `UPDATE atendimentos SET status = ?`
	args := []interface{}{status}

	if atendenteID > 0 {
		query += `, atendente_id = ?`
		args = append(args, atendenteID)
	}

	if status == models.StatusFinalizado || status == models.StatusAbandonado {
		query += `, finalizado_em = NOW()`
	}

	query += `, updated_at = NOW() WHERE id = ?`
	args = append(args, atendimentoID)

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status do atendimento: %w", err)
	}
	return nil
}

// IniciarAtendimento marca um atendimento como em andamento
func (r *AtendimentoRepository) IniciarAtendimento(atendimentoID int, atendenteID int) error {
	_, err := r.db.Exec(`
		UPDATE atendimentos 
		SET status = ?, atendente_id = ?, tempo_espera = TIMESTAMPDIFF(SECOND, iniciado_em, NOW()), updated_at = NOW()
		WHERE id = ?
	`, models.StatusEmAtendimento, atendenteID, atendimentoID)
	if err != nil {
		return fmt.Errorf("erro ao iniciar atendimento: %w", err)
	}
	return nil
}

// FinalizarAtendimento finaliza um atendimento
func (r *AtendimentoRepository) FinalizarAtendimento(atendimentoID int) error {
	_, err := r.db.Exec(`
		UPDATE atendimentos 
		SET status = ?, finalizado_em = NOW(), 
		    tempo_atendimento = TIMESTAMPDIFF(SECOND, iniciado_em, NOW()), updated_at = NOW()
		WHERE id = ?
	`, models.StatusFinalizado, atendimentoID)
	if err != nil {
		return fmt.Errorf("erro ao finalizar atendimento: %w", err)
	}
	return nil
}

// ============================================
// MENSAGENS
// ============================================

// EnviarMensagem envia uma nova mensagem no atendimento
func (r *AtendimentoRepository) EnviarMensagem(mensagem *models.Mensagem) error {
	query := `
		INSERT INTO mensagens (atendimento_id, remetente, conteudo, tipo, arquivo_url, arquivo_nome, arquivo_tamanho, enviado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
	`

	result, err := r.db.Exec(query,
		mensagem.AtendimentoID, mensagem.Remetente, mensagem.Conteudo,
		mensagem.Tipo, mensagem.ArquivoURL, mensagem.ArquivoNome,
		mensagem.ArquivoTamanho,
	)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID da mensagem: %w", err)
	}

	mensagem.ID = int(id)
	mensagem.EnviadoEm = time.Now()
	mensagem.CreatedAt = time.Now()
	mensagem.UpdatedAt = time.Now()
	return nil
}

// ListarMensagens lista mensagens de um atendimento
func (r *AtendimentoRepository) ListarMensagens(atendimentoID int) ([]*models.Mensagem, error) {
	query := `
		SELECT id, atendimento_id, remetente, conteudo, tipo,
		       arquivo_url, arquivo_nome, arquivo_tamanho, lida, enviado_em,
		       created_at, updated_at
		FROM mensagens
		WHERE atendimento_id = ?
		ORDER BY enviado_em ASC
	`

	rows, err := r.db.Query(query, atendimentoID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar mensagens: %w", err)
	}
	defer rows.Close()

	var mensagens []*models.Mensagem
	for rows.Next() {
		var m models.Mensagem
		err := rows.Scan(
			&m.ID, &m.AtendimentoID, &m.Remetente, &m.Conteudo, &m.Tipo,
			&m.ArquivoURL, &m.ArquivoNome, &m.ArquivoTamanho, &m.Lida, &m.EnviadoEm,
			&m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear mensagem: %w", err)
		}
		mensagens = append(mensagens, &m)
	}

	return mensagens, nil
}

// BuscarMensagem busca uma mensagem por ID
func (r *AtendimentoRepository) BuscarMensagem(id int) (*models.Mensagem, error) {
	query := `
		SELECT id, atendimento_id, remetente, conteudo, tipo,
		       arquivo_url, arquivo_nome, arquivo_tamanho, lida, enviado_em,
		       created_at, updated_at
		FROM mensagens
		WHERE id = ?
	`

	var m models.Mensagem
	err := r.db.QueryRow(query, id).Scan(
		&m.ID, &m.AtendimentoID, &m.Remetente, &m.Conteudo, &m.Tipo,
		&m.ArquivoURL, &m.ArquivoNome, &m.ArquivoTamanho, &m.Lida, &m.EnviadoEm,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar mensagem: %w", err)
	}

	return &m, nil
}

// MarcarMensagemLida marca uma mensagem como lida
func (r *AtendimentoRepository) MarcarMensagemLida(id int) error {
	_, err := r.db.Exec(`UPDATE mensagens SET lida = TRUE, updated_at = NOW() WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("erro ao marcar mensagem como lida: %w", err)
	}
	return nil
}

// ============================================
// ESTATÍSTICAS
// ============================================

// GetAtendimentoStats retorna estatísticas de atendimentos
func (r *AtendimentoRepository) GetAtendimentoStats(lojaID int) (*models.AtendimentoStats, error) {
	var stats models.AtendimentoStats

	// Total de atendimentos
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ?
	`, lojaID).Scan(&stats.TotalAtendimentos)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar atendimentos: %w", err)
	}

	// Em atendimento
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusEmAtendimento).Scan(&stats.EmAtendimento)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar em atendimento: %w", err)
	}

	// Aguardando
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusAguardando).Scan(&stats.Aguardando)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar aguardando: %w", err)
	}

	// Finalizados
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusFinalizado).Scan(&stats.Finalizados)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar finalizados: %w", err)
	}

	// Abandonados
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusAbandonado).Scan(&stats.Abandonados)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar abandonados: %w", err)
	}

	// Tempo médio de espera
	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(tempo_espera), 0) FROM atendimentos 
		WHERE loja_id = ? AND tempo_espera IS NOT NULL
	`, lojaID).Scan(&stats.TempoMedioEspera)
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular tempo médio de espera: %w", err)
	}

	// Tempo médio de atendimento
	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(tempo_atendimento), 0) FROM atendimentos 
		WHERE loja_id = ? AND tempo_atendimento IS NOT NULL
	`, lojaID).Scan(&stats.TempoMedioAtendimento)
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular tempo médio de atendimento: %w", err)
	}

	// Ticket médio
	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(o.total), 0) FROM orcamentos o
		JOIN atendimentos a ON o.atendimento_id = a.id
		WHERE a.loja_id = ? AND o.status = 'aprovado'
	`, lojaID).Scan(&stats.TicketMedio)
	if err != nil {
		return nil, fmt.Errorf("erro ao calcular ticket médio: %w", err)
	}

	return &stats, nil
}