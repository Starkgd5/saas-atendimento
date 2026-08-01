package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/redis/go-redis/v9"
)

type FilaService struct {
	redisClient *redis.Client
	maxClientes int
}

func NewFilaService(redisClient *redis.Client, maxClientes int) *FilaService {
	return &FilaService{
		redisClient: redisClient,
		maxClientes: maxClientes,
	}
}

// ============================================
// CHAVES DO REDIS
// ============================================

const (
	keyFilaEspera      = "fila:espera"      // Sorted Set: cliente_id -> timestamp
	keyFilaAtendimento = "fila:atendimento" // Set: cliente_ids em atendimento
	keyFilaLimite      = "fila:limite"      // String: limite máximo
	keyClienteInfo     = "cliente:info:%d"  // String: dados do cliente na fila
	keyFilaStats       = "fila:stats"       // Hash: estatísticas da fila
	keyFilaBloqueio    = "fila:bloqueio:%d" // String: bloqueio de cliente (anti-spam)
)

// ============================================
// OPERAÇÕES PRINCIPAIS
// ============================================

// AdicionarClienteFila adiciona um cliente à fila de espera
func (s *FilaService) AdicionarClienteFila(ctx context.Context, clienteID int, lojaID int) error {
	// 1. Verificar se o cliente já está na fila
	exists, err := s.estaNaFila(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("erro ao verificar cliente na fila: %w", err)
	}
	if exists {
		return fmt.Errorf("cliente %d já está na fila", clienteID)
	}

	// 2. Verificar bloqueio (anti-spam)
	bloqueado, err := s.estaBloqueado(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("erro ao verificar bloqueio: %w", err)
	}
	if bloqueado {
		return fmt.Errorf("cliente %d está bloqueado temporariamente", clienteID)
	}

	// 3. Verificar se atingiu o limite de atendimentos simultâneos
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return fmt.Errorf("erro ao verificar atendimentos: %w", err)
	}

	limite := s.maxClientes
	if emAtendimento >= int64(limite) {
		// Coloca na espera
		timestamp := float64(time.Now().UnixNano())
		err = s.redisClient.ZAdd(ctx, keyFilaEspera, redis.Z{
			Score:  timestamp,
			Member: clienteID,
		}).Err()
		if err != nil {
			return fmt.Errorf("erro ao adicionar à fila de espera: %w", err)
		}
	} else {
		// Entra direto em atendimento
		err = s.iniciarAtendimento(ctx, clienteID)
		if err != nil {
			return fmt.Errorf("erro ao iniciar atendimento: %w", err)
		}
	}

	// 4. Salvar informações do cliente na fila
	if err := s.salvarInfoCliente(ctx, clienteID, lojaID); err != nil {
		return err
	}

	// 5. Atualizar estatísticas
	if err := s.atualizarEstatisticas(ctx); err != nil {
		// Não falha a operação, apenas loga o erro
		_ = err
	}

	return nil
}

// ProximoClienteAtendimento puxa o próximo cliente da fila para atendimento
func (s *FilaService) ProximoClienteAtendimento(ctx context.Context) (int, error) {
	// 1. Verificar se há vagas
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return 0, fmt.Errorf("erro ao verificar atendimentos: %w", err)
	}
	if emAtendimento >= int64(s.maxClientes) {
		return 0, fmt.Errorf("limite de atendimentos atingido: %d", s.maxClientes)
	}

	// 2. Buscar próximo da fila de espera (menor timestamp)
	result, err := s.redisClient.ZPopMin(ctx, keyFilaEspera, 1).Result()
	if err != nil {
		return 0, fmt.Errorf("erro ao buscar próximo cliente: %w", err)
	}
	if len(result) == 0 {
		return 0, nil // Fila vazia
	}

	clienteID := int(result[0].Member.(int))

	// 3. Mover para atendimento
	err = s.iniciarAtendimento(ctx, clienteID)
	if err != nil {
		// Se falhar, recolocar na fila
		timestamp := float64(time.Now().UnixNano())
		_ = s.redisClient.ZAdd(ctx, keyFilaEspera, redis.Z{
			Score:  timestamp,
			Member: clienteID,
		}).Err()
		return 0, fmt.Errorf("erro ao iniciar atendimento: %w", err)
	}

	// 4. Atualizar estatísticas
	_ = s.atualizarEstatisticas(ctx)

	return clienteID, nil
}

