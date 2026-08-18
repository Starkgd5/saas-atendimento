package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type LoteRepository struct {
	db *sql.DB
}

func NewLoteRepository(db *sql.DB) *LoteRepository {
	return &LoteRepository{db: db}
}

// CriarLote cria um novo lote
func (r *LoteRepository) CriarLote(lote *models.Lote) error {
	query := `
		INSERT INTO lotes (
			produto_id, loja_id, numero_lote, data_fabricacao, data_validade,
			quantidade, quantidade_inicial, preco_custo, preco_venda, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		lote.ProdutoID, lote.LojaID, lote.NumeroLote,
		lote.DataFabricacao, lote.DataValidade,
		lote.Quantidade, lote.QuantidadeInicial,
		lote.PrecoCusto, lote.PrecoVenda,
		"Ativo",
	)
	if err != nil {
		return fmt.Errorf("erro ao criar lote: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do lote: %w", err)
	}

	lote.ID = int(id)
	lote.CreatedAt = time.Now()
	lote.UpdatedAt = time.Now()

	return nil
}

// BuscarLotePorID busca um lote pelo ID
func (r *LoteRepository) BuscarLotePorID(id int) (*models.Lote, error) {
	query := `
		SELECT id, produto_id, loja_id, numero_lote, data_fabricacao,
			data_validade, quantidade, quantidade_inicial, preco_custo,
			preco_venda, status, created_at, updated_at
		FROM lotes
		WHERE id = ?
	`

	var lote models.Lote

	err := r.db.QueryRow(query, id).Scan(
		&lote.ID, &lote.ProdutoID, &lote.LojaID, &lote.NumeroLote,
		&lote.DataFabricacao, &lote.DataValidade,
		&lote.Quantidade, &lote.QuantidadeInicial,
		&lote.PrecoCusto, &lote.PrecoVenda,
		&lote.Status, &lote.CreatedAt, &lote.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar lote: %w", err)
	}

	return &lote, nil
}

// BuscarLotesPorProduto busca todos os lotes de um produto
func (r *LoteRepository) BuscarLotesPorProduto(produtoID int, lojaID int) ([]*models.Lote, error) {
	query := `
		SELECT id, produto_id, loja_id, numero_lote, data_fabricacao,
			data_validade, quantidade, quantidade_inicial, preco_custo,
			preco_venda, status, created_at, updated_at
		FROM lotes
		WHERE produto_id = ? AND loja_id = ? AND status != 'Baixado'
		ORDER BY data_validade ASC
	`

	rows, err := r.db.Query(query, produtoID, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar lotes: %w", err)
	}
	defer rows.Close()

	var lotes []*models.Lote
	for rows.Next() {
		var l models.Lote
		err := rows.Scan(
			&l.ID, &l.ProdutoID, &l.LojaID, &l.NumeroLote,
			&l.DataFabricacao, &l.DataValidade,
			&l.Quantidade, &l.QuantidadeInicial,
			&l.PrecoCusto, &l.PrecoVenda,
			&l.Status, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear lote: %w", err)
		}
		lotes = append(lotes, &l)
	}

	return lotes, nil
}

// BuscarLotesValidos busca lotes com validade não vencida
func (r *LoteRepository) BuscarLotesValidos(produtoID int, lojaID int) ([]*models.Lote, error) {
	query := `
		SELECT id, produto_id, loja_id, numero_lote, data_fabricacao,
			data_validade, quantidade, quantidade_inicial, preco_custo,
			preco_venda, status, created_at, updated_at
		FROM lotes
		WHERE produto_id = ? AND loja_id = ? 
		AND status = 'Ativo' AND data_validade >= CURDATE()
		ORDER BY data_validade ASC
	`

	rows, err := r.db.Query(query, produtoID, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar lotes válidos: %w", err)
	}
	defer rows.Close()

	var lotes []*models.Lote
	for rows.Next() {
		var l models.Lote
		err := rows.Scan(
			&l.ID, &l.ProdutoID, &l.LojaID, &l.NumeroLote,
			&l.DataFabricacao, &l.DataValidade,
			&l.Quantidade, &l.QuantidadeInicial,
			&l.PrecoCusto, &l.PrecoVenda,
			&l.Status, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear lote: %w", err)
		}
		lotes = append(lotes, &l)
	}

	return lotes, nil
}

// AtualizarQuantidadeLote atualiza a quantidade de um lote
func (r *LoteRepository) AtualizarQuantidadeLote(loteID int, quantidade int) error {
	query := `UPDATE lotes SET quantidade = ?, updated_at = NOW() WHERE id = ?`
	_, err := r.db.Exec(query, quantidade, loteID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar quantidade do lote: %w", err)
	}
	return nil
}

// VerificarLotesVencidos verifica e atualiza lotes vencidos
func (r *LoteRepository) VerificarLotesVencidos() error {
	query := `
		UPDATE lotes 
		SET status = 'Vencido', updated_at = NOW()
		WHERE data_validade < CURDATE() AND status = 'Ativo'
	`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("erro ao verificar lotes vencidos: %w", err)
	}
	return nil
}
