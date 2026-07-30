#!/bin/bash
set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Iniciando deploy em produção...${NC}"

# Verificar variáveis de ambiente
if [ ! -f .env.prod ]; then
    echo -e "${RED}❌ Arquivo .env.prod não encontrado!${NC}"
    exit 1
fi

# Carregar variáveis
export $(cat .env.prod | grep -v '#' | xargs)

# Backup do banco
echo -e "${YELLOW}📦 Realizando backup do banco...${NC}"
docker exec saas-mariadb-prod mysqldump -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} > backups/backup_$(date +%Y%m%d_%H%M%S).sql

# Build das imagens
echo -e "${YELLOW}🏗️  Buildando imagens...${NC}"
docker-compose -f docker-compose.prod.yml build

# Parar serviços antigos
echo -e "${YELLOW}🛑 Parando serviços antigos...${NC}"
docker-compose -f docker-compose.prod.yml down

# Subir novos serviços
echo -e "${YELLOW}🚀 Subindo novos serviços...${NC}"
docker-compose -f docker-compose.prod.yml up -d

# Aguardar inicialização
echo -e "${YELLOW}⏳ Aguardando serviços iniciarem...${NC}"
sleep 10

# Verificar saúde
echo -e "${YELLOW}🔍 Verificando saúde dos serviços...${NC}"
for service in backend-go ia-service; do
    if docker ps --filter "name=saas-$service-prod" --filter "status=running" | grep -q .; then
        echo -e "${GREEN}✅ $service está rodando${NC}"
    else
        echo -e "${RED}❌ $service não está rodando!${NC}"
        docker logs saas-$service-prod --tail 50
        exit 1
    fi
done

# Testar API
echo -e "${YELLOW}🧪 Testando API...${NC}"
curl -s http://localhost:8080/health | grep -q "ok" && echo -e "${GREEN}✅ API está respondendo${NC}" || echo -e "${RED}❌ API não respondeu${NC}"

echo -e "${GREEN}✅ Deploy concluído com sucesso!${NC}"
echo -e "${GREEN}📍 Acesse: ${BASE_URL}${NC}"