// RemoverClienteFila remove um cliente da fila (voluntário ou abandono)
func (s *FilaService) RemoverClienteFila(ctx context.Context, clienteID int, motivo string) error {
	// Remover da espera
	if err := s.redisClient.ZRem(ctx, keyFilaEspera, clienteID).Err(); err != nil {
		return fmt.Errorf("erro ao remover da espera: %w", err)
	}

	// Remover do atendimento
	if err := s.redisClient.SRem(ctx, keyFilaAtendimento, clienteID).Err(); err != nil {
		return fmt.Errorf("erro ao remover do atendimento: %w", err)
	}

	// Limpar informações
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	if err := s.redisClient.Del(ctx, infoKey).Err(); err != nil {
		return fmt.Errorf("erro ao limpar informações: %w", err)
	}

	// Se motivo for abandono, adicionar bloqueio temporário
	if motivo == "abandono" {
		if err := s.bloquearCliente(ctx, clienteID, 5*time.Minute); err != nil {
			_ = err // Não falha a operação
		}
	}

	// Atualizar estatísticas
	_ = s.atualizarEstatisticas(ctx)

	return nil
}

// ============================================
// FINALIZAR ATENDIMENTO
// ============================================

// FinalizarAtendimento remove um cliente do atendimento e atualiza os status
func (s *FilaService) FinalizarAtendimento(ctx context.Context, clienteID int) error {
	// 1. Verificar se o cliente está em atendimento
	isMember, err := s.redisClient.SIsMember(ctx, keyFilaAtendimento, clienteID).Result()
	if err != nil {
		return fmt.Errorf("erro ao verificar cliente em atendimento: %w", err)
	}
	if !isMember {
		return fmt.Errorf("cliente %d não está em atendimento", clienteID)
	}

	// 2. Remover do atendimento
	err = s.redisClient.SRem(ctx, keyFilaAtendimento, clienteID).Err()
	if err != nil {
		return fmt.Errorf("erro ao remover cliente do atendimento: %w", err)
	}

	// 3. Remover da fila de espera (caso esteja)
	err = s.redisClient.ZRem(ctx, keyFilaEspera, clienteID).Err()
	if err != nil {
		// Não falha a operação, apenas loga
		_ = err
	}

	// 4. Limpar informações do cliente na fila
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	err = s.redisClient.Del(ctx, infoKey).Err()
	if err != nil {
		// Não falha a operação, apenas loga
		_ = err
	}

	// 5. Atualizar estatísticas da fila
	err = s.atualizarEstatisticas(ctx)
	if err != nil {
		// Não falha a operação, apenas loga
		_ = err
	}

	return nil
}

// FinalizarAtendimentoComDados é uma versão mais completa que também atualiza o banco
// Nota: Este método requer um *sql.DB para atualizar o banco
func (s *FilaService) FinalizarAtendimentoComDados(ctx context.Context, clienteID int, atendimentoID int, db interface{}) error {
	// 1. Finalizar na fila Redis
	err := s.FinalizarAtendimento(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("erro ao finalizar na fila: %w", err)
	}

	// 2. Atualizar status no banco de dados (se disponível)
	if db != nil && atendimentoID > 0 {
		type dbExecutor interface {
			Exec(query string, args ...interface{}) (interface{}, error)
		}

		if executor, ok := db.(dbExecutor); ok {
			query := `
				UPDATE atendimentos 
				SET status = 'finalizado', finalizado_em = NOW(), 
				    tempo_atendimento = TIMESTAMPDIFF(SECOND, iniciado_em, NOW())
				WHERE id = ?
			`
			_, err := executor.Exec(query, atendimentoID)
			if err != nil {
				return fmt.Errorf("erro ao atualizar atendimento no banco: %w", err)
			}

			// Atualizar último atendimento do cliente
			_, err = executor.Exec(`
				UPDATE clientes SET ultimo_atendimento = NOW() WHERE id = ?
			`, clienteID)
			if err != nil {
				// Não falha a operação, apenas loga
				_ = err
			}
		}
	}

	// 3. Atualizar estatísticas
	_ = s.atualizarEstatisticas(ctx)

	return nil
}

