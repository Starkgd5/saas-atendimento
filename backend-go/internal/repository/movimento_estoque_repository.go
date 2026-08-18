package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type MovimentoEstoqueRepository struct {
	db *sql.DB
}

func NewMovimentoEstoqueRepository(db *sql.DB) *MovimentoEstoqueRepository {
	return &MovimentoEstoqueRepository{db: db}
}

// CriarMovimento cria um novo movimento de estoque
func (r *MovimentoEstoqueRepository) CriarMovimento(movimento *models.MovimentoEstoque) error {
	query := `
		INSERT INTO movimentos_estoque (
			loja_id, produto_id, lote_id, tipo, quantidade,
			saldo_anterior, saldo_atual, motivo, documento, usuario_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		movimento.LojaID, movimento.ProdutoID, movimento.LoteID,
		movimento.Tipo, movimento.Quantidade,
		movimento.SaldoAnterior, movimento.SaldoAtual,
		movimento.Motivo, movimento.Documento, movimento.UsuarioID,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar movimento de estoque: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do movimento: %w", err)
	}

	movimento.ID = int(id)
	movimento.CreatedAt = time.Now()

	return nil
}

// ListarMovimentosPorProduto lista movimentos de um produto
func (r *MovimentoEstoqueRepository) ListarMovimentosPorProduto(produtoID int, lojaID int, limit int) ([]*models.MovimentoEstoque, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT id, loja_id, produto_id, lote_id, tipo, quantidade,
			saldo_anterior, saldo_atual, motivo, documento, usuario_id, created_at
		FROM movimentos_estoque
		WHERE produto_id = ? AND loja_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, produtoID, lojaID, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar movimentos: %w", err)
	}
	defer rows.Close()

	var movimentos []*models.MovimentoEstoque
	for rows.Next() {
		var m models.MovimentoEstoque
		var loteID sql.NullInt64

		err := rows.Scan(
			&m.ID, &m.LojaID, &m.ProdutoID, &loteID,
			&m.Tipo, &m.Quantidade,
			&m.SaldoAnterior, &m.SaldoAtual,
			&m.Motivo, &m.Documento, &m.UsuarioID, &m.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear movimento: %w", err)
		}

		if loteID.Valid {
			m.LoteID = int(loteID.Int64)
		}

		movimentos = append(movimentos, &m)
	}

	return movimentos, nil
}
