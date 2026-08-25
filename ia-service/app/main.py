import logging
import time
from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any

import uvicorn
from app.cache_service import cache_service
from app.config import config
from app.db_service import db_service
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from prometheus_fastapi_instrumentator import Instrumentator
from pydantic import BaseModel, Field

# ============================================
# CONFIGURAÇÃO DE LOGGING
# ============================================

logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL.upper()),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# ============================================
# MODELOS PYDANTIC
# ============================================


class OrcamentoRequest(BaseModel):
    cliente_id: int = Field(..., description="ID do cliente")
    loja_id: int = Field(..., description="ID da loja")
    mensagem: str = Field(..., description="Mensagem do cliente", min_length=1)
    historico: list[str] | None = Field(default=[], description="Histórico da conversa")
    produtos: list[str] | None = Field(
        default=[], description="Lista de produtos específicos"
    )


class OrcamentoResponse(BaseModel):
    orcamento: dict[str, Any] | None = None
    precisa_humano: bool = False
    motivo: str | None = None
    resposta_ia: str
    produtos: list[dict[str, Any]] | None = None
    tempo_processamento: float = 0.0


# ============================================
# SERVIÇO IA
# ============================================


class IAService:
    def __init__(self):
        self.limite_orcamento_auto = config.LIMITE_ORCAMENTO_AUTO
        self.start_time = time.time()
        self.requests_count = 0
        self.errors_count = 0
        self.cache_hits = 0
        self.cache_misses = 0
        logger.info(
            f"IA Service iniciado. Limite auto: R$ {self.limite_orcamento_auto:.2f}"
        )

    def get_uptime(self) -> float:
        return time.time() - self.start_time

    def classificar_intencao(self, mensagem: str) -> dict[str, Any]:
        """Classifica a intenção do cliente"""
        mensagem_lower = mensagem.lower()

        padroes = {
            "orcamento": [
                "quanto custa",
                "preço",
                "valor",
                "orçamento",
                "precisa de receita",
                "custa",
                "qual o valor",
                "precinho",
                "quanto está",
                "precifica",
            ],
            "pedido": [
                "quero comprar",
                "pedir",
                "encomendar",
                "gostaria de",
                "queria",
                "comprar",
                "adquirir",
                "reservar",
                "encomenda",
                "quero",
            ],
            "duvida": [
                "como funciona",
                "o que é",
                "para que serve",
                "como tomar",
                "indicação",
                "contra indicação",
                "efeitos colaterais",
            ],
            "entrega": [
                "entrega",
                "frete",
                "prazo",
                "chegada",
                "demora",
                "envio",
                "transportadora",
                "retirada",
            ],
            "receita": [
                "receita",
                "prescrição",
                "médico",
                "receitar",
                "receitado",
                "precisa de receita",
                "com receita",
                "sem receita",
            ],
        }

        for categoria, palavras in padroes.items():
            if any(p in mensagem_lower for p in palavras):
                confianca = 0.9 if len(palavras) > 3 else 0.7
                return {"categoria": categoria, "confianca": confianca}

        return {"categoria": "geral", "confianca": 0.4}

    def extrair_produtos(
        self, mensagem: str, produtos_especificos: list[str] | None = None
    ) -> list[dict[str, Any]]:
        """Extrai nomes de produtos da mensagem"""
        produtos_db = {
            "dipirona": {
                "id": 1,
                "nome": "Dipirona Sódica 500mg",
                "preco": 15.90,
                "categoria": "Analgésico",
            },
            "paracetamol": {
                "id": 2,
                "nome": "Paracetamol 750mg",
                "preco": 12.50,
                "categoria": "Analgésico",
            },
            "amoxicilina": {
                "id": 3,
                "nome": "Amoxicilina 500mg",
                "preco": 45.00,
                "categoria": "Antibiótico",
            },
            "losartana": {
                "id": 4,
                "nome": "Losartana Potássica 50mg",
                "preco": 38.90,
                "categoria": "Cardiovascular",
            },
            "metformina": {
                "id": 5,
                "nome": "Metformina 850mg",
                "preco": 22.30,
                "categoria": "Diabetes",
            },
        }

        if produtos_especificos:
            return [
                produtos_db.get(p) for p in produtos_especificos if p in produtos_db
            ]

        produtos_encontrados = []
        mensagem_lower = mensagem.lower()

        for key, produto in produtos_db.items():
            if key in mensagem_lower or produto["nome"].lower() in mensagem_lower:
                produtos_encontrados.append(produto)

        return produtos_encontrados

    def calcular_orcamento(
        self, produtos: list[dict[str, Any]]
    ) -> dict[str, Any] | None:
        """Calcula orçamento com base nos produtos"""
        if not produtos:
            return None

        itens = []
        total = 0.0

        for p in produtos:
            quantidade = p.get("quantidade", 1)
            subtotal = p["preco"] * quantidade
            itens.append(
                {
                    "id": p.get("id"),
                    "nome": p["nome"],
                    "preco": p["preco"],
                    "quantidade": quantidade,
                    "subtotal": subtotal,
                    "categoria": p.get("categoria", ""),
                }
            )
            total += subtotal

        return {
            "itens": itens,
            "total": total,
            "total_formatado": f"R$ {total:.2f}",
            "quantidade": len(produtos),
            "precisa_receita": any(
                p.get("categoria") == "Antibiótico" for p in produtos
            ),
            "produtos_unicos": len(set(p["nome"] for p in produtos)),
        }

    def gerar_resposta(
        self,
        mensagem: str,
        orcamento: dict[str, Any] | None,
        classificacao: dict[str, Any],
    ) -> str:
        """Gera resposta baseada no contexto"""
        if not orcamento:
            return self._gerar_resposta_sem_produto(mensagem, classificacao)

        if orcamento["precisa_receita"]:
            return self._gerar_resposta_com_receita(orcamento)

        return self._gerar_resposta_padrao(orcamento)

    def _gerar_resposta_sem_produto(
        self, mensagem: str, classificacao: dict[str, Any]
    ) -> str:
        """Gera resposta quando nenhum produto foi identificado"""
        if classificacao["categoria"] == "pedido":
            return "Entendi que você deseja fazer um pedido. Poderia me informar qual produto você gostaria de comprar? 🛒"

        if classificacao["categoria"] == "duvida":
            return "Claro! Vou ajudar com sua dúvida. Poderia me informar melhor qual produto você está se referindo? 🤔"

        if classificacao["categoria"] == "entrega":
            return "Sobre a entrega, podemos verificar o prazo após você escolher os produtos. Qual produto você está interessado? 📦"

        return "Desculpe, não entendi sobre qual produto você gostaria de informação. Poderia me informar o nome do medicamento? 💊"

    def _gerar_resposta_com_receita(self, orcamento: dict[str, Any]) -> str:
        """Gera resposta para produtos que precisam de receita"""
        itens_texto = self._formatar_itens(orcamento["itens"])

        return f"""📋 *Orçamento para {orcamento["quantidade"]} produto(s)*

{itens_texto}

💰 *Total: {orcamento["total_formatado"]}*

⚠️ *ATENÇÃO:* Este medicamento requer receita médica.
Um de nossos atendentes vai entrar em contato para dar continuidade.

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*
📝 *Para agilizar, tenha sua receita em mãos.*"""

    def _gerar_resposta_padrao(self, orcamento: dict[str, Any]) -> str:
        """Gera resposta padrão para orçamento"""
        itens_texto = self._formatar_itens(orcamento["itens"])

        return f"""📋 *Orçamento para {orcamento["quantidade"]} produto(s)*

{itens_texto}

💰 *Total: {orcamento["total_formatado"]}*

✅ *Todos os produtos estão disponíveis em estoque.*

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*
🛒 Deseja que eu reserve os produtos para você?

💡 *Dica:* Se quiser finalizar a compra, um atendente vai te ajudar!"""

    def _formatar_itens(self, itens: list[dict[str, Any]]) -> str:
        """Formata lista de itens para resposta"""
        return "\n".join(
            [
                f"• {item['nome']} - R$ {item['preco']:.2f} (x{item['quantidade']}) = R$ {item['subtotal']:.2f}"
                for item in itens
            ]
        )

    async def processar_mensagem(self, request: OrcamentoRequest) -> OrcamentoResponse:
        """Processa a mensagem e retorna resposta"""
        start_time = time.time()

        try:
            # Verificar cache
            cache_key = f"orcamento:{request.cliente_id}:{request.loja_id}:{hash(request.mensagem)}"
            cached = await cache_service.get(cache_key)

            if cached:
                self.cache_hits += 1
                logger.info(f"Cache hit para cliente {request.cliente_id}")
                return OrcamentoResponse(**cached)

            self.cache_misses += 1

            # Classificar intenção
            classificacao = self.classificar_intencao(request.mensagem)

            # Extrair produtos
            produtos = self.extrair_produtos(request.mensagem, request.produtos)

            # Calcular orçamento
            orcamento = self.calcular_orcamento(produtos)

            # Decidir se precisa de humano
            precisa_humano, motivo = self._decidir_intervencao_humana(
                classificacao, orcamento, request.mensagem
            )

            # Gerar resposta
            resposta = self.gerar_resposta(request.mensagem, orcamento, classificacao)

            # Atualizar métricas
            self.requests_count += 1

            response = OrcamentoResponse(
                orcamento=orcamento,
                precisa_humano=precisa_humano,
                motivo=motivo,
                resposta_ia=resposta,
                produtos=produtos if produtos else None,
                tempo_processamento=time.time() - start_time,
            )

            # Salvar no cache
            await cache_service.set(cache_key, response.dict(), 300)

            return response

        except Exception as e:
            self.errors_count += 1
            logger.error(f"Erro ao processar mensagem: {e!s}")
            raise

    def _decidir_intervencao_humana(
        self,
        classificacao: dict[str, Any],
        orcamento: dict[str, Any] | None,
        mensagem: str,
    ) -> tuple:
        """Decide se é necessária intervenção humana"""
        precisa_humano = False
        motivo = None

        if classificacao["categoria"] == "pedido":
            precisa_humano = True
            motivo = "Pedido de compra requer atendimento humano"

        elif orcamento and orcamento["precisa_receita"]:
            precisa_humano = True
            motivo = "Produto requer receita médica"

        elif orcamento and orcamento["total"] > self.limite_orcamento_auto:
            precisa_humano = True
            motivo = f"Orçamento acima do limite automático (R$ {self.limite_orcamento_auto:.2f})"

        elif classificacao["confianca"] < 0.6:
            precisa_humano = True
            motivo = "Baixa confiança na classificação da mensagem"

        elif "urgência" in mensagem.lower() or "urgente" in mensagem.lower():
            precisa_humano = True
            motivo = "Cliente indicou urgência no atendimento"

        return precisa_humano, motivo


