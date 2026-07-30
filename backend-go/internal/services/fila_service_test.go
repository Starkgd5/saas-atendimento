package services

import (
	"context"
	"testing"
	// "time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func TestFilaService_AdicionarClienteFila(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	service := NewFilaService(redisClient, 3)

	// Adicionar cliente
	err := service.AdicionarClienteFila(ctx, 1, 1)
	assert.NoError(t, err)

	// Verificar status
	status, err := service.GetStatusFila(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, status.EmAtendimento)
	assert.Equal(t, 0, status.EmEspera)
}

func TestFilaService_LimiteAtendimentos(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	service := NewFilaService(redisClient, 2) // Limite = 2

	// Adicionar 3 clientes
	for i := 1; i <= 3; i++ {
		err := service.AdicionarClienteFila(ctx, i, 1)
		assert.NoError(t, err)
	}

	// Verificar status
	status, err := service.GetStatusFila(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, status.EmAtendimento) // 2 em atendimento
	assert.Equal(t, 1, status.EmEspera)      // 1 na espera
}

func TestFilaService_ProximoCliente(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	service := NewFilaService(redisClient, 3)

	// Adicionar cliente
	err := service.AdicionarClienteFila(ctx, 1, 1)
	assert.NoError(t, err)

	// Puxar próximo
	clienteID, err := service.ProximoClienteAtendimento(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, clienteID)

	// Verificar status
	status, err := service.GetStatusFila(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, status.EmAtendimento)
}