// AbandonarAtendimento marca um atendimento como abandonado
func (s *FilaService) AbandonarAtendimento(ctx context.Context, clienteID int, atendimentoID int, db interface{}) error {
	// 1. Remover da fila
	err := s.FinalizarAtendimento(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("erro ao remover da fila: %w", err)
	}

	// 2. Marcar como abandonado no banco
	if db != nil && atendimentoID > 0 {
		type dbExecutor interface {
			Exec(query string, args ...interface{}) (interface{}, error)
		}

		if executor, ok := db.(dbExecutor); ok {
			query := `
				UPDATE atendimentos 
				SET status = 'abandonado', finalizado_em = NOW(), 
				    tempo_atendimento = TIMESTAMPDIFF(SECOND, iniciado_em, NOW())
				WHERE id = ?
			`
			_, err := executor.Exec(query, atendimentoID)
			if err != nil {
				return fmt.Errorf("erro ao marcar atendimento como abandonado: %w", err)
			}
		}
	}

	// 3. Bloquear cliente temporariamente (anti-spam)
	err = s.bloquearCliente(ctx, clienteID, 5*time.Minute)
	if err != nil {
		// Não falha a operação, apenas loga
		_ = err
	}

	// 4. Atualizar estatísticas
	_ = s.atualizarEstatisticas(ctx)

	return nil
}

// LiberarClienteBloqueado remove o bloqueio de um cliente
func (s *FilaService) LiberarClienteBloqueado(ctx context.Context, clienteID int) error {
	blockKey := fmt.Sprintf(keyFilaBloqueio, clienteID)
	return s.redisClient.Del(ctx, blockKey).Err()
}

// VerificarClienteBloqueado verifica se um cliente está bloqueado
func (s *FilaService) VerificarClienteBloqueado(ctx context.Context, clienteID int) (bool, error) {
	return s.estaBloqueado(ctx, clienteID)
}

// GetTempoMedioAtendimento calcula o tempo médio de atendimento atual
func (s *FilaService) GetTempoMedioAtendimento(ctx context.Context) (int64, error) {
	members, err := s.redisClient.SMembers(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return 0, err
	}

	if len(members) == 0 {
		return 0, nil
	}

	var totalTempo int64
	agora := time.Now().UnixNano()

	for _, member := range members {
		clienteID, _ := strconv.Atoi(member)
		infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
		infoJSON, err := s.redisClient.Get(ctx, infoKey).Result()
		if err == nil {
			var info map[string]interface{}
			if err := json.Unmarshal([]byte(infoJSON), &info); err == nil {
				if entrada, ok := info["entrada"].(float64); ok {
					tempoAtendimento := (agora - int64(entrada)) / int64(time.Second)
					totalTempo += tempoAtendimento
				}
			}
		}
	}

	if len(members) > 0 {
		return totalTempo / int64(len(members)), nil
	}
	return 0, nil
}

// GetClientesEmAtendimento retorna a lista de clientes em atendimento
func (s *FilaService) GetClientesEmAtendimento(ctx context.Context) ([]map[string]interface{}, error) {
	members, err := s.redisClient.SMembers(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return nil, err
	}

	var clientes []map[string]interface{}
	for _, member := range members {
		clienteID, _ := strconv.Atoi(member)
		info, err := s.getInfoCliente(ctx, clienteID)
		if err == nil && info != nil {
			info["status"] = "em_atendimento"
			clientes = append(clientes, info)
		}
	}

	return clientes, nil
}

// GetHistoricoCliente retorna o histórico de atendimentos de um cliente
func (s *FilaService) GetHistoricoCliente(ctx context.Context, clienteID int) ([]map[string]interface{}, error) {
	info, err := s.getInfoCliente(ctx, clienteID)
	if err != nil {
		return nil, err
	}

	historico := []map[string]interface{}{}
	if info != nil {
		naFila, _ := s.estaNaFila(ctx, clienteID)
		info["na_fila"] = naFila

		bloqueado, _ := s.estaBloqueado(ctx, clienteID)
		info["bloqueado"] = bloqueado

		historico = append(historico, info)
	}

	return historico, nil
}

// ============================================
// OPERAÇÕES INTERNAS
// ============================================