ia_service = IAService()

# ============================================
# LIFESPAN
# ============================================


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("🚀 IA Service iniciando...")
    logger.info(
        f"💰 Limite orçamento automático: R$ {ia_service.limite_orcamento_auto:.2f}"
    )

    # Testar Redis
    try:
        await cache_service.set("health_check", {"status": "ok"}, 10)
        logger.info("✅ Redis conectado com sucesso")
    except Exception as e:
        logger.warning(f"⚠️ Redis não disponível: {e}")

    # Testar banco
    try:
        from sqlalchemy import text

        async with db_service.async_session() as session:
            await session.execute(text("SELECT 1"))
        logger.info("✅ Banco de dados conectado com sucesso")
    except Exception as e:
        logger.warning(f"⚠️ Banco de dados não disponível: {e}")

    yield

    # Shutdown
    logger.info("🛑 IA Service desligando...")
    logger.info(
        f"📊 Estatísticas: {ia_service.requests_count} requisições, {ia_service.errors_count} erros"
    )


# ============================================
# FASTAPI APP
# ============================================

app = FastAPI(
    title="SaaS IA Service",
    description="Serviço de IA para orçamentos e atendimento com cache Redis",
    version="2.0.0",
    lifespan=lifespan,
)

# ============================================
# MIDDLEWARES
# ============================================

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.middleware("http")
async def log_requests(request: Request, call_next):
    start_time = time.time()
    response = await call_next(request)
    process_time = time.time() - start_time
    logger.info(
        f"Request: {request.method} {request.url.path} - "
        f"Status: {response.status_code} - "
        f"Time: {process_time:.3f}s"
    )
    response.headers["X-Process-Time"] = str(process_time)
    return response


