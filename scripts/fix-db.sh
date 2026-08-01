#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}🔧 Corrigindo configurações do banco${NC}"
echo -e "${YELLOW}========================================${NC}"

# Carregar variáveis
export $(grep -v '^#' .env | xargs)

echo -e "${GREEN}📝 Credenciais:${NC}"
echo "DB_USER: ${DB_USER}"
echo "DB_PASSWORD: ${DB_PASSWORD}"
echo "DB_NAME: ${DB_NAME}"
echo "DB_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}"

# 1. Parar serviços
echo -e "${YELLOW}🛑 Parando serviços...${NC}"
docker-compose stop backend-go ia-service 2>/dev/null || true

# 2. Verificar se o banco existe
if docker ps --filter "name=saas-mariadb" --filter "status=running" | grep -q .; then
    echo -e "${GREEN}✅ MariaDB está rodando${NC}"
else
    echo -e "${YELLOW}⚠️ MariaDB não está rodando. Iniciando...${NC}"
    docker-compose up -d mariadb
    sleep 10
fi

# 3. Testar acesso root
echo -e "${YELLOW}🔍 Testando acesso root...${NC}"
if docker exec saas-mariadb mysql -uroot -p${DB_ROOT_PASSWORD} -e "SELECT 1" 2>/dev/null; then
    echo -e "${GREEN}✅ Acesso root OK${NC}"
else
    echo -e "${RED}❌ Falha no acesso root!${NC}"
    echo -e "${YELLOW}🔄 Recriando MariaDB...${NC}"
    docker-compose stop mariadb
    docker-compose rm -f mariadb
    docker volume rm -f saas-mariadb 2>/dev/null || true
    docker-compose up -d mariadb
    sleep 15
fi

# 4. Criar usuário se não existir
echo -e "${YELLOW}🔍 Verificando usuário ${DB_USER}...${NC}"
if docker exec saas-mariadb mysql -u${DB_USER} -p${DB_PASSWORD} -e "SELECT 1" 2>/dev/null; then
    echo -e "${GREEN}✅ Usuário ${DB_USER} existe${NC}"
else
    echo -e "${YELLOW}⚠️ Criando usuário ${DB_USER}...${NC}"
    docker exec saas-mariadb mysql -uroot -p${DB_ROOT_PASSWORD} -e "
    CREATE USER IF NOT EXISTS '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASSWORD}';
    CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASSWORD}';
    CREATE DATABASE IF NOT EXISTS ${DB_NAME};
    GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';
    GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';
    FLUSH PRIVILEGES;
    "
    echo -e "${GREEN}✅ Usuário criado com sucesso!${NC}"
fi

# 5. Subir backend
echo -e "${YELLOW}🚀 Subindo backend...${NC}"
docker-compose up -d backend-go ia-service

sleep 10

# 6. Verificar saúde
echo -e "${YELLOW}🔍 Verificando saúde do backend...${NC}"
if curl -s -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Backend saudável!${NC}"
else
    echo -e "${RED}❌ Backend não respondeu. Verificando logs...${NC}"
    docker logs saas-backend --tail 30
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ Correção concluída!${NC}"
echo -e "${GREEN}📍 Acesse: http://localhost:8080/health${NC}"
echo -e "${GREEN}========================================${NC}"