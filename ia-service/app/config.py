import os

from dotenv import load_dotenv

load_dotenv()


class Config:
    # Servidor
    IA_PORT: int = int(os.getenv("IA_PORT", "8001"))
    IA_HOST: str = os.getenv("IA_HOST", "0.0.0.0")
    DEBUG: bool = os.getenv("DEBUG", "false").lower() == "true"

    # Banco de Dados
    DB_HOST: str = os.getenv("DB_HOST", "mariadb")
    DB_PORT: int = int(os.getenv("DB_PORT", "3306"))
    DB_USER: str = os.getenv("DB_USER", "saas_user")
    DB_PASSWORD: str = os.getenv("DB_PASSWORD", "User@2024#Secure!")
    DB_NAME: str = os.getenv("DB_NAME", "saas_atendimento")
    DB_POOL_SIZE: int = int(os.getenv("DB_POOL_SIZE", "10"))
    DB_MAX_OVERFLOW: int = int(os.getenv("DB_MAX_OVERFLOW", "20"))

    # Redis
    REDIS_HOST: str = os.getenv("REDIS_HOST", "redis")
    REDIS_PORT: int = int(os.getenv("REDIS_PORT", "6379"))
    REDIS_PASSWORD: str | None = os.getenv("REDIS_PASSWORD", None)
    REDIS_DB: int = int(os.getenv("REDIS_DB", "0"))
    CACHE_TTL: int = int(os.getenv("CACHE_TTL", "300"))

    # IA
    LIMITE_ORCAMENTO_AUTO: float = float(os.getenv("LIMITE_ORCAMENTO_AUTO", "500.00"))
    MODEL_PATH: str = os.getenv("MODEL_PATH", "/models/llama-2-7b.Q4_K_M.gguf")
    IA_TIMEOUT: int = int(os.getenv("IA_TIMEOUT", "30"))

    # Logging
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "info")
    LOG_FORMAT: str = os.getenv("LOG_FORMAT", "json")

    @classmethod
    def get_database_url(cls) -> str:
        return f"mysql+aiomysql://{cls.DB_USER}:{cls.DB_PASSWORD}@{cls.DB_HOST}:{cls.DB_PORT}/{cls.DB_NAME}?charset=utf8mb4"

    @classmethod
    def get_redis_url(cls) -> str:
        if cls.REDIS_PASSWORD:
            return f"redis://:{cls.REDIS_PASSWORD}@{cls.REDIS_HOST}:{cls.REDIS_PORT}/{cls.REDIS_DB}"
        return f"redis://{cls.REDIS_HOST}:{cls.REDIS_PORT}/{cls.REDIS_DB}"


config = Config()
