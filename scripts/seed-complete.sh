#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}🚀 POPULANDO SISTEMA COMPLETO${NC}"
echo -e "${BLUE}========================================${NC}"

# Carregar variáveis
export $(grep -v '^#' .env | xargs)

# Verificar containers
echo -e "${YELLOW}🔍 Verificando containers...${NC}"

if ! docker ps --filter "name=saas-mariadb" --filter "status=running" | grep -q .; then
    echo -e "${RED}❌ MariaDB não está rodando!${NC}"
    echo -e "${YELLOW}🔄 Iniciando...${NC}"
    docker-compose up -d mariadb
    sleep 15
fi

if ! docker ps --filter "name=saas-redis" --filter "status=running" | grep -q .; then
    echo -e "${RED}❌ Redis não está rodando!${NC}"
    echo -e "${YELLOW}🔄 Iniciando...${NC}"
    docker-compose up -d redis
    sleep 5
fi

echo -e "${GREEN}✅ Containers OK${NC}"

# 1. Popular banco
echo -e "${YELLOW}📦 Populando banco de dados...${NC}"
docker exec -i saas-mariadb mysql -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} < scripts/seed-full-db.sql 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Banco populado com sucesso!${NC}"
else
    echo -e "${YELLOW}⚠️ Tentando com root...${NC}"
    docker exec -i saas-mariadb mysql -uroot -p${DB_ROOT_PASSWORD} ${DB_NAME} < scripts/seed-full-db.sql
fi

# 2. Popular Redis
echo -e "${YELLOW}📦 Populando Redis...${NC}"
chmod +x scripts/seed-redis-advanced.sh
./scripts/seed-redis-advanced.sh

# 3. Mostrar resumo
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ POPULAÇÃO CONCLUÍDA!${NC}"
echo -e "${BLUE}========================================${NC}"

echo -e "${YELLOW}📊 Resumo do banco:${NC}"
docker exec saas-mariadb mysql -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} -e "
SELECT 
    'Lojas' as Tabela, COUNT(*) as Total FROM lojas UNION ALL
SELECT 'Funcionários', COUNT(*) FROM usuarios UNION ALL
SELECT 'Clientes', COUNT(*) FROM clientes UNION ALL
SELECT 'Produtos', COUNT(*) FROM produtos UNION ALL
SELECT 'Lotes', COUNT(*) FROM lotes UNION ALL
SELECT 'Atendimentos', COUNT(*) FROM atendimentos UNION ALL
SELECT 'Mensagens', COUNT(*) FROM mensagens UNION ALL
SELECT 'Vendas', COUNT(*) FROM vendas UNION ALL
SELECT 'Itens Venda', COUNT(*) FROM itens_venda UNION ALL
SELECT 'Orçamentos', COUNT(*) FROM orcamentos UNION ALL
SELECT 'Reclamações', COUNT(*) FROM reclamacoes UNION ALL
SELECT 'Movimentos', COUNT(*) FROM movimentos_estoque;
" 2>/dev/null

echo -e "${YELLOW}📊 Faturamento por loja:${NC}"
docker exec saas-mariadb mysql -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} -e "
SELECT 
    l.codigo,
    l.nome,
    COALESCE(SUM(v.total), 0) as faturamento,
    COUNT(DISTINCT v.id) as vendas
FROM lojas l
LEFT JOIN vendas v ON l.id = v.loja_id
GROUP BY l.id
ORDER BY faturamento DESC;
" 2>/dev/null

echo -e "${YELLOW}📊 Filas no Redis:${NC}"
for LOJA in 1 2 3 4 5 6 7 8 9 10 11 12 14 15 16; do
    ATEND=$(docker exec saas-redis redis-cli SCARD "fila:atendimento:$LOJA" 2>/dev/null || echo 0)
    ESPERA=$(docker exec saas-redis redis-cli ZCARD "fila:espera:$LOJA" 2>/dev/null || echo 0)
    LIMITE=$(docker exec saas-redis redis-cli GET "fila:limite:$LOJA" 2>/dev/null || echo 3)
    echo "  Loja $LOJA: Atendimento: $ATEND, Espera: $ESPERA, Limite: $LIMITE"
done

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}📍 Acesse: http://localhost:3000${NC}"
echo -e "${GREEN}👤 Login: admin@saas.com / admin123${NC}"
echo -e "${BLUE}========================================${NC}"