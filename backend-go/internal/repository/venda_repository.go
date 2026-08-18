package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type VendaRepository struct {
	db *sql.DB
}

func NewVendaRepository(db *sql.DB) *VendaRepository {
	return &VendaRepository{db: db}
}

// CriarVenda cria uma nova venda
func (r *VendaRepository) CriarVenda(venda *models.Venda) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	// Gerar número da venda
	numeroVenda, err := r.gerarNumeroVenda()
	if err != nil {
		return fmt.Errorf("erro ao gerar número da venda: %w", err)
	}
	venda.NumeroVenda = numeroVenda

	// Inserir venda
	query := `
		INSERT INTO vendas (
			loja_id, cliente_id, atendente_id, farmaceutico_id, numero_venda,
			tipo_pagamento, status, subtotal, desconto, total, receita_anexada, observacao
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		venda.LojaID, venda.ClienteID, venda.AtendenteID, venda.FarmaceuticoID,
		venda.NumeroVenda, venda.TipoPagamento, "Pendente",
		venda.Subtotal, venda.Desconto, venda.Total,
		venda.ReceitaAnexada, venda.Observacao,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar venda: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID da venda: %w", err)
	}
	venda.ID = int(id)
	venda.CreatedAt = time.Now()
	venda.UpdatedAt = time.Now()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commit transação: %w", err)
	}

	return nil
}

// criarItemVenda insere um item de venda
func (r *VendaRepository) criarItemVenda(tx *sql.Tx, item *models.ItemVenda) error {
	query := `
		INSERT INTO itens_venda (venda_id, produto_id, lote_id, quantidade, preco_unit, total)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		item.VendaID, item.ProdutoID, item.LoteID,
		item.Quantidade, item.PrecoUnit, item.Total,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = int(id)

	return nil
}

// gerarNumeroVenda gera um número único para venda
func (r *VendaRepository) gerarNumeroVenda() (string, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) + 1 FROM vendas WHERE DATE(created_at) = CURDATE()
	`).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	data := time.Now().Format("20060102")
	return fmt.Sprintf("V%s%04d", data, count), nil
}

// BuscarVendaPorID busca uma venda pelo ID
func (r *VendaRepository) BuscarVendaPorID(id int) (*models.Venda, error) {
	query := `
		SELECT id, loja_id, cliente_id, atendente_id, farmaceutico_id,
			numero_venda, tipo_pagamento, status, subtotal, desconto,
			total, receita_anexada, observacao, created_at, updated_at
		FROM vendas
		WHERE id = ?
	`

	var venda models.Venda
	var clienteID, farmaceuticoID sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&venda.ID, &venda.LojaID, &clienteID, &venda.AtendenteID, &farmaceuticoID,
		&venda.NumeroVenda, &venda.TipoPagamento, &venda.Status,
		&venda.Subtotal, &venda.Desconto, &venda.Total,
		&venda.ReceitaAnexada, &venda.Observacao, &venda.CreatedAt, &venda.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if clienteID.Valid {
		id := int(clienteID.Int64)
		venda.ClienteID = &id
	}
	if farmaceuticoID.Valid {
		id := int(farmaceuticoID.Int64)
		venda.FarmaceuticoID = &id
	}

	return &venda, nil
}

// BuscarItensVenda busca os itens de uma venda
func (r *VendaRepository) BuscarItensVenda(vendaID int) ([]models.ItemVenda, error) {
	query := `
		SELECT id, venda_id, produto_id, lote_id, quantidade, preco_unit, total
		FROM itens_venda
		WHERE venda_id = ?
	`

	rows, err := r.db.Query(query, vendaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var itens []models.ItemVenda
	for rows.Next() {
		var item models.ItemVenda
		err := rows.Scan(
			&item.ID, &item.VendaID, &item.ProdutoID, &item.LoteID,
			&item.Quantidade, &item.PrecoUnit, &item.Total,
		)
		if err != nil {
			return nil, err
		}
		itens = append(itens, item)
	}

	return itens, nil
}

// ListarVendas lista vendas com filtros
func (r *VendaRepository) ListarVendas(lojaID int, status string, dataInicio, dataFim string, limit, offset int) ([]*models.Venda, int, error) {
	if limit == 0 {
		limit = 50
	}

	args := []interface{}{lojaID}
	whereClause := "WHERE loja_id = ?"

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	if dataInicio != "" {
		whereClause += " AND DATE(created_at) >= ?"
		args = append(args, dataInicio)
	}

	if dataFim != "" {
		whereClause += " AND DATE(created_at) <= ?"
		args = append(args, dataFim)
	}

	// Contar total
	countQuery := `SELECT COUNT(*) FROM vendas ` + whereClause
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Buscar vendas
	query := `
		SELECT id, loja_id, cliente_id, atendente_id, farmaceutico_id,
			numero_venda, tipo_pagamento, status, subtotal, desconto,
			total, receita_anexada, observacao, created_at, updated_at
		FROM vendas ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var vendas []*models.Venda
	for rows.Next() {
		var v models.Venda
		var clienteID, farmaceuticoID sql.NullInt64

		err := rows.Scan(
			&v.ID, &v.LojaID, &clienteID, &v.AtendenteID, &farmaceuticoID,
			&v.NumeroVenda, &v.TipoPagamento, &v.Status,
			&v.Subtotal, &v.Desconto, &v.Total,
			&v.ReceitaAnexada, &v.Observacao, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if clienteID.Valid {
			id := int(clienteID.Int64)
			v.ClienteID = &id
		}
		if farmaceuticoID.Valid {
			id := int(farmaceuticoID.Int64)
			v.FarmaceuticoID = &id
		}

		vendas = append(vendas, &v)
	}

	return vendas, total, nil
}

// AtualizarStatusVenda atualiza o status de uma venda
func (r *VendaRepository) AtualizarStatusVenda(id int, status string) error {
	_, err := r.db.Exec(`
		UPDATE vendas SET status = ?, updated_at = NOW()
		WHERE id = ?
	`, status, id)
	return err
}
