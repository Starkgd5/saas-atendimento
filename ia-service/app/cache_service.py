import json
import logging
from typing import Any

import redis
from app.config import config

logger = logging.getLogger(__name__)


class CacheService:
    def __init__(self):
        self.redis_host = config.REDIS_HOST
        self.redis_port = config.REDIS_PORT
        self.redis_password = config.REDIS_PASSWORD
        self.redis_db = config.REDIS_DB
        self.ttl = config.CACHE_TTL

        logger.info(f"Conectando ao Redis: {self.redis_host}:{self.redis_port}")

        # Configurar cliente Redis com senha
        redis_kwargs = {
            "host": self.redis_host,
            "port": self.redis_port,
            "db": self.redis_db,
            "decode_responses": True,
            "socket_connect_timeout": 5,
            "socket_timeout": 5,
        }

        if self.redis_password:
            redis_kwargs["password"] = self.redis_password
            logger.info("🔐 Usando autenticação com senha")
        else:
            logger.info("🔓 Sem autenticação")

        self.client = redis.Redis(**redis_kwargs)

        try:
            self.client.ping()
            logger.info("✅ Redis conectado com sucesso")
        except Exception as e:
            logger.error(f"❌ Erro ao conectar ao Redis: {e}")
            raise

    async def get(self, key: str) -> Any | None:
        try:
            data = self.client.get(key)
            if data:
                return json.loads(data)
            return None
        except Exception as e:
            logger.error(f"Erro ao buscar cache: {e}")
            return None

    async def set(self, key: str, value: Any, ttl: int | None = None) -> bool:
        try:
            ttl = ttl or self.ttl
            self.client.setex(key, ttl, json.dumps(value))
            return True
        except Exception as e:
            logger.error(f"Erro ao salvar cache: {e}")
            return False

    async def delete(self, key: str) -> bool:
        try:
            self.client.delete(key)
            return True
        except Exception as e:
            logger.error(f"Erro ao deletar cache: {e}")
            return False

    async def delete_pattern(self, pattern: str) -> int:
        try:
            keys = self.client.keys(pattern)
            if keys:
                return self.client.delete(*keys)
            return 0
        except Exception as e:
            logger.error(f"Erro ao deletar cache por padrão: {e}")
            return 0

    async def exists(self, key: str) -> bool:
        try:
            return bool(self.client.exists(key))
        except Exception as e:
            logger.error(f"Erro ao verificar existência no cache: {e}")
            return False

    async def get_or_set(self, key: str, callback, ttl: int | None = None):
        cached = await self.get(key)
        if cached is not None:
            return cached

        result = await callback()
        if result:
            await self.set(key, result, ttl)
        return result


cache_service = CacheService()
