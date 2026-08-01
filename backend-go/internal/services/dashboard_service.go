package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
)

type DashboardService struct {
	db *sql.DB
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{db: db}
}

// ============================================
// MÉTRICAS PRINCIPAIS
// ============================================

// GetMetrics retorna todas as métricas do dashboard
func (s *DashboardService) GetMetrics(ctx context.Context, lojaID int) (*models.DashboardMetrics, error) {
	metrics := &models.DashboardMetrics{}

	var err error

	// Total de clientes
	metrics.TotalClientes, err = s.GetTotalClientes(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar total de clientes: %w", err)
	}

	// Atendimentos hoje
	metrics.AtendimentosHoje, err = s.GetAtendimentosHoje(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos de hoje: %w", err)
	}

	// Atendimentos do mês
	metrics.AtendimentosMes, err = s.GetAtendimentosMes(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos do mês: %w", err)
	}

	// Ticket médio
	metrics.TicketMedio, err = s.GetTicketMedio(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar ticket médio: %w", err)
	}

	// Taxa de conversão
	metrics.TaxaConversao, err = s.GetTaxaConversao(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar taxa de conversão: %w", err)
	}

	// Orçamentos gerados hoje
	metrics.OrcamentosGerados, err = s.GetOrcamentosHoje(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar orçamentos gerados: %w", err)
	}

	// Tempo médio de espera
	metrics.TempoMedioEspera, err = s.GetTempoMedioEspera(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tempo médio de espera: %w", err)
	}

	// Tempo médio de atendimento
	metrics.TempoMedioAtendimento, err = s.GetTempoMedioAtendimento(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar tempo médio de atendimento: %w", err)
	}

	// Horário de pico
	metrics.HorarioPico, err = s.GetHorarioPico(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar horário de pico: %w", err)
	}

	// Produtos mais vendidos
	metrics.ProdutosMaisVendidos, err = s.GetProdutosMaisVendidos(ctx, lojaID, 5)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar produtos mais vendidos: %w", err)
	}

	// Reclamações pendentes
	metrics.ReclamacoesPendentes, err = s.GetReclamacoesPendentes(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar reclamações pendentes: %w", err)
	}

	// Total finalizados
	metrics.TotalFinalizados, err = s.GetTotalFinalizados(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar total finalizados: %w", err)
	}

	// Abandonos
	metrics.Abandonos, err = s.GetAbandonos(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar abandonos: %w", err)
	}

	// Taxa de abandono
	if metrics.TotalFinalizados+metrics.Abandonos > 0 {
		metrics.TaxaAbandono = float64(metrics.Abandonos) / float64(metrics.TotalFinalizados+metrics.Abandonos) * 100
	}

	return metrics, nil
}

// ============================================
// MÉTRICAS INDIVIDUAIS
// ============================================

// GetTotalClientes retorna o total de clientes
func (s *DashboardService) GetTotalClientes(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM clientes WHERE loja_id = ?
	`, lojaID).Scan(&count)
	return count, err
}

// GetAtendimentosHoje retorna o total de atendimentos de hoje
func (s *DashboardService) GetAtendimentosHoje(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
	`, lojaID).Scan(&count)
	return count, err
}

// GetAtendimentosMes retorna o total de atendimentos do mês
func (s *DashboardService) GetAtendimentosMes(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND MONTH(iniciado_em) = MONTH(CURDATE()) 
		AND YEAR(iniciado_em) = YEAR(CURDATE())
	`, lojaID).Scan(&count)
	return count, err
}

// GetTicketMedio retorna o ticket médio
func (s *DashboardService) GetTicketMedio(ctx context.Context, lojaID int) (float64, error) {
	var avg float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(total), 0) FROM orcamentos 
		WHERE loja_id = ? AND status = ?
	`, lojaID, models.OrcamentoAprovado).Scan(&avg)
	return avg, err
}

// GetTaxaConversao retorna a taxa de conversão (orçamentos aprovados / total)
func (s *DashboardService) GetTaxaConversao(ctx context.Context, lojaID int) (float64, error) {
	var taxa float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(
			(SELECT COUNT(*) FROM orcamentos WHERE loja_id = ? AND status = ?) * 100.0 / 
			NULLIF((SELECT COUNT(*) FROM orcamentos WHERE loja_id = ?), 0), 0
		)
	`, lojaID, models.OrcamentoAprovado, lojaID).Scan(&taxa)
	return taxa, err
}

// GetOrcamentosHoje retorna o total de orçamentos gerados hoje
func (s *DashboardService) GetOrcamentosHoje(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orcamentos 
		WHERE loja_id = ? AND DATE(created_at) = CURDATE()
	`, lojaID).Scan(&count)
	return count, err
}