// iniciarAtendimento move um cliente para atendimento ativo
func (s *FilaService) iniciarAtendimento(ctx context.Context, clienteID int) error {
	// Adicionar ao set de atendimento
	err := s.redisClient.SAdd(ctx, keyFilaAtendimento, clienteID).Err()
	if err != nil {
		return err
	}

	// Remover da fila de espera
	err = s.redisClient.ZRem(ctx, keyFilaEspera, clienteID).Err()
	if err != nil {
		return err
	}

	return nil
}

// salvarInfoCliente salva as informações do cliente na fila
func (s *FilaService) salvarInfoCliente(ctx context.Context, clienteID int, lojaID int) error {
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	info := map[string]interface{}{
		"cliente_id": clienteID,
		"loja_id":    lojaID,
		"entrada":    time.Now().Unix(),
		"status":     "na_fila",
	}
	infoJSON, _ := json.Marshal(info)
	return s.redisClient.Set(ctx, infoKey, infoJSON, 24*time.Hour).Err()
}

// bloquearCliente bloqueia um cliente temporariamente
func (s *FilaService) bloquearCliente(ctx context.Context, clienteID int, duracao time.Duration) error {
	key := fmt.Sprintf(keyFilaBloqueio, clienteID)
	return s.redisClient.Set(ctx, key, "1", duracao).Err()
}

// estaBloqueado verifica se um cliente está bloqueado
func (s *FilaService) estaBloqueado(ctx context.Context, clienteID int) (bool, error) {
	key := fmt.Sprintf(keyFilaBloqueio, clienteID)
	exists, err := s.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// estaNaFila verifica se o cliente está na fila (espera ou atendimento)
func (s *FilaService) estaNaFila(ctx context.Context, clienteID int) (bool, error) {
	// Verificar em espera
	_, err := s.redisClient.ZScore(ctx, keyFilaEspera, strconv.Itoa(clienteID)).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if err == nil {
		return true, nil
	}

	// Verificar em atendimento
	isMember, err := s.redisClient.SIsMember(ctx, keyFilaAtendimento, clienteID).Result()
	if err != nil {
		return false, err
	}

	return isMember, nil
}

// atualizarEstatisticas atualiza as estatísticas da fila
func (s *FilaService) atualizarEstatisticas(ctx context.Context) error {
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return err
	}

	emEspera, err := s.redisClient.ZCard(ctx, keyFilaEspera).Result()
	if err != nil {
		return err
	}

	stats := map[string]interface{}{
		"em_atendimento": emAtendimento,
		"em_espera":      emEspera,
		"limite":         s.maxClientes,
		"atualizado_em":  time.Now().Unix(),
	}

	statsJSON, _ := json.Marshal(stats)
	return s.redisClient.Set(ctx, keyFilaStats, statsJSON, 0).Err()
}

// ============================================
// CONSULTAS
// ============================================

// GetStatusFila retorna o status atual da fila
func (s *FilaService) GetStatusFila(ctx context.Context) (*models.FilaStatus, error) {
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos: %w", err)
	}

	emEspera, err := s.redisClient.ZCard(ctx, keyFilaEspera).Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar espera: %w", err)
	}

	return &models.FilaStatus{
		EmAtendimento: int(emAtendimento),
		EmEspera:      int(emEspera),
		Limite:        s.maxClientes,
	}, nil
}

// GetClientesNaFila retorna a lista de clientes na fila
func (s *FilaService) GetClientesNaFila(ctx context.Context) ([]map[string]interface{}, error) {
	// Buscar clientes em atendimento
	atendimentoMembers, err := s.redisClient.SMembers(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atendimentos: %w", err)
	}

	// Buscar clientes em espera
	esperaMembers, err := s.redisClient.ZRange(ctx, keyFilaEspera, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar espera: %w", err)
	}

	var clientes []map[string]interface{}

	// Adicionar clientes em atendimento
	for _, member := range atendimentoMembers {
		clienteID, _ := strconv.Atoi(member)
		info, _ := s.getInfoCliente(ctx, clienteID)
		if info != nil {
			info["status"] = "em_atendimento"
			info["posicao"] = 0
			clientes = append(clientes, info)
		}
	}

	// Adicionar clientes em espera
	for posicao, member := range esperaMembers {
		clienteID, _ := strconv.Atoi(member)
		info, _ := s.getInfoCliente(ctx, clienteID)
		if info != nil {
			info["status"] = "aguardando"
			info["posicao"] = posicao + 1
			clientes = append(clientes, info)
		}
	}

	return clientes, nil
}

