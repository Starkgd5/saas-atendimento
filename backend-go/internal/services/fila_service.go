package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/Starkgd5/saas-atendimento/internal/models"
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

// Chaves do Redis
const (
	keyFilaEspera     = "fila:espera"      // Sorted Set: cliente_id -> timestamp
	keyFilaAtendimento = "fila:atendimento" // Set: cliente_ids em atendimento
	keyFilaLimite     = "fila:limite"      // String: limite máximo
	keyClienteInfo    = "cliente:info:%d"  // Hash: dados do cliente na fila
)

// AdicionarClienteFila adiciona um cliente à fila de espera
func (s *FilaService) AdicionarClienteFila(ctx context.Context, clienteID int, lojaID int) error {
	// 1. Verificar se o cliente já está na fila
	exists, err := s.estaNaFila(ctx, clienteID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("cliente %d já está na fila", clienteID)
	}

	// 2. Verificar se atingiu o limite de atendimentos simultâneos
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return err
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
			return err
		}
	} else {
		// Entra direto em atendimento
		err = s.iniciarAtendimento(ctx, clienteID)
		if err != nil {
			return err
		}
	}

	// Salvar informações do cliente na fila
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	info := map[string]interface{}{
		"cliente_id": clienteID,
		"loja_id":    lojaID,
		"entrada":    time.Now().Unix(),
	}
	infoJSON, _ := json.Marshal(info)
	err = s.redisClient.Set(ctx, infoKey, infoJSON, 24*time.Hour).Err()
	if err != nil {
		return err
	}

	return nil
}

// ProximoClienteAtendimento puxa o próximo cliente da fila para atendimento
func (s *FilaService) ProximoClienteAtendimento(ctx context.Context) (int, error) {
	// 1. Verificar se há vagas
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return 0, err
	}
	if emAtendimento >= int64(s.maxClientes) {
		return 0, fmt.Errorf("limite de atendimentos atingido: %d", s.maxClientes)
	}

	// 2. Buscar próximo da fila de espera (menor timestamp)
	result, err := s.redisClient.ZPopMin(ctx, keyFilaEspera, 1).Result()
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil // Fila vazia
	}

	clienteID := int(result[0].Member.(int))

	// 3. Mover para atendimento
	err = s.iniciarAtendimento(ctx, clienteID)
	if err != nil {
		return 0, err
	}

	return clienteID, nil
}

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

// FinalizarAtendimento remove um cliente do atendimento
func (s *FilaService) FinalizarAtendimento(ctx context.Context, clienteID int) error {
	// Remover do atendimento
	err := s.redisClient.SRem(ctx, keyFilaAtendimento, clienteID).Err()
	if err != nil {
		return err
	}

	// Limpar informações
	infoKey := fmt.Sprintf(keyClienteInfo, clienteID)
	err = s.redisClient.Del(ctx, infoKey).Err()
	if err != nil {
		return err
	}

	return nil
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

// GetStatusFila retorna o status atual da fila
func (s *FilaService) GetStatusFila(ctx context.Context) (*models.FilaStatus, error) {
	emAtendimento, err := s.redisClient.SCard(ctx, keyFilaAtendimento).Result()
	if err != nil {
		return nil, err
	}

	emEspera, err := s.redisClient.ZCard(ctx, keyFilaEspera).Result()
	if err != nil {
		return nil, err
	}

	return &models.FilaStatus{
		EmAtendimento: int(emAtendimento),
		EmEspera:      int(emEspera),
		Limite:        s.maxClientes,
	}, nil
}

// SetLimite atualiza o limite máximo de atendimentos
func (s *FilaService) SetLimite(ctx context.Context, limite int) error {
	s.maxClientes = limite
	// Salvar no Redis para persistência
	return s.redisClient.Set(ctx, keyFilaLimite, limite, 0).Err()
}