# ============================================
# ENDPOINTS
# ============================================


@app.get("/")
async def root():
    return {
        "service": "IA Service",
        "version": "2.0.0",
        "status": "online",
        "docs": "/docs",
        "health": "/api/ia/health",
    }


@app.get("/api/ia/health")
async def health_check():
    redis_status = "ok"
    try:
        await cache_service.set("health_check", {"status": "ok"}, 10)
    except Exception as e:
        redis_status = f"error: {e!s}"

    db_status = "ok"
    try:
        from sqlalchemy import text

        async with db_service.async_session() as session:
            await session.execute(text("SELECT 1"))
    except Exception as e:
        db_status = f"error: {e!s}"

    return {
        "status": "healthy"
        if redis_status == "ok" and db_status == "ok"
        else "degraded",
        "service": "IA Service",
        "version": "2.0.0",
        "timestamp": datetime.now().isoformat(),
        "uptime": ia_service.get_uptime(),
        "redis": redis_status,
        "database": db_status,
    }


@app.get("/api/ia/status")
async def get_status():
    return {
        "status": "online",
        "service": "IA Service",
        "version": "2.0.0",
        "uptime": ia_service.get_uptime(),
        "requests": ia_service.requests_count,
        "errors": ia_service.errors_count,
        "cache_hits": ia_service.cache_hits,
        "cache_misses": ia_service.cache_misses,
        "cache_ratio": ia_service.cache_hits
        / max(ia_service.cache_hits + ia_service.cache_misses, 1)
        * 100,
        "limite_orcamento_auto": ia_service.limite_orcamento_auto,
        "timestamp": datetime.now().isoformat(),
    }


