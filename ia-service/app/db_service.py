import logging

from app.config import config
from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine
from sqlalchemy.orm import sessionmaker

logger = logging.getLogger(__name__)


class DatabaseService:
    def __init__(self):
        # Usar o método get_database_url() em vez de DATABASE_URL
        database_url = config.get_database_url()
        logger.info(
            f"Conectando ao banco: {config.DB_HOST}:{config.DB_PORT}/{config.DB_NAME}"
        )

        self.engine = create_async_engine(
            database_url,
            pool_size=config.DB_POOL_SIZE,
            max_overflow=config.DB_MAX_OVERFLOW,
            pool_pre_ping=True,
            pool_recycle=3600,
            echo=False,
        )
        self.async_session = sessionmaker(
            self.engine,
            class_=AsyncSession,
            expire_on_commit=False,
            autocommit=False,
            autoflush=False,
        )

    async def get_session(self) -> AsyncSession:
        return self.async_session()

    async def get_produto_by_id(self, produto_id: int, loja_id: int) -> dict:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    SELECT id, nome, descricao, categoria, preco_venda, 
                           requere_receita, tarja, ativo
                    FROM produtos 
                    WHERE id = :produto_id AND loja_id = :loja_id
                """),
                {"produto_id": produto_id, "loja_id": loja_id},
            )
            row = result.fetchone()
            if row:
                return {
                    "id": row[0],
                    "nome": row[1],
                    "descricao": row[2],
                    "categoria": row[3],
                    "preco": row[4],
                    "requere_receita": row[5],
                    "tarja": row[6],
                    "ativo": row[7],
                }
            return None

    async def get_produtos_by_nome(
        self, nome: str, loja_id: int, limit: int = 10
    ) -> list:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    SELECT id, nome, descricao, categoria, preco_venda, 
                           requere_receita, tarja
                    FROM produtos 
                    WHERE loja_id = :loja_id 
                    AND nome LIKE :nome 
                    AND ativo = 1
                    LIMIT :limit
                """),
                {"loja_id": loja_id, "nome": f"%{nome}%", "limit": limit},
            )
            rows = result.fetchall()
            return [
                {
                    "id": row[0],
                    "nome": row[1],
                    "descricao": row[2],
                    "categoria": row[3],
                    "preco": row[4],
                    "requere_receita": row[5],
                    "tarja": row[6],
                }
                for row in rows
            ]

    async def get_estoque_total(self, produto_id: int, loja_id: int) -> int:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    SELECT COALESCE(SUM(quantidade), 0)
                    FROM lotes
                    WHERE produto_id = :produto_id 
                    AND loja_id = :loja_id 
                    AND status = 'Ativo'
                    AND data_validade >= CURDATE()
                """),
                {"produto_id": produto_id, "loja_id": loja_id},
            )
            return result.scalar() or 0

    async def get_cliente_by_id(self, cliente_id: int) -> dict:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    SELECT id, nome, telefone, email, cpf
                    FROM clientes 
                    WHERE id = :cliente_id AND ativo = 1
                """),
                {"cliente_id": cliente_id},
            )
            row = result.fetchone()
            if row:
                return {
                    "id": row[0],
                    "nome": row[1],
                    "telefone": row[2],
                    "email": row[3],
                    "cpf": row[4],
                }
            return None

    async def criar_orcamento(self, dados: dict) -> int:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    INSERT INTO orcamentos 
                    (cliente_id, loja_id, total, desconto, total_com_desconto, 
                     observacao, status, created_at)
                    VALUES 
                    (:cliente_id, :loja_id, :total, :desconto, :total_com_desconto,
                     :observacao, 'pendente', NOW())
                """),
                dados,
            )
            await session.commit()
            return result.lastrowid

    async def criar_atendimento(self, dados: dict) -> int:
        async with self.async_session() as session:
            result = await session.execute(
                text("""
                    INSERT INTO atendimentos 
                    (cliente_id, loja_id, status, iniciado_em)
                    VALUES 
                    (:cliente_id, :loja_id, 'aguardando', NOW())
                """),
                dados,
            )
            await session.commit()
            return result.lastrowid


db_service = DatabaseService()
