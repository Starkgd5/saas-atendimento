.PHONY: help build up down logs clean test deploy backup restore health

help:
	@echo "Comandos disponíveis:"
	@echo "  make build      - Buildar todos os containers"
	@echo "  make up         - Subir serviços em desenvolvimento"
	@echo "  make down       - Parar serviços"
	@echo "  make logs       - Ver logs em tempo real"
	@echo "  make test       - Rodar testes"
	@echo "  make clean      - Limpar volumes e cache"
	@echo "  make deploy     - Deploy em produção"
	@echo "  make backup     - Fazer backup do banco"
	@echo "  make restore    - Restaurar backup do banco"
	@echo "  make health     - Verificar saúde dos serviços"
	@echo "  make scale      - Escalar serviços"
	@echo "  make metrics    - Ver métricas"

build:
	@echo "🏗️  Buildando containers..."
	docker-compose build --no-cache

up:
	@echo "🚀 Subindo serviços..."
	docker-compose up -d
	@echo "✅ Serviços rodando:"
	@echo "   - Backend:  http://localhost:8080"
	@echo "   - Frontend: http://localhost:3000"
	@echo "   - Adminer:  http://localhost:8081"

down:
	@echo "🛑 Parando serviços..."
	docker-compose down

logs:
	docker-compose logs -f

test:
	@echo "🧪 Rodando testes..."
	cd backend-go && go test -v -race ./...
	cd frontend-react && npm test -- --watchAll=false

clean:
	@echo "🧹 Limpando tudo..."
	docker-compose down -v	docker system prune -f

deploy:
	@echo "🚀 Deploy em produção..."
	chmod +x scripts/*.sh
	./scripts/deploy.sh

backup:
	@echo "📦 Backup do banco..."
	./scripts/backup.sh

restore:
	@echo "🔄 Restaurando backup..."
	@read -p "Digite o nome do arquivo de backup: " file; \
	docker exec -i saas-mariadb-prod mysql -u$${DB_USER} -p$${DB_PASSWORD} $${DB_NAME} < ./backups/$$file

health:
	./scripts/health-check.sh

scale:
	@echo "📈 Escalando serviços..."
	@read -p "Número de replicas (backend): " replicas; \
	docker-compose -f docker-compose.prod.yml up -d --scale backend-go=$$replicas

metrics:
	@echo "📊 Métricas dos serviços:"
	@echo ""
	@echo "=== Containers ==="
	docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Image}}"
	@echo ""
	@echo "=== CPU/Memória ==="
	docker stats --no-stream