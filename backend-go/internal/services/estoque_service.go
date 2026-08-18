package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
)

type EstoqueService struct {
	produtoRepo   *repository.ProdutoRepository
	loteRepo      *repository.LoteRepository
	movimentoRepo *repository.MovimentoEstoqueRepository
	db            *sql.DB
}

func NewEstoqueService(
	produtoRepo *repository.ProdutoRepository,
	loteRepo *repository.LoteRepository,
	movimentoRepo *repository.MovimentoEstoqueRepository,
	db *sql.DB,
) *EstoqueService {
	return &EstoqueService{
		produtoRepo:   produtoRepo,
		loteRepo:      loteRepo,
		movimentoRepo: movimentoRepo,
		db:            db,
	}
}

// EntradaEstoque realiza uma entrada de estoque (compra)
func (s *EstoqueService) EntradaEstoque(ctx context.Context, produtoID, lojaID int, lote *models.Lote, usuarioID int) error {
	// 1. Verificar se o produto existe
	produto, err := s.produtoRepo.BuscarProdutoPorID(produtoID, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar produto: %w", err)
	}
	if produto == nil {
		return fmt.Errorf("produto não encontrado")
	}

	// 2. Buscar quantidade atual do estoque (soma dos lotes)
	lotex, err := s.loteRepo.BuscarLotesPorProduto(produtoID, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar lotes: %w", err)
	}

	estoqueAtual := 0
	for _, l := range lotex {
		if l.Status == "Ativo" {
			estoqueAtual += l.Quantidade
		}
	}

	// 3. Iniciar transação
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback()

	// 4. Criar o lote
	lote.ProdutoID = produtoID
	lote.LojaID = lojaID
	lote.QuantidadeInicial = lote.Quantidade
	lote.Status = "Ativo"

	if err := s.loteRepo.CriarLote(lote); err != nil {
		return fmt.Errorf("erro ao criar lote: %w", err)
	}

	// 5. Registrar movimento de entrada
	movimento := &models.MovimentoEstoque{
		LojaID:        lojaID,
		ProdutoID:     produtoID,
		LoteID:        lote.ID,
		Tipo:          "Entrada",
		Quantidade:    lote.Quantidade,
		SaldoAnterior: estoqueAtual,
		SaldoAtual:    estoqueAtual + lote.Quantidade,
		Motivo:        "Compra",
		Documento:     lote.NumeroLote,
		UsuarioID:     usuarioID,
	}

	if err := s.movimentoRepo.CriarMovimento(movimento); err != nil {
		return fmt.Errorf("erro ao criar movimento: %w", err)
	}

	// 6. Commit da transação
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao commit transação: %w", err)
	}

	return nil
}

// SaidaEstoque realiza uma saída de estoque (venda)
func (s *EstoqueService) SaidaEstoque(ctx context.Context, produtoID, lojaID int, quantidade int, loteID int, usuarioID int, motivo string, documento string) error {
	// 1. Buscar o lote
	lote, err := s.loteRepo.BuscarLotePorID(loteID)
	if err != nil {
		return fmt.Errorf("erro ao buscar lote: %w", err)
	}
	if lote == nil {
		return fmt.Errorf("lote não encontrado")
	}

	if lote.Quantidade < quantidade {
		return fmt.Errorf("estoque insuficiente no lote")
	}

	// 2. Buscar quantidade atual do estoque (soma dos lotes)
	lotex, err := s.loteRepo.BuscarLotesPorProduto(produtoID, lojaID)
	if err != nil {
		return fmt.Errorf("erro ao buscar lotes: %w", err)
	}

	estoqueAtual := 0
	for _, l := range lotex {
		if l.Status == "Ativo" {
			estoqueAtual += l.Quantidade
		}
	}

	// 3. Atualizar quantidade do lote
	novaQuantidade := lote.Quantidade - quantidade
	if err := s.loteRepo.AtualizarQuantidadeLote(loteID, novaQuantidade); err != nil {
		return fmt.Errorf("erro ao atualizar lote: %w", err)
	}

	// 4. Registrar movimento de saída
	movimento := &models.MovimentoEstoque{
		LojaID:        lojaID,
		ProdutoID:     produtoID,
		LoteID:        lote.ID,
		Tipo:          "Saída",
		Quantidade:    quantidade,
		SaldoAnterior: estoqueAtual,
		SaldoAtual:    estoqueAtual - quantidade,
		Motivo:        motivo,
		Documento:     documento,
		UsuarioID:     usuarioID,
	}

	if err := s.movimentoRepo.CriarMovimento(movimento); err != nil {
		return fmt.Errorf("erro ao criar movimento: %w", err)
	}

	return nil
}

// CalcularEstoqueTotal calcula o estoque total de um produto (soma dos lotes ativos)
func (s *EstoqueService) CalcularEstoqueTotal(produtoID int, lojaID int) (int, error) {
	lotes, err := s.loteRepo.BuscarLotesPorProduto(produtoID, lojaID)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, l := range lotes {
		if l.Status == "Ativo" {
			total += l.Quantidade
		}
	}

	return total, nil
}

// VerificarValidades verifica lotes vencidos e alerta
func (s *EstoqueService) VerificarValidades(ctx context.Context) ([]*models.Lote, error) {
	// Atualizar status dos lotes vencidos
	if err := s.loteRepo.VerificarLotesVencidos(); err != nil {
		return nil, err
	}

	// Buscar lotes vencidos
	// (usando a mesma query de lotes válidos mas com status Vencido)
	// Para simplificar, vamos buscar todos os lotes e filtrar
	return nil, nil
}

// ObterAlertasEstoque retorna alertas de estoque (baixo e vencido)
func (s *EstoqueService) ObterAlertasEstoque(ctx context.Context, lojaID int) (map[string]interface{}, error) {
	// 1. Produtos com estoque baixo
	produtosBaixo, _, err := s.produtoRepo.ListarProdutos(lojaID, "", nil, 100, 0)
	if err != nil {
		return nil, err
	}

	baixoEstoque := []map[string]interface{}{}
	for _, p := range produtosBaixo {
		// Verificar se produto está ativo
		if !p.Ativo {
			continue
		}

		// Calcular estoque total
		total, err := s.CalcularEstoqueTotal(p.ID, lojaID)
		if err != nil {
			continue
		}

		if total <= p.EstoqueMinimo {
			baixoEstoque = append(baixoEstoque, map[string]interface{}{
				"produto_id":   p.ID,
				"nome":         p.Nome,
				"estoque":      total,
				"estoque_min":  p.EstoqueMinimo,
				"ponto_pedido": p.PontoPedido,
			})
		}
	}

	// 2. Produtos com lotes vencendo (próximos 30 dias)
	lotesVencendo, err := s.loteRepo.BuscarLotesValidos(0, lojaID)
	if err != nil {
		return nil, err
	}

	vencendo := []map[string]interface{}{}
	for _, l := range lotesVencendo {
		dias := int(time.Until(l.DataValidade).Hours() / 24)
		if dias <= 30 {
			produto, err := s.produtoRepo.BuscarProdutoPorID(l.ProdutoID, lojaID)
			if err != nil || produto == nil {
				continue
			}

			vencendo = append(vencendo, map[string]interface{}{
				"produto_id":     l.ProdutoID,
				"nome":           produto.Nome,
				"lote_id":        l.ID,
				"numero_lote":    l.NumeroLote,
				"quantidade":     l.Quantidade,
				"data_validade":  l.DataValidade,
				"dias_restantes": dias,
			})
		}
	}

	return map[string]interface{}{
		"baixo_estoque": baixoEstoque,
		"vencendo":      vencendo,
	}, nil
}
