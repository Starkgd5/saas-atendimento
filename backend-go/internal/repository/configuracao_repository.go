package repository

import (
	"database/sql"
	"fmt"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type ConfiguracaoRepository struct {
	db *sql.DB
}

func NewConfiguracaoRepository(db *sql.DB) *ConfiguracaoRepository {
	return &ConfiguracaoRepository{db: db}
}

func (r *ConfiguracaoRepository) ListarConfiguracoes(lojaID int) ([]models.Configuracao, error) {
	query := `
		SELECT id, loja_id, chave, valor, descricao, created_at, updated_at
		FROM configuracao_loja
		WHERE loja_id = ?
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar configurações: %w", err)
	}
	defer rows.Close()

	var configs []models.Configuracao
	for rows.Next() {
		var c models.Configuracao
		err := rows.Scan(
			&c.ID, &c.LojaID, &c.Chave, &c.Valor,
			&c.Descricao, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear configuração: %w", err)
		}
		configs = append(configs, c)
	}

	return configs, nil
}

func (r *ConfiguracaoRepository) BuscarConfiguracao(lojaID int, chave string) (*models.Configuracao, error) {
	query := `
		SELECT id, loja_id, chave, valor, descricao, created_at, updated_at
		FROM configuracao_loja
		WHERE loja_id = ? AND chave = ?
	`

	var c models.Configuracao
	err := r.db.QueryRow(query, lojaID, chave).Scan(
		&c.ID, &c.LojaID, &c.Chave, &c.Valor,
		&c.Descricao, &c.CreatedAt, &c.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar configuração: %w", err)
	}

	return &c, nil
}

func (r *ConfiguracaoRepository) AtualizarConfiguracao(lojaID int, chave string, valor string) error {
	query := `
		INSERT INTO configuracao_loja (loja_id, chave, valor, descricao)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE valor = ?, updated_at = NOW()
	`

	_, err := r.db.Exec(query, lojaID, chave, valor, "", valor)
	if err != nil {
		return fmt.Errorf("erro ao atualizar configuração: %w", err)
	}
	return nil
}