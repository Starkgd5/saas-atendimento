package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type cacheService struct {
	client *redis.Client
}

// ============================================
// OPERAÇÕES BÁSICAS
// ============================================

// Get busca um valor do cache
func (c *cacheService) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

// GetString busca um valor como string
func (c *cacheService) GetString(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// GetJSON busca e decodifica um valor JSON do cache
func (c *cacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Set armazena um valor no cache com TTL
func (c *cacheService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("erro ao serializar valor: %w", err)
		}
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

// SetString armazena uma string no cache
func (c *cacheService) SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete remove uma chave do cache
func (c *cacheService) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists verifica se uma chave existe no cache
func (c *cacheService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// TTL retorna o tempo restante de uma chave
func (c *cacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// ============================================
// OPERAÇÕES EM LOTE
// ============================================

// MGet busca múltiplas chaves
func (c *cacheService) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return c.client.MGet(ctx, keys...).Result()
}

// MSet armazena múltiplos valores
func (c *cacheService) MSet(ctx context.Context, values map[string]interface{}, ttl time.Duration) error {
	pipe := c.client.Pipeline()
	for key, val := range values {
		pipe.Set(ctx, key, val, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// ============================================
// LIMPEZA E INVALIDAÇÃO
// ============================================

// ClearByPattern limpa todas as chaves que correspondem a um padrão
func (c *cacheService) ClearByPattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// ClearByPrefix limpa todas as chaves com um prefixo
func (c *cacheService) ClearByPrefix(ctx context.Context, prefix string) error {
	return c.ClearByPattern(ctx, prefix+"*")
}

// ClearAll limpa todas as chaves do cache
func (c *cacheService) ClearAll(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// ============================================
// GET OR SET (PADRÃO)
// ============================================

// GetOrSet tenta buscar do cache, se não encontrar, executa a função e armazena
func (c *cacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	// 1. Tentar buscar do cache
	cached, err := c.Get(ctx, key)
	if err == nil {
		var result interface{}
		if json.Unmarshal(cached, &result) == nil {
			return result, nil
		}
		// Se não conseguir decodificar, tenta como string
		return string(cached), nil
	}

	// 2. Executar função para gerar novo valor
	result, err := fn()
	if err != nil {
		return nil, err
	}

	// 3. Salvar no cache
	if err := c.Set(ctx, key, result, ttl); err != nil {
		// Não falha a operação, apenas loga o erro
		_ = err
	}

	return result, nil
}

// GetOrSetJSON é similar ao GetOrSet mas com tipagem
func (c *cacheService) GetOrSetJSON(ctx context.Context, key string, ttl time.Duration, dest interface{}, fn func() (interface{}, error)) error {
	// 1. Tentar buscar do cache
	err := c.GetJSON(ctx, key, dest)
	if err == nil {
		return nil
	}

	// 2. Executar função para gerar novo valor
	result, err := fn()
	if err != nil {
		return err
	}

	// 3. Salvar no cache
	if err := c.Set(ctx, key, result, ttl); err != nil {
		// Não falha a operação, apenas loga o erro
		_ = err
	}

	// 4. Retornar o resultado
	data, _ := json.Marshal(result)
	return json.Unmarshal(data, dest)
}

// ============================================
// CONTADORES E INCREMENTOS
// ============================================

// Incr incrementa um contador
func (c *cacheService) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// IncrBy incrementa um contador por um valor específico
func (c *cacheService) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// Decr decrementa um contador
func (c *cacheService) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// ============================================
// ESTATÍSTICAS
// ============================================

// GetStats retorna estatísticas do cache
func (c *cacheService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"info": info,
		"keys": c.client.DBSize(ctx).Val(),
	}, nil
}
