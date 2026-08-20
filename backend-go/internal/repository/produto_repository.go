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
// CRUD BÁSICO
// ============================================

// CriarProduto cria um novo produto
func (r *ProdutoRepository) CriarProduto(produto *models.Produto) error {
	query := `
		INSERT INTO produtos (
			loja_id, codigo_barras, nome, descricao, categoria, sub_categoria,
			fabricante, fornecedor_id, registro_anvisa, classe_terapeutica,
			tarja, requere_receita, tipo_receita, preco_custo, preco_venda,
			margem_lucro, preco_minimo, estoque_minimo, estoque_maximo, ponto_pedido, ativo
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		produto.LojaID, produto.CodigoBarras, produto.Nome, produto.Descricao,
		produto.Categoria, produto.SubCategoria, produto.Fabricante, produto.FornecedorID,
		produto.RegistroANVISA, produto.ClasseTerapeutica, produto.Tarja,
		produto.RequereReceita, produto.TipoReceita, produto.PrecoCusto,
		produto.PrecoVenda, produto.MargemLucro, produto.PrecoMinimo,
		produto.EstoqueMinimo, produto.EstoqueMaximo, produto.PontoPedido,
		true,
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

// ListarProdutosEmFalta lista produtos com estoque abaixo do mínimo
func (r *ProdutoRepository) ListarProdutosEmFalta(lojaID int) ([]*models.Produto, error) {
	// Como o estoque é calculado via lotes, usamos uma subquery
	query := `
		SELECT p.id, p.loja_id, p.codigo_barras, p.nome, p.descricao, p.categoria, p.sub_categoria,
			p.fabricante, p.fornecedor_id, p.registro_anvisa, p.classe_terapeutica,
			p.tarja, p.requere_receita, p.tipo_receita, p.preco_custo, p.preco_venda,
			p.margem_lucro, p.preco_minimo, p.estoque_minimo, p.estoque_maximo, p.ponto_pedido,
			p.ativo, p.created_at, p.updated_at
		FROM produtos p
		WHERE p.loja_id = ? AND p.ativo = 1
		AND (
			SELECT COALESCE(SUM(l.quantidade), 0)
			FROM lotes l
			WHERE l.produto_id = p.id AND l.status = 'Ativo'
		) <= p.estoque_minimo
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar produtos em falta: %w", err)
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
			return nil, fmt.Errorf("erro ao scanear produto: %w", err)
		}

		if fornecedorID.Valid {
			p.FornecedorID = int(fornecedorID.Int64)
		}

		produtos = append(produtos, &p)
	}

	return produtos, nil
}

// ListarCategorias lista todas as categorias disponíveis
func (r *ProdutoRepository) ListarCategorias(lojaID int) ([]string, error) {
	query := `
		SELECT DISTINCT categoria
		FROM produtos
		WHERE loja_id = ? AND ativo = 1
		ORDER BY categoria ASC
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar categorias: %w", err)
	}
	defer rows.Close()

	var categorias []string
	for rows.Next() {
		var categoria string
		if err := rows.Scan(&categoria); err != nil {
			return nil, fmt.Errorf("erro ao scanear categoria: %w", err)
		}
		categorias = append(categorias, categoria)
	}

	return categorias, nil
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

// NomeExiste verifica se um nome de produto já existe
func (r *ProdutoRepository) NomeExiste(nome string, lojaID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM produtos WHERE nome = ? AND loja_id = ?)
	`, nome, lojaID).Scan(&exists)
	return exists, err
}

// NomeExisteExcluindoID verifica se um nome já existe (excluindo um ID)
func (r *ProdutoRepository) NomeExisteExcluindoID(nome string, lojaID int, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM produtos WHERE nome = ? AND loja_id = ? AND id != ?)
	`, nome, lojaID, id).Scan(&exists)
	return exists, err
}

// ProdutoExiste verifica se um produto existe
func (r *ProdutoRepository) ProdutoExiste(id int, lojaID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM produtos WHERE id = ? AND loja_id = ?)
	`, id, lojaID).Scan(&exists)
	return exists, err
}
