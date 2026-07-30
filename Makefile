.PHONY: help build up down logs clean test restart status shell-backend shell-db

help:
	@echo "Comandos disponíveis:"
	@echo "  make build   - Buildar todos os containers"
	@echo "  make up      - Subir todos os serviços"
	@echo "  make down    - Parar todos os serviços"
	@echo "  make logs    - Ver logs em tempo real"
	@echo "  make test    - Rodar testes"
	@echo "  make clean   - Limpar volumes e cache"
	@echo "  make restart - Reiniciar serviços"
	@echo "  make status  - Ver status dos containers"

build:
	@echo "🏗️  Buildando containers..."
	docker-compose build --no-cache

up:
	@echo "🚀 Subindo serviços..."
	docker-compose up -d
	@echo ""
	@echo "✅ Serviços rodando:"
	@echo "   - Backend:  http://localhost:8080"
	@echo "   - Frontend: http://localhost:3000"
	@echo "   - Adminer:  http://localhost:8081"
	@echo ""
	@echo "📝 Para ver logs: make logs"

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
	docker-compose down -v
	docker system prune -f

restart: down up

status:
	docker-compose ps

shell-backend:
	docker exec -it saas-backend sh

shell-db:
	docker exec -it saas-mariadb mysql -u root -proot123

test-integration:
	@echo "🧪 Rodando testes de integração..."
	docker-compose up -d mariadb redis
	sleep 5
	cd backend-go && go test -v -race -tags=integration ./...
	docker-compose down
