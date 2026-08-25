package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type CompraRepository struct {
	db *sql.DB
}

func NewCompraRepository(db *sql.DB) *CompraRepository {
	return &CompraRepository{db: db}
}

// ============================================
// CRUD BÁSICO
// ============================================

// CriarCompra cria uma nova compra
func (r *CompraRepository) CriarCompra(compra *models.Compra) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	// Inserir compra
	query := `
		INSERT INTO compras (
			loja_id, fornecedor_id, numero_nf, data_emissao, data_recebimento,
			status, subtotal, desconto, total, observacao
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var dataRecebimento interface{}
	if compra.DataRecebimento != nil {
		dataRecebimento = compra.DataRecebimento
	} else {
		dataRecebimento = nil
	}

	result, err := tx.Exec(query,
		compra.LojaID, compra.FornecedorID, compra.NumeroNF,
		compra.DataEmissao, dataRecebimento,
		"Em Andamento", compra.Subtotal, compra.Desconto,
		compra.Total, compra.Observacao,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar compra: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID da compra: %w", err)
	}
	compra.ID = int(id)
	compra.CreatedAt = time.Now()
	compra.UpdatedAt = time.Now()

	// Inserir itens
	for _, item := range compra.Itens {
		item.CompraID = compra.ID
		if err := r.criarItemCompra(tx, &item); err != nil {
			return fmt.Errorf("erro ao criar item da compra: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commit transação: %w", err)
	}

	return nil
}

// criarItemCompra insere um item de compra
func (r *CompraRepository) criarItemCompra(tx *sql.Tx, item *models.ItemCompra) error {
	query := `
		INSERT INTO itens_compra (compra_id, produto_id, quantidade, preco_unit, total)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := tx.Exec(query,
		item.CompraID, item.ProdutoID, item.Quantidade,
		item.PrecoUnit, item.Total,
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

// BuscarCompraPorID busca uma compra pelo ID
func (r *CompraRepository) BuscarCompraPorID(id int) (*models.Compra, error) {
	query := `
		SELECT c.id, c.loja_id, c.fornecedor_id, c.numero_nf, c.data_emissao,
		       c.data_recebimento, c.status, c.subtotal, c.desconto, c.total,
		       c.observacao, c.created_at, c.updated_at,
		       f.razao_social, f.nome_fantasia, f.cnpj, f.telefone, f.email
		FROM compras c
		LEFT JOIN fornecedores f ON c.fornecedor_id = f.id
		WHERE c.id = ?
	`

	var compra models.Compra
	var fornecedor models.Fornecedor
	var dataRecebimento sql.NullTime
	var razaoSocial, nomeFantasia, cnpj, telefone, email sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&compra.ID, &compra.LojaID, &compra.FornecedorID, &compra.NumeroNF,
		&compra.DataEmissao, &dataRecebimento, &compra.Status,
		&compra.Subtotal, &compra.Desconto, &compra.Total,
		&compra.Observacao, &compra.CreatedAt, &compra.UpdatedAt,
		&razaoSocial, &nomeFantasia, &cnpj, &telefone, &email,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("erro ao buscar compra: %w", err)
	}

	if dataRecebimento.Valid {
		compra.DataRecebimento = &dataRecebimento.Time
	}

	// Preencher fornecedor se disponível
	if razaoSocial.Valid {
		fornecedor.RazaoSocial = razaoSocial.String
		fornecedor.NomeFantasia = nomeFantasia.String
		fornecedor.CNPJ = cnpj.String
		fornecedor.Telefone = telefone.String
		fornecedor.Email = email.String
		compra.Fornecedor = &fornecedor
	}

	// Buscar itens da compra
	itens, err := r.BuscarItensCompra(id)
	if err != nil {
		return nil, err
	}
	compra.Itens = itens

	return &compra, nil
}

// BuscarItensCompra busca os itens de uma compra
func (r *CompraRepository) BuscarItensCompra(compraID int) ([]models.ItemCompra, error) {
	query := `
		SELECT ic.id, ic.compra_id, ic.produto_id, ic.quantidade, ic.preco_unit, ic.total,
		       p.nome, p.codigo_barras, p.categoria
		FROM itens_compra ic
		LEFT JOIN produtos p ON ic.produto_id = p.id
		WHERE ic.compra_id = ?
	`

	rows, err := r.db.Query(query, compraID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar itens da compra: %w", err)
	}
	defer rows.Close()

	var itens []models.ItemCompra
	for rows.Next() {
		var item models.ItemCompra
		var produtoNome, produtoCodigo, produtoCategoria sql.NullString

		err := rows.Scan(
			&item.ID, &item.CompraID, &item.ProdutoID,
			&item.Quantidade, &item.PrecoUnit, &item.Total,
			&produtoNome, &produtoCodigo, &produtoCategoria,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear item: %w", err)
		}

		// Criar objeto produto se disponível
		if produtoNome.Valid {
			produto := &models.Produto{
				Nome:       produtoNome.String,
				CodigoBarras: produtoCodigo.String,
				Categoria:  produtoCategoria.String,
			}
			item.Produto = produto
		}

		itens = append(itens, item)
	}

	return itens, nil
}

