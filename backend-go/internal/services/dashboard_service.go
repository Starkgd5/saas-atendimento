package services

import (
	"context"
	"database/sql"
	// "time"
)

type DashboardService struct {
	db *sql.DB
}

type DashboardMetrics struct {
	TotalClientes        int              `json:"total_clientes"`
	AtendimentosHoje     int              `json:"atendimentos_hoje"`
	AtendimentosMes      int              `json:"atendimentos_mes"`
	TicketMedio          float64          `json:"ticket_medio"`
	TaxaConversao        float64          `json:"taxa_conversao"`
	OrcamentosGerados    int              `json:"orcamentos_gerados"`
	TempoMedioEspera     int              `json:"tempo_medio_espera"` // em segundos
	HorarioPico          string           `json:"horario_pico"`
	ProdutosMaisVendidos []ProdutoVendido `json:"produtos_mais_vendidos"`
}

type ProdutoVendido struct {
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Total      float64 `json:"total"`
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{db: db}
}

func (s *DashboardService) GetMetrics(ctx context.Context, lojaID int) (*DashboardMetrics, error) {
	metrics := &DashboardMetrics{}

	// Total de clientes
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM clientes WHERE loja_id = ?", lojaID,
	).Scan(&metrics.TotalClientes)
	if err != nil {
		return nil, err
	}

	// Atendimentos hoje
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
	`, lojaID).Scan(&metrics.AtendimentosHoje)
	if err != nil {
		return nil, err
	}

	// Atendimentos no mês
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND MONTH(iniciado_em) = MONTH(CURDATE()) 
		AND YEAR(iniciado_em) = YEAR(CURDATE())
	`, lojaID).Scan(&metrics.AtendimentosMes)
	if err != nil {
		return nil, err
	}

	// Ticket médio
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(total), 0) FROM orcamentos 
		WHERE loja_id = ? AND status = 'aprovado'
	`, lojaID).Scan(&metrics.TicketMedio)
	if err != nil {
		return nil, err
	}

	// Taxa de conversão
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(
			(SELECT COUNT(*) FROM orcamentos WHERE loja_id = ? AND status = 'aprovado') * 100.0 / 
			NULLIF((SELECT COUNT(*) FROM orcamentos WHERE loja_id = ?), 0), 0
		)
	`, lojaID, lojaID).Scan(&metrics.TaxaConversao)
	if err != nil {
		return nil, err
	}

	// Orçamentos gerados
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orcamentos WHERE loja_id = ? AND DATE(created_at) = CURDATE()
	`, lojaID).Scan(&metrics.OrcamentosGerados)
	if err != nil {
		return nil, err
	}

	// Produtos mais vendidos
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.nome, SUM(o.quantidade) as total_qtd, SUM(o.total) as total_vendido
		FROM orcamentos o
		JOIN produtos p ON o.produto_id = p.id
		WHERE o.loja_id = ? AND o.status = 'aprovado'
		GROUP BY p.id, p.nome
		ORDER BY total_qtd DESC
		LIMIT 5
	`, lojaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var produtos []ProdutoVendido
	for rows.Next() {
		var p ProdutoVendido
		if err := rows.Scan(&p.Nome, &p.Quantidade, &p.Total); err != nil {
			return nil, err
		}
		produtos = append(produtos, p)
	}
	metrics.ProdutosMaisVendidos = produtos

	return metrics, nil
}
