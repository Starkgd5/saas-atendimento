#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🔍 Verificando conexões do banco...${NC}"

# Verificar conexões ativas
docker exec saas-mariadb-prod mysql -uroot -p${DB_ROOT_PASSWORD} -e "
SHOW PROCESSLIST;
" 2>/dev/null

# Verificar variáveis de timeout
docker exec saas-mariadb-prod mysql -uroot -p${DB_ROOT_PASSWORD} -e "
SHOW VARIABLES LIKE 'wait_timeout';
SHOW VARIABLES LIKE 'interactive_timeout';
SHOW VARIABLES LIKE 'max_connections';
SHOW VARIABLES LIKE 'connect_timeout';
" 2>/dev/null

# Verificar logs de erro
docker exec saas-mariadb-prod tail -20 /var/lib/mysql/*.err 2>/dev/null || echo "Log de erro não encontrado"

echo -e "${GREEN}✅ Verificação concluída${NC}"