// ListarCompras lista todas as compras de uma loja
func (r *CompraRepository) ListarCompras(lojaID int) ([]*models.Compra, error) {
	query := `
		SELECT c.id, c.numero_nf, c.status, c.total, c.data_emissao, c.data_recebimento,
		       c.created_at, f.razao_social, f.id as fornecedor_id
		FROM compras c
		LEFT JOIN fornecedores f ON c.fornecedor_id = f.id
		WHERE c.loja_id = ?
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar compras: %w", err)
	}
	defer rows.Close()

	var compras []*models.Compra
	for rows.Next() {
		var c models.Compra
		var dataRecebimento sql.NullTime
		var razaoSocial sql.NullString
		var fornecedorID sql.NullInt64

		err := rows.Scan(
			&c.ID, &c.NumeroNF, &c.Status, &c.Total,
			&c.DataEmissao, &dataRecebimento, &c.CreatedAt,
			&razaoSocial, &fornecedorID,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear compra: %w", err)
		}

		if dataRecebimento.Valid {
			c.DataRecebimento = &dataRecebimento.Time
		}
		if razaoSocial.Valid {
			c.Fornecedor = &models.Fornecedor{
				ID:          int(fornecedorID.Int64),
				RazaoSocial: razaoSocial.String,
			}
		}

		compras = append(compras, &c)
	}

	return compras, nil
}

// ListarComprasPorStatus lista compras por status
func (r *CompraRepository) ListarComprasPorStatus(lojaID int, status string) ([]*models.Compra, error) {
	query := `
		SELECT c.id, c.numero_nf, c.status, c.total, c.data_emissao, c.data_recebimento,
		       c.created_at, f.razao_social
		FROM compras c
		LEFT JOIN fornecedores f ON c.fornecedor_id = f.id
		WHERE c.loja_id = ? AND c.status = ?
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(query, lojaID, status)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar compras por status: %w", err)
	}
	defer rows.Close()

	var compras []*models.Compra
	for rows.Next() {
		var c models.Compra
		var dataRecebimento sql.NullTime
		var razaoSocial sql.NullString

		err := rows.Scan(
			&c.ID, &c.NumeroNF, &c.Status, &c.Total,
			&c.DataEmissao, &dataRecebimento, &c.CreatedAt,
			&razaoSocial,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear compra: %w", err)
		}

		if dataRecebimento.Valid {
			c.DataRecebimento = &dataRecebimento.Time
		}

		compras = append(compras, &c)
	}

	return compras, nil
}

// ============================================
// ATUALIZAÇÕES
// ============================================

// AtualizarCompra atualiza uma compra existente
func (r *CompraRepository) AtualizarCompra(compra *models.Compra) error {
	query := `
		UPDATE compras SET
			numero_nf = ?, data_emissao = ?, desconto = ?,
			total = ?, observacao = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	_, err := r.db.Exec(query,
		compra.NumeroNF, compra.DataEmissao, compra.Desconto,
		compra.Total, compra.Observacao,
		compra.ID, compra.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar compra: %w", err)
	}
	return nil
}

// ReceberCompra marca uma compra como recebida
func (r *CompraRepository) ReceberCompra(id int) error {
	_, err := r.db.Exec(`
		UPDATE compras 
		SET status = 'Recebido', data_recebimento = NOW(), updated_at = NOW()
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("erro ao receber compra: %w", err)
	}
	return nil
}

// CancelarCompra cancela uma compra
func (r *CompraRepository) CancelarCompra(id int) error {
	_, err := r.db.Exec(`
		UPDATE compras 
		SET status = 'Cancelado', updated_at = NOW()
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("erro ao cancelar compra: %w", err)
	}
	return nil
}

// ============================================
// ESTATÍSTICAS E RELATÓRIOS
// ============================================

// GetTotalCompras retorna o total de compras
func (r *CompraRepository) GetTotalCompras(lojaID int) (int, error) {
	var total int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM compras WHERE loja_id = ?
	`, lojaID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("erro ao contar compras: %w", err)
	}
	return total, nil
}

// GetTotalComprasMes retorna o total de compras do mês
func (r *CompraRepository) GetTotalComprasMes(lojaID int) (float64, error) {
	var total float64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(total), 0) FROM compras 
		WHERE loja_id = ? AND MONTH(created_at) = MONTH(CURDATE()) 
		AND YEAR(created_at) = YEAR(CURDATE())
	`, lojaID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("erro ao calcular total de compras do mês: %w", err)
	}
	return total, nil
}

// GetComprasPorFornecedor retorna compras agrupadas por fornecedor
func (r *CompraRepository) GetComprasPorFornecedor(lojaID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			f.id, f.razao_social,
			COUNT(c.id) as total_compras,
			COALESCE(SUM(c.total), 0) as total_gasto
		FROM fornecedores f
		LEFT JOIN compras c ON f.id = c.fornecedor_id AND c.loja_id = ?
		WHERE f.loja_id = ?
		GROUP BY f.id, f.razao_social
		ORDER BY total_gasto DESC
	`

	rows, err := r.db.Query(query, lojaID, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar compras por fornecedor: %w", err)
	}
	defer rows.Close()

	var resultados []map[string]interface{}
	for rows.Next() {
		var id int
		var razaoSocial string
		var totalCompras int
		var totalGasto float64

		if err := rows.Scan(&id, &razaoSocial, &totalCompras, &totalGasto); err != nil {
			return nil, fmt.Errorf("erro ao scanear resultado: %w", err)
		}

		resultados = append(resultados, map[string]interface{}{
			"fornecedor_id":   id,
			"fornecedor":      razaoSocial,
			"total_compras":   totalCompras,
			"total_gasto":     totalGasto,
		})
	}

	return resultados, nil
}