@app.post("/api/ia/orcamento", response_model=OrcamentoResponse)
async def processar_orcamento(request: OrcamentoRequest):
    try:
        logger.info(
            f"Processando orçamento: cliente={request.cliente_id}, loja={request.loja_id}"
        )
        response = await ia_service.processar_mensagem(request)

        if response.precisa_humano:
            logger.info(f"🔄 Intervenção humana necessária: {response.motivo}")

        return response

    except ValueError as e:
        logger.error(f"Erro de validação: {e!s}")
        raise HTTPException(status_code=400, detail=str(e))

    except Exception as e:
        logger.error(f"Erro ao processar orçamento: {e!s}")
        raise HTTPException(
            status_code=500, detail="Erro interno ao processar orçamento"
        )


@app.get("/api/ia/produtos")
async def listar_produtos(loja_id: int = 1, limit: int = 20):
    try:
        produtos = await db_service.get_produtos_by_nome("", loja_id, limit)
        return {"produtos": produtos, "total": len(produtos)}
    except Exception as e:
        logger.error(f"Erro ao listar produtos: {e!s}")
        return {"produtos": [], "total": 0, "error": str(e)}


@app.get("/api/ia/produtos/search")
async def buscar_produtos(q: str, loja_id: int = 1, limit: int = 10):
    try:
        produtos = await db_service.get_produtos_by_nome(q, loja_id, limit)
        return {"query": q, "total": len(produtos), "produtos": produtos}
    except Exception as e:
        logger.error(f"Erro ao buscar produtos: {e!s}")
        return {"query": q, "total": 0, "produtos": [], "error": str(e)}


@app.get("/api/ia/cache")
async def get_cache_info():
    return {
        "cache_service": "redis",
        "stats": {
            "hits": ia_service.cache_hits,
            "misses": ia_service.cache_misses,
            "ratio": ia_service.cache_hits
            / max(ia_service.cache_hits + ia_service.cache_misses, 1)
            * 100,
        },
    }


@app.delete("/api/ia/cache")
async def clear_cache(pattern: str | None = None):
    if pattern:
        deleted = await cache_service.delete_pattern(f"*{pattern}*")
        return {"message": f"Cache limpo para padrão: {pattern}", "deleted": deleted}
    else:
        await cache_service.delete_pattern("*")
        return {"message": "Cache limpo completamente"}


# ============================================
# PROMETHEUS METRICS
# ============================================

try:
    instrumentator = Instrumentator()
    instrumentator.instrument(app).expose(app, include_in_schema=False)
    logger.info("✅ Prometheus metrics enabled")
except Exception as e:
    logger.warning(f"⚠️ Prometheus metrics disabled: {e}")

# ============================================
# EXECUÇÃO
# ============================================

if __name__ == "__main__":
    uvicorn.run(
        "app.main:app",
        host=config.IA_HOST,
        port=config.IA_PORT,
        log_level=config.LOG_LEVEL.lower(),
        workers=2,
    )
