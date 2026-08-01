#!/bin/bash

BACKUP_DIR="./backups"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

# Criar diretório de backup
mkdir -p $BACKUP_DIR

# Backup do banco
echo "📦 Backup do banco de dados..."
docker exec saas-mariadb-prod mysqldump -u${DB_USER} -p${DB_PASSWORD} ${DB_NAME} > ${BACKUP_DIR}/backup_${DATE}.sql

# Backup das configurações
echo "📦 Backup das configurações..."
cp .env.production ${BACKUP_DIR}/env_${DATE}.backup
cp docker-compose.prod.yml ${BACKUP_DIR}/docker-compose_${DATE}.backup

# Compactar
gzip ${BACKUP_DIR}/backup_${DATE}.sql

# Remover backups antigos
echo "🧹 Removendo backups com mais de ${RETENTION_DAYS} dias..."
find ${BACKUP_DIR} -name "*.sql.gz" -mtime +${RETENTION_DAYS} -delete
find ${BACKUP_DIR} -name "*.backup" -mtime +${RETENTION_DAYS} -delete

echo "✅ Backup concluído: backup_${DATE}.sql.gz"