// getInfoCliente retorna as informações de um cliente na fila
func (s *FilaService) getInfoCliente(ctx context.Context, clienteID int) (map[string]interface{}, error) {
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	infoJSON, err := s.redisClient.Get(ctx, infoKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(infoJSON), &info); err != nil {
		return nil, err
	}

	return info, nil
}

// GetTempoMedioEspera calcula o tempo médio de espera atual
func (s *FilaService) GetTempoMedioEspera(ctx context.Context) (int64, error) {
	// Buscar todos os clientes na fila de espera
	members, err := s.redisClient.ZRangeWithScores(ctx, keyFilaEspera, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	if len(members) == 0 {
		return 0, nil
	}

	var totalEspera int64
	agora := time.Now().UnixNano()

	for _, member := range members {
		entrada := int64(member.Score)
		tempoEspera := (agora - entrada) / int64(time.Second)
		totalEspera += tempoEspera
	}

	return totalEspera / int64(len(members)), nil
}

// ============================================
// ADMINISTRAÇÃO
// ============================================

// SetLimite atualiza o limite máximo de atendimentos
func (s *FilaService) SetLimite(ctx context.Context, limite int) error {
	if limite < 1 || limite > 20 {
		return fmt.Errorf("limite deve ser entre 1 e 20")
	}

	s.maxClientes = limite
	if err := s.redisClient.Set(ctx, keyFilaLimite, limite, 0).Err(); err != nil {
		return fmt.Errorf("erro ao salvar limite: %w", err)
	}

	// Atualizar estatísticas
	_ = s.atualizarEstatisticas(ctx)

	return nil
}

// GetLimite retorna o limite máximo atual
func (s *FilaService) GetLimite(ctx context.Context) (int, error) {
	limite, err := s.redisClient.Get(ctx, keyFilaLimite).Int()
	if err == redis.Nil {
		return s.maxClientes, nil
	}
	if err != nil {
		return s.maxClientes, err
	}
	return limite, nil
}

// ResetarFila limpa toda a fila (emergência)
func (s *FilaService) ResetarFila(ctx context.Context) error {
	// Limpar todos os clientes
	if err := s.redisClient.Del(ctx, keyFilaEspera).Err(); err != nil {
		return fmt.Errorf("erro ao limpar espera: %w", err)
	}
	if err := s.redisClient.Del(ctx, keyFilaAtendimento).Err(); err != nil {
		return fmt.Errorf("erro ao limpar atendimento: %w", err)
	}
	if err := s.redisClient.Del(ctx, keyFilaStats).Err(); err != nil {
		return fmt.Errorf("erro ao limpar stats: %w", err)
	}

	// Limpar informações dos clientes (padrão cliente:info:*)
	iter := s.redisClient.Scan(ctx, 0, "cliente:info:*", 100).Iterator()
	for iter.Next(ctx) {
		if err := s.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			_ = err
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("erro ao limpar informações: %w", err)
	}

	return nil
}

// GetEstatisticas retorna estatísticas detalhadas da fila
func (s *FilaService) GetEstatisticas(ctx context.Context) (map[string]interface{}, error) {
	statsJSON, err := s.redisClient.Get(ctx, keyFilaStats).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// ============================================
// MÉTRICAS E MONITORAMENTO
// ============================================

// GetMetricasFila retorna métricas para monitoramento
func (s *FilaService) GetMetricasFila(ctx context.Context) (map[string]interface{}, error) {
	status, err := s.GetStatusFila(ctx)
	if err != nil {
		return nil, err
	}

	tempoEspera, err := s.GetTempoMedioEspera(ctx)
	if err != nil {
		return nil, err
	}

	clientes, err := s.GetClientesNaFila(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":              status,
		"tempo_medio_espera":  tempoEspera,
		"total_clientes_fila": len(clientes),
		"timestamp":           time.Now().Unix(),
	}, nil
}
