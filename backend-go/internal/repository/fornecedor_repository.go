package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type FornecedorRepository struct {
	db *sql.DB
}

func NewFornecedorRepository(db *sql.DB) *FornecedorRepository {
	return &FornecedorRepository{db: db}
}

func (r *FornecedorRepository) CriarFornecedor(fornecedor *models.Fornecedor) error {
	query := `
		INSERT INTO fornecedores (
			loja_id, razao_social, nome_fantasia, cnpj, ie, inscricao_municipal,
			endereco, numero, complemento, bairro, cidade, estado, cep,
			telefone, email, contato_nome, contato_telefone, ativo
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query,
		fornecedor.LojaID, fornecedor.RazaoSocial, fornecedor.NomeFantasia,
		fornecedor.CNPJ, fornecedor.IE, fornecedor.InscricaoMunicipal,
		fornecedor.Endereco, fornecedor.Numero, fornecedor.Complemento,
		fornecedor.Bairro, fornecedor.Cidade, fornecedor.Estado, fornecedor.CEP,
		fornecedor.Telefone, fornecedor.Email, fornecedor.ContatoNome,
		fornecedor.ContatoTelefone, true,
	)
	if err != nil {
		return fmt.Errorf("erro ao criar fornecedor: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("erro ao obter ID do fornecedor: %w", err)
	}

	fornecedor.ID = int(id)
	fornecedor.CreatedAt = time.Now()
	fornecedor.UpdatedAt = time.Now()
	return nil
}

func (r *FornecedorRepository) BuscarFornecedorPorID(id int, lojaID int) (*models.Fornecedor, error) {
	query := `
		SELECT id, loja_id, razao_social, nome_fantasia, cnpj, ie, inscricao_municipal,
			endereco, numero, complemento, bairro, cidade, estado, cep,
			telefone, email, contato_nome, contato_telefone, ativo, created_at, updated_at
		FROM fornecedores
		WHERE id = ? AND loja_id = ?
	`

	var f models.Fornecedor

	err := r.db.QueryRow(query, id, lojaID).Scan(
		&f.ID, &f.LojaID, &f.RazaoSocial, &f.NomeFantasia,
		&f.CNPJ, &f.IE, &f.InscricaoMunicipal,
		&f.Endereco, &f.Numero, &f.Complemento,
		&f.Bairro, &f.Cidade, &f.Estado, &f.CEP,
		&f.Telefone, &f.Email, &f.ContatoNome, &f.ContatoTelefone,
		&f.Ativo, &f.CreatedAt, &f.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar fornecedor: %w", err)
	}

	return &f, nil
}

func (r *FornecedorRepository) ListarFornecedores(lojaID int) ([]*models.Fornecedor, error) {
	query := `
		SELECT id, loja_id, razao_social, nome_fantasia, cnpj, telefone, email,
			contato_nome, contato_telefone, ativo, created_at
		FROM fornecedores
		WHERE loja_id = ?
		ORDER BY razao_social ASC
	`

	rows, err := r.db.Query(query, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar fornecedores: %w", err)
	}
	defer rows.Close()

	var fornecedores []*models.Fornecedor
	for rows.Next() {
		var f models.Fornecedor

		err := rows.Scan(
			&f.ID, &f.LojaID, &f.RazaoSocial, &f.NomeFantasia,
			&f.CNPJ, &f.Telefone, &f.Email,
			&f.ContatoNome, &f.ContatoTelefone,
			&f.Ativo, &f.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao scanear fornecedor: %w", err)
		}

		fornecedores = append(fornecedores, &f)
	}

	return fornecedores, nil
}

func (r *FornecedorRepository) AtualizarFornecedor(fornecedor *models.Fornecedor) error {
	query := `
		UPDATE fornecedores SET
			razao_social = ?, nome_fantasia = ?, cnpj = ?, ie = ?,
			inscricao_municipal = ?, endereco = ?, numero = ?, complemento = ?,
			bairro = ?, cidade = ?, estado = ?, cep = ?,
			telefone = ?, email = ?, contato_nome = ?, contato_telefone = ?,
			ativo = ?, updated_at = NOW()
		WHERE id = ? AND loja_id = ?
	`

	_, err := r.db.Exec(query,
		fornecedor.RazaoSocial, fornecedor.NomeFantasia, fornecedor.CNPJ,
		fornecedor.IE, fornecedor.InscricaoMunicipal,
		fornecedor.Endereco, fornecedor.Numero, fornecedor.Complemento,
		fornecedor.Bairro, fornecedor.Cidade, fornecedor.Estado, fornecedor.CEP,
		fornecedor.Telefone, fornecedor.Email, fornecedor.ContatoNome,
		fornecedor.ContatoTelefone, fornecedor.Ativo,
		fornecedor.ID, fornecedor.LojaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao atualizar fornecedor: %w", err)
	}

	return nil
}

func (r *FornecedorRepository) DeletarFornecedor(id int, lojaID int) error {
	_, err := r.db.Exec(`DELETE FROM fornecedores WHERE id = ? AND loja_id = ?`, id, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao deletar fornecedor: %w", err)
	}
	return nil
}