// GetTempoMedioEspera retorna o tempo médio de espera em segundos
func (s *DashboardService) GetTempoMedioEspera(ctx context.Context, lojaID int) (int, error) {
	var avg sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT AVG(tempo_espera)
		FROM atendimentos
		WHERE loja_id = ? AND tempo_espera IS NOT NULL
	`, lojaID).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return int(avg.Int64), nil
}

// GetTempoMedioAtendimento retorna o tempo médio de atendimento em segundos
func (s *DashboardService) GetTempoMedioAtendimento(ctx context.Context, lojaID int) (float64, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT AVG(TIMESTAMPDIFF(SECOND, iniciado_em, finalizado_em))
		FROM atendimentos
		WHERE loja_id = ? AND status = ? AND finalizado_em IS NOT NULL
	`, lojaID, models.StatusFinalizado).Scan(&avg)
	if err != nil {
		return 0, err
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// GetHorarioPico retorna o horário de pico de atendimentos
func (s *DashboardService) GetHorarioPico(ctx context.Context, lojaID int) (string, error) {
	var horario string
	err := s.db.QueryRowContext(ctx, `
		SELECT CONCAT(
			LPAD(HOUR(iniciado_em), 2, '0'), ':00 - ',
			LPAD(HOUR(iniciado_em) + 1, 2, '0'), ':00'
		) as horario
		FROM atendimentos
		WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
		GROUP BY HOUR(iniciado_em)
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, lojaID).Scan(&horario)
	if err == sql.ErrNoRows {
		return "N/A", nil
	}
	return horario, err
}

// GetProdutosMaisVendidos retorna os produtos mais vendidos
func (s *DashboardService) GetProdutosMaisVendidos(ctx context.Context, lojaID int, limit int) ([]models.ProdutoVendido, error) {
	if limit == 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			p.nome as produto_nome,
			SUM(oi.quantidade) as total_quantidade,
			SUM(oi.total) as total_vendido
		FROM orcamento_itens oi
		JOIN orcamentos o ON oi.orcamento_id = o.id
		JOIN produtos p ON oi.produto_id = p.id
		WHERE o.loja_id = ? AND o.status = ?
		GROUP BY p.id, p.nome
		ORDER BY total_quantidade DESC
		LIMIT ?
	`, lojaID, models.OrcamentoAprovado, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var produtos []models.ProdutoVendido
	for rows.Next() {
		var p models.ProdutoVendido
		if err := rows.Scan(&p.Nome, &p.Quantidade, &p.Total); err != nil {
			return nil, err
		}
		produtos = append(produtos, p)
	}

	return produtos, nil
}

// GetReclamacoesPendentes retorna o número de reclamações pendentes
func (s *DashboardService) GetReclamacoesPendentes(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reclamacoes 
		WHERE loja_id = ? AND status = ?
	`, lojaID, models.ReclamacaoPendente).Scan(&count)
	return count, err
}

// GetTotalFinalizados retorna o total de atendimentos finalizados
func (s *DashboardService) GetTotalFinalizados(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusFinalizado).Scan(&count)
	return count, err
}

// GetAbandonos retorna o total de atendimentos abandonados
func (s *DashboardService) GetAbandonos(ctx context.Context, lojaID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos 
		WHERE loja_id = ? AND status = ?
	`, lojaID, models.StatusAbandonado).Scan(&count)
	return count, err
}

// ============================================
// MÉTRICAS AVANÇADAS
// ============================================

// GetMetricasDiarias retorna métricas diárias para gráficos
func (s *DashboardService) GetMetricasDiarias(ctx context.Context, lojaID int, dias int) ([]map[string]interface{}, error) {
	if dias == 0 {
		dias = 7
	}

	query := `
		SELECT 
			DATE(iniciado_em) as data,
			COUNT(*) as total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as finalizados,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as abandonados,
			AVG(TIMESTAMPDIFF(SECOND, iniciado_em, finalizado_em)) as tempo_medio
		FROM atendimentos
		WHERE loja_id = ? AND iniciado_em >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		GROUP BY DATE(iniciado_em)
		ORDER BY data DESC
	`

	rows, err := s.db.QueryContext(ctx, query,
		models.StatusFinalizado,
		models.StatusAbandonado,
		lojaID,
		dias,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []map[string]interface{}
	for rows.Next() {
		var data string
		var total, finalizados, abandonados int
		var tempoMedio sql.NullFloat64

		if err := rows.Scan(&data, &total, &finalizados, &abandonados, &tempoMedio); err != nil {
			return nil, err
		}

		resultado := map[string]interface{}{
			"data":          data,
			"total":         total,
			"finalizados":   finalizados,
			"abandonados":   abandonados,
			"taxa_sucesso":  float64(finalizados) / float64(total+1) * 100,
		}
		if tempoMedio.Valid {
			resultado["tempo_medio"] = tempoMedio.Float64
		}
		resultados = append(resultados, resultado)
	}

	return resultados, nil
}

// GetMetricasPorHora retorna métricas por hora do dia
func (s *DashboardService) GetMetricasPorHora(ctx context.Context, lojaID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			HOUR(iniciado_em) as hora,
			COUNT(*) as total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as finalizados
		FROM atendimentos
		WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
		GROUP BY HOUR(iniciado_em)
		ORDER BY hora ASC
	`

	rows, err := s.db.QueryContext(ctx, query, models.StatusFinalizado, lojaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []map[string]interface{}
	for rows.Next() {
		var hora, total, finalizados int

		if err := rows.Scan(&hora, &total, &finalizados); err != nil {
			return nil, err
		}

		resultados = append(resultados, map[string]interface{}{
			"hora":        hora,
			"total":       total,
			"finalizados": finalizados,
		})
	}

	return resultados, nil
}

// GetSatisfacaoCliente retorna métricas de satisfação (baseado em reclamações)
func (s *DashboardService) GetSatisfacaoCliente(ctx context.Context, lojaID int) (map[string]interface{}, error) {
	var totalReclamacoes, resolvidas int

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as resolvidas
		FROM reclamacoes
		WHERE loja_id = ?
	`, lojaID, models.ReclamacaoResolvido).Scan(&totalReclamacoes, &resolvidas)

	if err != nil {
		return nil, err
	}

	taxaResolucao := 0.0
	if totalReclamacoes > 0 {
		taxaResolucao = float64(resolvidas) / float64(totalReclamacoes) * 100
	}

	return map[string]interface{}{
		"total_reclamacoes": totalReclamacoes,
		"resolvidas":        resolvidas,
		"taxa_resolucao":    taxaResolucao,
		"score":             100 - (float64(totalReclamacoes-resolvidas) / float64(totalReclamacoes+1) * 10),
	}, nil
}

// ============================================
// RELATÓRIOS
// ============================================

// GetRelatorioCompleto retorna um relatório completo com todas as métricas
func (s *DashboardService) GetRelatorioCompleto(ctx context.Context, lojaID int, periodo string) (map[string]interface{}, error) {
	metrics, err := s.GetMetrics(ctx, lojaID)
	if err != nil {
		return nil, err
	}

	metricasDiarias, err := s.GetMetricasDiarias(ctx, lojaID, 7)
	if err != nil {
		return nil, err
	}

	metricasPorHora, err := s.GetMetricasPorHora(ctx, lojaID)
	if err != nil {
		return nil, err
	}

	satisfacao, err := s.GetSatisfacaoCliente(ctx, lojaID)
	if err != nil {
		return nil, err
	}

	// Calcular crescimento
	var crescimento float64
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COALESCE(
				(SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND MONTH(iniciado_em) = MONTH(CURDATE()) AND YEAR(iniciado_em) = YEAR(CURDATE())) /
				NULLIF((SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND MONTH(iniciado_em) = MONTH(DATE_SUB(CURDATE(), INTERVAL 1 MONTH)) AND YEAR(iniciado_em) = YEAR(DATE_SUB(CURDATE(), INTERVAL 1 MONTH))), 0) * 100 - 100,
				0
			)
	`, lojaID, lojaID).Scan(&crescimento)

	if err != nil && err != sql.ErrNoRows {
		crescimento = 0
	}

	return map[string]interface{}{
		"metrics":           metrics,
		"metricas_diarias":  metricasDiarias,
		"metricas_por_hora": metricasPorHora,
		"satisfacao":        satisfacao,
		"crescimento":       crescimento,
		"periodo":           periodo,
		"gerado_em":         time.Now().Format(time.RFC3339),
	}, nil
}