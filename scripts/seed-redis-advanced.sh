#!/bin/bash

echo "🚀 Populando Redis com filas para todas as lojas..."

# Lista de lojas (1 a 16, pulando 13)
LOJAS="1 2 3 4 5 6 7 8 9 10 11 12 14 15 16"

# Limpar Redis
docker exec saas-redis redis-cli FLUSHDB

echo "📦 Criando filas para cada loja..."

for LOJA in $LOJAS; do
    echo "  Loja $LOJA..."
    
    # Limite de atendimentos (3-5 por loja)
    LIMITE=$((RANDOM % 3 + 3))
    docker exec saas-redis redis-cli SET "fila:limite:$LOJA" "$LIMITE"
    
    # Adicionar clientes na fila de espera (2-5 clientes)
    ESPERA=$((RANDOM % 4 + 2))
    for i in $(seq 1 $ESPERA); do
        CLIENTE_ID=$((LOJA * 100 + i))
        TIMESTAMP=$(date +%s%N)
        docker exec saas-redis redis-cli ZADD "fila:espera:$LOJA" $TIMESTAMP $CLIENTE_ID
        
        # Informações do cliente
        INFO="{\"cliente_id\":$CLIENTE_ID,\"loja_id\":$LOJA,\"entrada\":$(date +%s),\"status\":\"na_fila\"}"
        docker exec saas-redis redis-cli SET "cliente:info:$CLIENTE_ID" "$INFO"
    done
    
    # Adicionar clientes em atendimento (1-3 clientes)
    ATENDIMENTO=$((RANDOM % 3 + 1))
    for i in $(seq 1 $ATENDIMENTO); do
        CLIENTE_ID=$((LOJA * 100 + 50 + i))
        docker exec saas-redis redis-cli SADD "fila:atendimento:$LOJA" $CLIENTE_ID
        
        INFO="{\"cliente_id\":$CLIENTE_ID,\"loja_id\":$LOJA,\"entrada\":$(date +%s),\"status\":\"em_atendimento\"}"
        docker exec saas-redis redis-cli SET "cliente:info:$CLIENTE_ID" "$INFO"
    done
    
    # Estatísticas da fila
    STATS="{\"em_atendimento\":$ATENDIMENTO,\"em_espera\":$ESPERA,\"limite\":$LIMITE,\"atualizado_em\":$(date +%s)}"
    docker exec saas-redis redis-cli SET "fila:stats:$LOJA" "$STATS"
    
    # Chave global para compatibilidade
    docker exec saas-redis redis-cli SADD "fila:atendimento" $((LOJA * 100 + 50 + 1))
done

echo "✅ Redis populado com sucesso!"
echo ""
echo "📊 Resumo:"
for LOJA in $LOJAS; do
    echo "  Loja $LOJA:"
    echo "    Em atendimento: $(docker exec saas-redis redis-cli SCARD fila:atendimento:$LOJA)"
    echo "    Em espera: $(docker exec saas-redis redis-cli ZCARD fila:espera:$LOJA)"
    echo "    Limite: $(docker exec saas-redis redis-cli GET fila:limite:$LOJA)"
done