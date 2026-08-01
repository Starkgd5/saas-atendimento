#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}🔍 Verificando saúde dos serviços...${NC}"

# Verificar cada serviço
services=("backend-go" "ia-service" "frontend" "mariadb" "redis" "prometheus" "grafana")

for service in "${services[@]}"; do
    if docker ps --filter "name=saas-$service-prod" --filter "status=running" | grep -q .; then
        echo -e "${GREEN}✅ $service está rodando${NC}"
    else
        echo -e "${RED}❌ $service não está rodando!${NC}"
        docker logs "saas-$service-prod" --tail 20
        exit 1
    fi
done

# Testar API
echo -e "${YELLOW}🧪 Testando API...${NC}"
if curl -s -f https://${DOMAIN}/health > /dev/null; then
    echo -e "${GREEN}✅ API está respondendo${NC}"
else
    echo -e "${RED}❌ API não respondeu${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Todos os serviços estão saudáveis!${NC}"