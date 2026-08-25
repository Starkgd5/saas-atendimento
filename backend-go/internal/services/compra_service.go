package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type CompraService struct {
	compraRepo     *repository.CompraRepository
	fornecedorRepo *repository.FornecedorRepository
	produtoRepo    *repository.ProdutoRepository
	loteRepo       *repository.LoteRepository
	estoqueService *EstoqueService
	db             *sql.DB
}

func NewCompraService(
	compraRepo *repository.CompraRepository,
	fornecedorRepo *repository.FornecedorRepository,
	produtoRepo *repository.ProdutoRepository,
	loteRepo *repository.LoteRepository,
	estoqueService *EstoqueService,
	db *sql.DB,
) *CompraService {
	return &CompraService{
		compraRepo:     compraRepo,
		fornecedorRepo: fornecedorRepo,
		produtoRepo:    produtoRepo,
		loteRepo:       loteRepo,
		estoqueService: estoqueService,
		db:             db,
	}
}

// ============================================
// CRIAÇÃO E GESTÃO
// ============================================

// CriarCompra cria uma nova compra
func (s *CompraService) CriarCompra(ctx context.Context, compra *models.Compra) error {
	// Validar fornecedor
	if compra.FornecedorID == 0 {
		return fmt.Errorf("fornecedor é obrigatório")
	}

	fornecedor, err := s.fornecedorRepo.BuscarFornecedorPorID(compra.FornecedorID, compra.LojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar fornecedor: %w", err)
	}
	if fornecedor == nil {
		return fmt.Errorf("fornecedor não encontrado")
	}

	// Validar itens
	if len(compra.Itens) == 0 {
		return fmt.Errorf("compra deve ter pelo menos um item")
	}

	// Calcular subtotal e validar produtos
	var subtotal float64
	for i, item := range compra.Itens {
		produto, err := s.produtoRepo.BuscarProdutoPorID(item.ProdutoID, compra.LojaID)
		if err != nil {
			return fmt.Errorf("erro ao buscar produto %d: %w", item.ProdutoID, err)
		}
		if produto == nil {
			return fmt.Errorf("produto %d não encontrado", item.ProdutoID)
		}
		if !produto.Ativo {
			return fmt.Errorf("produto %s está inativo", produto.Nome)
		}

		// Definir preço unitário como preço de custo
		compra.Itens[i].PrecoUnit = produto.PrecoCusto
		compra.Itens[i].Total = produto.PrecoCusto * float64(item.Quantidade)
		subtotal += compra.Itens[i].Total
	}

	compra.Subtotal = subtotal
	compra.Total = subtotal - compra.Desconto
	compra.Status = "Em Andamento"

	return s.compraRepo.CriarCompra(compra)
}

// ReceberCompra processa o recebimento de uma compra
func (s *CompraService) ReceberCompra(ctx context.Context, compraID int, lojaID int, usuarioID int) (*models.Compra, error) {
	// Buscar compra
	compra, err := s.compraRepo.BuscarCompraPorID(compraID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar compra: %w", err)
	}
	if compra == nil {
		return nil, fmt.Errorf("compra não encontrada")
	}

	if compra.LojaID != lojaID {
		return nil, fmt.Errorf("compra não pertence a esta loja")
	}

	if compra.Status == "Recebido" {
		return nil, fmt.Errorf("compra já foi recebida")
	}

	if compra.Status == "Cancelado" {
		return nil, fmt.Errorf("compra foi cancelada e não pode ser recebida")
	}

	// Processar entrada de estoque para cada item
	var itensProcessados []map[string]interface{}
	var erros []string

	for _, item := range compra.Itens {
		// Buscar produto
		produto, err := s.produtoRepo.BuscarProdutoPorID(item.ProdutoID, lojaID)
		if err != nil {
			erros = append(erros, fmt.Sprintf("Erro ao buscar produto %d: %v", item.ProdutoID, err))
			continue
		}
		if produto == nil {
			erros = append(erros, fmt.Sprintf("Produto %d não encontrado", item.ProdutoID))
			continue
		}

		// Gerar número do lote
		numeroLote := fmt.Sprintf("CMP-%d-%s", compraID, time.Now().Format("20060102"))

		// Criar lote
		lote := &models.Lote{
			ProdutoID:        item.ProdutoID,
			LojaID:           lojaID,
			NumeroLote:       numeroLote,
			DataFabricacao:   compra.DataEmissao,
			DataValidade:     compra.DataEmissao.AddDate(1, 0, 0), // 1 ano depois
			Quantidade:       item.Quantidade,
			QuantidadeInicial: item.Quantidade,
			PrecoCusto:       item.PrecoUnit,
			PrecoVenda:       produto.PrecoVenda,
			Status:           "Ativo",
		}

		// Entrada no estoque
		if err := s.estoqueService.EntradaEstoque(ctx, item.ProdutoID, lojaID, lote, usuarioID); err != nil {
			erros = append(erros, fmt.Sprintf("Erro ao dar entrada no estoque do produto %s: %v", produto.Nome, err))
			continue
		}

		itensProcessados = append(itensProcessados, map[string]interface{}{
			"produto_id":    item.ProdutoID,
			"produto_nome":  produto.Nome,
			"quantidade":    item.Quantidade,
			"numero_lote":   numeroLote,
			"preco_custo":   item.PrecoUnit,
			"preco_venda":   produto.PrecoVenda,
		})
	}

	// Se houver erros e nenhum item processado, retornar erro
	if len(erros) > 0 && len(itensProcessados) == 0 {
		return nil, fmt.Errorf("falha ao processar recebimento: %v", erros)
	}

	// Atualizar status da compra
	if err := s.compraRepo.ReceberCompra(compraID); err != nil {
		return nil, fmt.Errorf("erro ao atualizar status da compra: %w", err)
	}

	// Buscar compra atualizada
	compraAtualizada, err := s.compraRepo.BuscarCompraPorID(compraID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar compra atualizada: %w", err)
	}

	return compraAtualizada, nil
}

