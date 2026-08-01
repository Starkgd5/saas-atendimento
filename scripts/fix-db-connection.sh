#!/bin/bash

echo "🔧 Reparando conexões com o banco..."

# 1. Reiniciar MariaDB
echo "🔄 Reiniciando MariaDB..."
docker-compose -f docker-compose.prod.yml restart mariadb

sleep 10

# 2. Otimizar tabelas
echo "🔧 Otimizando tabelas..."
docker exec saas-mariadb-prod mysql -uroot -p${DB_ROOT_PASSWORD} -e "
USE ${DB_NAME};
OPTIMIZE TABLE lojas, clientes, atendimentos, mensagens, produtos, orcamentos, reclamacoes, usuarios, configuracoes;
" 2>/dev/null

# 3. Limpar conexões ociosas
echo "🧹 Limpando conexões ociosas..."
docker exec saas-mariadb-prod mysql -uroot -p${DB_ROOT_PASSWORD} -e "
SET GLOBAL wait_timeout = 28800;
SET GLOBAL interactive_timeout = 28800;
SET GLOBAL max_connections = 200;
" 2>/dev/null

# 4. Reiniciar backend
echo "🔄 Reiniciando backend..."
docker-compose -f docker-compose.prod.yml restart backend-go

echo "✅ Reparo concluído!"