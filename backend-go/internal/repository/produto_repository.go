package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type ProdutoRepository struct {
	db *sql.DB
}

func NewProdutoRepository(db *sql.DB) *ProdutoRepository {
	return &ProdutoRepository{db: db}
}

// ============================================
// PRODUTOS
// ============================================

// CriarProduto cria um novo produto
func (r *ProdutoRepository) CriarProduto(produto *models.Produto) error {
	query := `
		INSERT INTO produtos (
			loja_id, codigo_barras, nome, descricao, categoria, sub_categoria,
			fabricante, fornecedor_id, registro_anvisa, classe_terapeutica,
			tarja, requere_receita, tipo_receita, preco_custo, preco_venda,
			margem_lucro, preco_minimo, estoque_minimo, estoque_maximo, ponto_pedido
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		produto.LojaID, produto.CodigoBarras, produto.Nome, produto.Descricao,
		produto.Categoria, produto.SubCategoria, produto.Fabricante, produto.FornecedorID,
		produto.RegistroANVISA, produto.ClasseTerapeutica, produto.Tarja,
		produto.RequereReceita, produto.TipoReceita, produto.PrecoCusto,
		produto.PrecoVenda, produto.MargemLucro, produto.PrecoMinimo,
		produto.EstoqueMinimo, produto.EstoqueMaximo, produto.PontoPedido,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar produto: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do produto: %w", err)
	}

	produto.ID = int(id)
	produto.CreatedAt = time.Now()
	produto.UpdatedAt = time.Now()
	produto.Ativo = true

	return nil
}

// BuscarProdutoPorID busca um produto pelo ID
func (r *ProdutoRepository) BuscarProdutoPorID(id int, lojaID int) (*models.Produto, error) {
	query := `
		SELECT id, loja_id, codigo_barras, nome, descricao, categoria, sub_categoria,
			fabricante, fornecedor_id, registro_anvisa, classe_terapeutica,
			tarja, requere_receita, tipo_receita, preco_custo, preco_venda,
			margem_lucro, preco_minimo, estoque_minimo, estoque_maximo, ponto_pedido,
			ativo, created_at, updated_at
		FROM produtos
		WHERE id = ? AND loja_id = ?
	`

	var produto models.Produto
	var fornecedorID sql.NullInt64

	err := r.db.QueryRow(query, id, lojaID).Scan(
		&produto.ID, &produto.LojaID, &produto.CodigoBarras, &produto.Nome,
		&produto.Descricao, &produto.Categoria, &produto.SubCategoria,
		&produto.Fabricante, &fornecedorID,
		&produto.RegistroANVISA, &produto.ClasseTerapeutica,
		&produto.Tarja, &produto.RequereReceita, &produto.TipoReceita,
		&produto.PrecoCusto, &produto.PrecoVenda,
		&produto.MargemLucro, &produto.PrecoMinimo,
		&produto.EstoqueMinimo, &produto.EstoqueMaximo, &produto.PontoPedido,
		&produto.Ativo, &produto.CreatedAt, &produto.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produto: %w", err)
	}

	if fornecedorID.Valid {
		produto.FornecedorID = int(fornecedorID.Int64)
	}

	return &produto, nil
}

// BuscarProdutoPorCodigoBarras busca um produto pelo código de barras
func (r *ProdutoRepository) BuscarProdutoPorCodigoBarras(codigo string, lojaID int) (*models.Produto, error) {
	query := `
		SELECT id, loja_id, codigo_barras, nome, descricao, categoria, sub_categoria,
			fabricante, fornecedor_id, registro_anvisa, classe_terapeutica,
			tarja, requere_receita, tipo_receita, preco_custo, preco_venda,
			margem_lucro, preco_minimo, estoque_minimo, estoque_maximo, ponto_pedido,
			ativo, created_at, updated_at
		FROM produtos
		WHERE codigo_barras = ? AND loja_id = ?
	`

	var produto models.Produto
	var fornecedorID sql.NullInt64

	err := r.db.QueryRow(query, codigo, lojaID).Scan(
		&produto.ID, &produto.LojaID, &produto.CodigoBarras, &produto.Nome,
		&produto.Descricao, &produto.Categoria, &produto.SubCategoria,
		&produto.Fabricante, &fornecedorID,
		&produto.RegistroANVISA, &produto.ClasseTerapeutica,
		&produto.Tarja, &produto.RequereReceita, &produto.TipoReceita,
		&produto.PrecoCusto, &produto.PrecoVenda,
		&produto.MargemLucro, &produto.PrecoMinimo,
		&produto.EstoqueMinimo, &produto.EstoqueMaximo, &produto.PontoPedido,
		&produto.Ativo, &produto.CreatedAt, &produto.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produto por código: %w", err)
	}

	if fornecedorID.Valid {
		produto.FornecedorID = int(fornecedorID.Int64)
	}

	return &produto, nil
}

// ListarProdutos lista produtos com filtros
func (r *ProdutoRepository) ListarProdutos(lojaID int, categoria string, ativo *bool, limit, offset int) ([]*models.Produto, int, error) {
	if limit == 0 {
		limit = 50
	}

	args := []interface{}{lojaID}
	whereClause := "WHERE loja_id = ?"

	if categoria != "" {
		whereClause += " AND categoria = ?"
		args = append(args, categoria)
	}

	if ativo != nil {
		whereClause += " AND ativo = ?"
		args = append(args, *ativo)
	}

	// Contar total
	countQuery := `SELECT COUNT(*) FROM produtos ` + whereClause
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao contar produtos: %w", err)
	}

	// Buscar produtos
	query := `
		SELECT id, loja_id, codigo_barras, nome, descricao, categoria, sub_categoria,
			fabricante, fornecedor_id, registro_anvisa, classe_terapeutica,
			tarja, requere_receita, tipo_receita, preco_custo, preco_venda,
			margem_lucro, preco_minimo, estoque_minimo, estoque_maximo, ponto_pedido,
			ativo, created_at, updated_at
		FROM produtos ` + whereClause + `
		ORDER BY nome ASC
		LIMIT ? OFFSET ?
	`
	args = append(args, limit, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar produtos: %w", err)
	}
	defer rows.Close()

	var produtos []*models.Produto
	for rows.Next() {
		var p models.Produto
		var fornecedorID sql.NullInt64

		err := rows.Scan(
			&p.ID, &p.LojaID, &p.CodigoBarras, &p.Nome,
			&p.Descricao, &p.Categoria, &p.SubCategoria,
			&p.Fabricante, &fornecedorID,
			&p.RegistroANVISA, &p.ClasseTerapeutica,
			&p.Tarja, &p.RequereReceita, &p.TipoReceita,
			&p.PrecoCusto, &p.PrecoVenda,
			&p.MargemLucro, &p.PrecoMinimo,
			&p.EstoqueMinimo, &p.EstoqueMaximo, &p.PontoPedido,
			&p.Ativo, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear produto: %w", err)
		}

		if fornecedorID.Valid {
			p.FornecedorID = int(fornecedorID.Int64)
		}

		produtos = append(produtos, &p)
	}

	return produtos, total, nil
}

// AtualizarProduto atualiza um produto
func (r *ProdutoRepository) AtualizarProduto(produto *models.Produto) error {
	query := `
		UPDATE produtos SET
			codigo_barras = ?, nome = ?, descricao = ?, categoria = ?,
			sub_categoria = ?, fabricante = ?, fornecedor_id = ?,
			registro_anvisa = ?, classe_terapeutica = ?,
			tarja = ?, requere_receita = ?, tipo_receita = ?,
			preco_custo = ?, preco_venda = ?, margem_lucro = ?,
			preco_minimo = ?, estoque_minimo = ?, estoque_maximo = ?,
			ponto_pedido = ?, ativo = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	_, err := r.db.Exec(query,
		produto.CodigoBarras, produto.Nome, produto.Descricao, produto.Categoria,
		produto.SubCategoria, produto.Fabricante, produto.FornecedorID,
		produto.RegistroANVISA, produto.ClasseTerapeutica,
		produto.Tarja, produto.RequereReceita, produto.TipoReceita,
		produto.PrecoCusto, produto.PrecoVenda, produto.MargemLucro,
		produto.PrecoMinimo, produto.EstoqueMinimo, produto.EstoqueMaximo,
		produto.PontoPedido, produto.Ativo,
		produto.ID, produto.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar produto: %w", err)
	}

	return nil
}
