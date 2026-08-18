package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type VendaService struct {
	vendaRepo      *repository.VendaRepository
	produtoRepo    *repository.ProdutoRepository
	loteRepo       *repository.LoteRepository
	estoqueService *EstoqueService
	db             *sql.DB
}

func NewVendaService(
	vendaRepo *repository.VendaRepository,
	produtoRepo *repository.ProdutoRepository,
	loteRepo *repository.LoteRepository,
	estoqueService *EstoqueService,
	db *sql.DB,
) *VendaService {
	return &VendaService{
		vendaRepo:      vendaRepo,
		produtoRepo:    produtoRepo,
		loteRepo:       loteRepo,
		estoqueService: estoqueService,
		db:             db,
	}
}

// ProcessarVenda processa uma venda completa
func (s *VendaService) ProcessarVenda(ctx context.Context, venda *models.Venda, itens []models.ItemVenda, usuarioID int) (*models.Venda, error) {
	// 1. Validar itens
	if len(itens) == 0 {
		return nil, fmt.Errorf("venda deve ter pelo menos um item")
	}

	// 2. Validar estoque para cada item
	for _, item := range itens {
		// Buscar produto
		produto, err := s.produtoRepo.BuscarProdutoPorID(item.ProdutoID, venda.LojaID)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar produto: %w", err)
		}
		if produto == nil {
			return nil, fmt.Errorf("produto %d não encontrado", item.ProdutoID)
		}

		// Verificar se produto está ativo
		if !produto.Ativo {
			return nil, fmt.Errorf("produto %s está inativo", produto.Nome)
		}

		// Buscar lote
		lote, err := s.loteRepo.BuscarLotePorID(item.LoteID)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar lote: %w", err)
		}
		if lote == nil {
			return nil, fmt.Errorf("lote não encontrado para produto %s", produto.Nome)
		}

		// Verificar validade do lote
		if lote.DataValidade.Before(time.Now()) {
			return nil, fmt.Errorf("lote do produto %s está vencido", produto.Nome)
		}

		// Verificar quantidade em estoque
		if lote.Quantidade < item.Quantidade {
			return nil, fmt.Errorf("estoque insuficiente para %s (disponível: %d)", produto.Nome, lote.Quantidade)
		}

		// Para medicamentos controlados, verificar receita
		if produto.RequereReceita && !venda.ReceitaAnexada {
			return nil, fmt.Errorf("produto %s requer receita médica", produto.Nome)
		}
	}

	// 3. Calcular subtotal e total
	var subtotal float64
	for _, item := range itens {
		// Buscar preço do produto
		produto, err := s.produtoRepo.BuscarProdutoPorID(item.ProdutoID, venda.LojaID)
		if err != nil {
			return nil, err
		}
		item.PrecoUnit = produto.PrecoVenda
		item.Total = item.PrecoUnit * float64(item.Quantidade)
		subtotal += item.Total
	}

	venda.Subtotal = subtotal
	venda.Total = subtotal - venda.Desconto
	venda.AtendenteID = usuarioID

	// 4. Criar venda no banco
	if err := s.vendaRepo.CriarVenda(venda); err != nil {
		return nil, fmt.Errorf("erro ao criar venda: %w", err)
	}

	// 5. Dar baixa no estoque para cada item
	for _, item := range itens {
		if err := s.estoqueService.SaidaEstoque(
			ctx,
			item.ProdutoID,
			venda.LojaID,
			item.Quantidade,
			item.LoteID,
			usuarioID,
			"Venda",
			venda.NumeroVenda,
		); err != nil {
			return nil, fmt.Errorf("erro ao dar baixa no estoque: %w", err)
		}
	}

	// 6. Atualizar status da venda para Pago
	if err := s.vendaRepo.AtualizarStatusVenda(venda.ID, "Pago"); err != nil {
		return nil, fmt.Errorf("erro ao atualizar status da venda: %w", err)
	}

	// 7. Atualizar cliente (se informado)
	if venda.ClienteID != nil {
		if err := s.atualizarCliente(*venda.ClienteID, venda.LojaID, venda.Total); err != nil {
			// Não falha a venda, apenas loga
			_ = err
		}
	}

	return venda, nil
}

