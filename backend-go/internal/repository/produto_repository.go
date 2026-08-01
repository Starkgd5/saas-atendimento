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

// BuscarProdutoPorID busca um produto pelo ID
func (r *ProdutoRepository) BuscarProdutoPorID(id int, lojaID int) (*models.Produto, error) {
	query := `
		SELECT id, nome, descricao, categoria, preco, preco_custo, estoque,
		       estoque_min, requere_receita, ativo, created_at, updated_at
		FROM produtos
		WHERE id = ? AND loja_id = ?
	`

	var produto models.Produto

	err := r.db.QueryRow(query, id, lojaID).Scan(
		&produto.ID,
		&produto.Nome,
		&produto.Descricao,
		&produto.Categoria,
		&produto.Preco,
		&produto.PrecoCusto,
		&produto.Estoque,
		&produto.EstoqueMin,
		&produto.RequereReceita,
		&produto.Ativo,
		&produto.CreatedAt,
		&produto.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produto: %w", err)
	}

	produto.LojaID = lojaID
	return &produto, nil
}

// ListarProdutos lista produtos com paginação
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
		SELECT id, nome, descricao, categoria, preco, preco_custo, estoque,
		       estoque_min, requere_receita, ativo, created_at, updated_at
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

		err := rows.Scan(
			&p.ID,
			&p.Nome,
			&p.Descricao,
			&p.Categoria,
			&p.Preco,
			&p.PrecoCusto,
			&p.Estoque,
			&p.EstoqueMin,
			&p.RequereReceita,
			&p.Ativo,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao scanear produto: %w", err)
		}

		p.LojaID = lojaID
		produtos = append(produtos, &p)
	}

	return produtos, total, nil
}

// ListarProdutosEmFalta lista produtos com estoque abaixo do mínimo
func (r *ProdutoRepository) ListarProdutosEmFalta(lojaID int) ([]*models.Produto, error) {
	query := `
		SELECT id, nome, descricao, categoria, preco, preco_custo, estoque,
		       estoque_min, requere_receita, ativo, created_at, updated_at
		FROM produtos
		WHERE loja_id = ? AND estoque <= estoque_min AND ativo = 1
		ORDER BY estoque ASC
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar produtos em falta: %w", err)
	}
	defer rows.Close()

	var produtos []*models.Produto
	for rows.Next() {
		var p models.Produto

		err := rows.Scan(
			&p.ID,
			&p.Nome,
			&p.Descricao,
			&p.Categoria,
			&p.Preco,
			&p.PrecoCusto,
			&p.Estoque,
			&p.EstoqueMin,
			&p.RequereReceita,
			&p.Ativo,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear produto: %w", err)
		}

		p.LojaID = lojaID
		produtos = append(produtos, &p)
	}

	return produtos, nil
}

// ListarCategorias lista todas as categorias de produtos
func (r *ProdutoRepository) ListarCategorias(lojaID int) ([]string, error) {
	query := `
		SELECT DISTINCT categoria
		FROM produtos
		WHERE loja_id = ?
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

// ============================================
// CRIAÇÃO E ATUALIZAÇÃO
// ============================================

// CriarProduto cria um novo produto
func (r *ProdutoRepository) CriarProduto(produto *models.Produto) error {
	query := `
		INSERT INTO produtos (loja_id, nome, descricao, categoria, preco, preco_custo,
		                      estoque, estoque_min, requere_receita, ativo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		produto.LojaID,
		produto.Nome,
		produto.Descricao,
		produto.Categoria,
		produto.Preco,
		produto.PrecoCusto,
		produto.Estoque,
		produto.EstoqueMin,
		produto.RequereReceita,
		produto.Ativo,
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
	return nil
}

// AtualizarProduto atualiza um produto existente
func (r *ProdutoRepository) AtualizarProduto(produto *models.Produto) error {
	query := `
		UPDATE produtos 
		SET nome = ?, descricao = ?, categoria = ?, preco = ?, preco_custo = ?,
		    estoque = ?, estoque_min = ?, requere_receita = ?, ativo = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	result, err := r.db.Exec(query,
		produto.Nome,
		produto.Descricao,
		produto.Categoria,
		produto.Preco,
		produto.PrecoCusto,
		produto.Estoque,
		produto.EstoqueMin,
		produto.RequereReceita,
		produto.Ativo,
		produto.ID,
		produto.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar produto: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("produto não encontrado")
	}

	return nil
}

// AtualizarEstoque atualiza o estoque de um produto
func (r *ProdutoRepository) AtualizarEstoque(id int, lojaID int, quantidade int) error {
	query := `
		UPDATE produtos 
		SET estoque = estoque + ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`
	result, err := r.db.Exec(query, quantidade, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar estoque: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("produto não encontrado")
	}

	return nil
}

// ============================================
// EXCLUSÃO
// ============================================

// DeletarProduto deleta um produto
func (r *ProdutoRepository) DeletarProduto(id int, lojaID int) error {
	result, err := r.db.Exec(`DELETE FROM produtos WHERE id = ? AND loja_id = ?`, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao deletar produto: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao obter linhas afetadas: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("produto não encontrado")
	}

	return nil
}

// ============================================
// VALIDAÇÕES
// ============================================

// ProdutoExiste verifica se um produto existe
func (r *ProdutoRepository) ProdutoExiste(id int, lojaID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM produtos WHERE id = ? AND loja_id = ?)`, id, lojaID).Scan(&exists)
	return exists, err
}

// NomeExiste verifica se um nome de produto já existe
func (r *ProdutoRepository) NomeExiste(nome string, lojaID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM produtos WHERE nome = ? AND loja_id = ?)`, nome, lojaID).Scan(&exists)
	return exists, err
}

// NomeExisteExcluindoID verifica se um nome já existe (excluindo um ID)
func (r *ProdutoRepository) NomeExisteExcluindoID(nome string, lojaID int, id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM produtos WHERE nome = ? AND loja_id = ? AND id != ?)`, nome, lojaID, id).Scan(&exists)
	return exists, err
}