// CancelarCompra cancela uma compra
func (s *CompraService) CancelarCompra(ctx context.Context, compraID int) error {
	compra, err := s.compraRepo.BuscarCompraPorID(compraID)
	if err != nil {
		return fmt.Errorf("erro ao buscar compra: %w", err)
	}
	if compra == nil {
		return fmt.Errorf("compra não encontrada")
	}

	if compra.Status == "Recebido" {
		return fmt.Errorf("compra já foi recebida e não pode ser cancelada")
	}

	if compra.Status == "Cancelado" {
		return fmt.Errorf("compra já está cancelada")
	}

	return s.compraRepo.CancelarCompra(compraID)
}

// ============================================
// CONSULTAS
// ============================================

// BuscarCompra busca uma compra por ID
func (s *CompraService) BuscarCompra(ctx context.Context, compraID int) (*models.Compra, error) {
	return s.compraRepo.BuscarCompraPorID(compraID)
}

// ListarCompras lista todas as compras de uma loja
func (s *CompraService) ListarCompras(ctx context.Context, lojaID int) ([]*models.Compra, error) {
	return s.compraRepo.ListarCompras(lojaID)
}

// ListarComprasPorStatus lista compras por status
func (s *CompraService) ListarComprasPorStatus(ctx context.Context, lojaID int, status string) ([]*models.Compra, error) {
	return s.compraRepo.ListarComprasPorStatus(lojaID, status)
}

// ============================================
// ESTATÍSTICAS
// ============================================

// GetTotalCompras retorna o total de compras
func (s *CompraService) GetTotalCompras(ctx context.Context, lojaID int) (int, error) {
	return s.compraRepo.GetTotalCompras(lojaID)
}

// GetTotalComprasMes retorna o total de compras do mês
func (s *CompraService) GetTotalComprasMes(ctx context.Context, lojaID int) (float64, error) {
	return s.compraRepo.GetTotalComprasMes(lojaID)
}

// GetComprasPorFornecedor retorna compras agrupadas por fornecedor
func (s *CompraService) GetComprasPorFornecedor(ctx context.Context, lojaID int) ([]map[string]interface{}, error) {
	return s.compraRepo.GetComprasPorFornecedor(lojaID)
}

// ============================================
// VALIDAÇÕES
// ============================================

// ValidarCompra verifica se uma compra pode ser processada
func (s *CompraService) ValidarCompra(compra *models.Compra) error {
	// Verificar fornecedor
	if compra.FornecedorID == 0 {
		return fmt.Errorf("fornecedor é obrigatório")
	}

	// Verificar itens
	if len(compra.Itens) == 0 {
		return fmt.Errorf("compra deve ter pelo menos um item")
	}

	// Verificar quantidades
	for _, item := range compra.Itens {
		if item.Quantidade <= 0 {
			return fmt.Errorf("quantidade deve ser maior que zero")
		}
	}

	// Verificar desconto
	if compra.Desconto < 0 {
		return fmt.Errorf("desconto não pode ser negativo")
	}

	if compra.Desconto > compra.Subtotal {
		return fmt.Errorf("desconto não pode ser maior que o subtotal")
	}

	return nil
}