// atualizarCliente atualiza dados do cliente após compra
func (s *VendaService) atualizarCliente(clienteID, lojaID int, total float64) error {
	_, err := s.db.Exec(`
		UPDATE clientes SET 
			ultima_compra = NOW(),
			total_compras = total_compras + ?,
			quantidade_compras = quantidade_compras + 1
		WHERE id = ? AND loja_id = ?
	`, total, clienteID, lojaID)
	return err
}

// CancelarVenda cancela uma venda e reverte o estoque
func (s *VendaService) CancelarVenda(ctx context.Context, vendaID int, usuarioID int) error {
	// 1. Buscar venda
	venda, err := s.vendaRepo.BuscarVendaPorID(vendaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar venda: %w", err)
	}
	if venda == nil {
		return fmt.Errorf("venda não encontrada")
	}

	if venda.Status == "Cancelado" {
		return fmt.Errorf("venda já está cancelada")
	}

	// 2. Reverter estoque para cada item
	for _, item := range venda.Itens {
		if err := s.estoqueService.EntradaEstoque(
			ctx,
			item.ProdutoID,
			venda.LojaID,
			&models.Lote{
				NumeroLote:   fmt.Sprintf("CANCEL-%s", venda.NumeroVenda),
				DataValidade: time.Now().AddDate(1, 0, 0),
				Quantidade:   item.Quantidade,
				PrecoCusto:   0,
				PrecoVenda:   item.PrecoUnit,
			},
			usuarioID,
		); err != nil {
			return fmt.Errorf("erro ao reverter estoque: %w", err)
		}
	}

	// 3. Atualizar status da venda
	if err := s.vendaRepo.AtualizarStatusVenda(vendaID, "Cancelado"); err != nil {
		return fmt.Errorf("erro ao cancelar venda: %w", err)
	}

	return nil
}

// RelatorioVendasDiarias gera relatório de vendas diárias
func (s *VendaService) RelatorioVendasDiarias(ctx context.Context, lojaID int, data string) (map[string]interface{}, error) {
	if data == "" {
		data = time.Now().Format("2006-01-02")
	}

	var totalVendas, totalItens int
	var valorTotal float64

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_vendas,
			COALESCE(SUM(total), 0) as valor_total,
			COALESCE(SUM(quantidade), 0) as total_itens
		FROM vendas v
		JOIN itens_venda iv ON v.id = iv.venda_id
		WHERE v.loja_id = ? AND DATE(v.created_at) = ?
		AND v.status = 'Pago'
	`, lojaID, data).Scan(&totalVendas, &valorTotal, &totalItens)

	if err != nil {
		return nil, err
	}

	// Produtos mais vendidos no dia
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			p.nome,
			SUM(iv.quantidade) as quantidade,
			SUM(iv.total) as total
		FROM itens_venda iv
		JOIN vendas v ON iv.venda_id = v.id
		JOIN produtos p ON iv.produto_id = p.id
		WHERE v.loja_id = ? AND DATE(v.created_at) = ?
		AND v.status = 'Pago'
		GROUP BY p.id, p.nome
		ORDER BY quantidade DESC
		LIMIT 5
	`, lojaID, data)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var produtosMaisVendidos []map[string]interface{}
	for rows.Next() {
		var nome string
		var quantidade int
		var total float64

		if err := rows.Scan(&nome, &quantidade, &total); err != nil {
			return nil, err
		}

		produtosMaisVendidos = append(produtosMaisVendidos, map[string]interface{}{
			"nome":       nome,
			"quantidade": quantidade,
			"total":      total,
		})
	}

	return map[string]interface{}{
		"data":                   data,
		"total_vendas":           totalVendas,
		"valor_total":            valorTotal,
		"total_itens":            totalItens,
		"ticket_medio":           valorTotal / float64(totalVendas+1),
		"produtos_mais_vendidos": produtosMaisVendidos,
	}, nil
}
