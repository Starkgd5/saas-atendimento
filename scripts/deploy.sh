#!/bin/bash
set -e

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}🚀 SaaS Atendimento - Deploy${NC}"
echo -e "${BLUE}========================================${NC}"

# Verificar .env.production
if [ ! -f .env.production ]; then
    echo -e "${RED}❌ Arquivo .env.production não encontrado!${NC}"
    exit 1
fi

# Carregar variáveis
export $(cat .env.production | grep -v '^#' | xargs)

# Backup do banco
echo -e "${YELLOW}📦 Realizando backup do banco...${NC}"
./scripts/backup.sh

# Build das imagens
echo -e "${YELLOW}🏗️  Buildando imagens...${NC}"
docker-compose -f docker-compose.prod.yml build --parallel

# Parar serviços antigos
echo -e "${YELLOW}🛑 Parando serviços antigos...${NC}"
docker-compose -f docker-compose.prod.yml down

# Subir novos serviços
echo -e "${YELLOW}🚀 Subindo novos serviços...${NC}"
docker-compose -f docker-compose.prod.yml up -d

# Aguardar inicialização
echo -e "${YELLOW}⏳ Aguardando serviços iniciarem...${NC}"
sleep 15

# Verificar saúde
echo -e "${YELLOW}🔍 Verificando saúde dos serviços...${NC}"
./scripts/health-check.sh

# Mostrar status
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ Deploy concluído com sucesso!${NC}"
echo -e "${GREEN}📍 Acesse: https://${DOMAIN}${NC}"
echo -e "${GREEN}📊 Grafana: https://${DOMAIN}/grafana${NC}"
echo -e "${GREEN}📈 Prometheus: http://localhost:9090${NC}"
echo -e "${GREEN}========================================${NC}"