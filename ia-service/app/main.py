import json
import logging
import os
import re
import time
from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any, Dict, List, Optional

import uvicorn
from fastapi import FastAPI, HTTPException, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from prometheus_fastapi_instrumentator import Instrumentator
from pydantic import BaseModel, Field

# ============================================
# CONFIGURAÇÃO
# ============================================

LOG_LEVEL = os.getenv("LOG_LEVEL", "info")
LOG_FORMAT = os.getenv("LOG_FORMAT", "json")
LIMITE_ORCAMENTO_AUTO = float(os.getenv("LIMITE_ORCAMENTO_AUTO", "500.00"))
MODEL_PATH = os.getenv("MODEL_PATH", "/models/llama-2-7b.Q4_K_M.gguf")

# ============================================
# LOGGING
# ============================================

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL.upper()),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# ============================================
# MODELOS
# ============================================


class OrcamentoRequest(BaseModel):
    cliente_id: int = Field(..., description="ID do cliente")
    loja_id: int = Field(..., description="ID da loja")
    mensagem: str = Field(..., description="Mensagem do cliente", min_length=1)
    historico: Optional[List[str]] = Field(default=[], description="Histórico da conversa")
    produtos: Optional[List[str]] = Field(default=[], description="Lista de produtos específicos")


class OrcamentoResponse(BaseModel):
    orcamento: Optional[Dict[str, Any]] = None
    precisa_humano: bool = False
    motivo: Optional[str] = None
    resposta_ia: str
    produtos: Optional[List[Dict[str, Any]]] = None
    tempo_processamento: float = 0.0


class Produto(BaseModel):
    id: Optional[int] = None
    nome: str
    preco: float
    categoria: str
    quantidade: Optional[int] = 1


class IAStatusResponse(BaseModel):
    status: str
    service: str
    version: str
    produtos_disponiveis: int
    limite_orcamento_auto: float
    timestamp: str
    uptime: float


class HealthResponse(BaseModel):
    status: str
    service: str
    timestamp: str
    uptime: float
    version: str

# ============================================
# BASE DE DADOS DE PRODUTOS
# ============================================


PRODUTOS = {
    "dipirona": {
        "id": 1,
        "nome": "Dipirona Sódica 500mg",
        "preco": 15.90,
        "categoria": "Analgésico",
        "descricao": "Analgésico e antitérmico"
    },
    "paracetamol": {
        "id": 2,
        "nome": "Paracetamol 750mg",
        "preco": 12.50,
        "categoria": "Analgésico",
        "descricao": "Analgésico e antitérmico"
    },
    "amoxicilina": {
        "id": 3,
        "nome": "Amoxicilina 500mg",
        "preco": 45.00,
        "categoria": "Antibiótico",
        "descricao": "Antibiótico de amplo espectro"
    },
    "losartana": {
        "id": 4,
        "nome": "Losartana Potássica 50mg",
        "preco": 38.90,
        "categoria": "Cardiovascular",
        "descricao": "Anti-hipertensivo"
    },
    "metformina": {
        "id": 5,
        "nome": "Metformina 850mg",
        "preco": 22.30,
        "categoria": "Diabetes",
        "descricao": "Antidiabético"
    },
    "omeprazol": {
        "id": 6,
        "nome": "Omeprazol 20mg",
        "preco": 28.50,
        "categoria": "Gastrointestinal",
        "descricao": "Inibidor de bomba de prótons"
    },
    "ibuprofeno": {
        "id": 7,
        "nome": "Ibuprofeno 600mg",
        "preco": 18.90,
        "categoria": "Anti-inflamatório",
        "descricao": "Anti-inflamatório não esteroidal"
    },
    "azitromicina": {
        "id": 8,
        "nome": "Azitromicina 500mg",
        "preco": 65.00,
        "categoria": "Antibiótico",
        "descricao": "Antibiótico macrolídeo"
    },
    "vitamina_c": {
        "id": 9,
        "nome": "Vitamina C 1g",
        "preco": 35.00,
        "categoria": "Suplemento",
        "descricao": "Suplemento vitamínico"
    },
    "sertralina": {
        "id": 10,
        "nome": "Cloridrato de Sertralina 50mg",
        "preco": 55.00,
        "categoria": "Saúde Mental",
        "descricao": "Antidepressivo"
    }
}

# ============================================
# SERVIÇO IA
# ============================================


class IAService:
    def __init__(self):
        self.limite_orcamento_auto = LIMITE_ORCAMENTO_AUTO
        self.produtos = PRODUTOS
        self.start_time = time.time()
        self.requests_count = 0
        self.errors_count = 0
        logger.info(f"IA Service iniciado. Limite auto: R$ {self.limite_orcamento_auto:.2f}")
        logger.info(f"Produtos disponíveis: {len(self.produtos)}")

    def get_uptime(self) -> float:
        return time.time() - self.start_time

    def classificar_intencao(self, mensagem: str) -> Dict[str, Any]:
        """Classifica a intenção do cliente"""
        mensagem_lower = mensagem.lower()

        padroes = {
            "orcamento": [
                "quanto custa", "preço", "valor", "orçamento", "precisa de receita",
                "custa", "qual o valor", "precinho", "quanto está", "precifica"
            ],
            "pedido": [
                "quero comprar", "pedir", "encomendar", "gostaria de", "queria",
                "comprar", "adquirir", "reservar", "encomenda"
            ],
            "duvida": [
                "como funciona", "o que é", "para que serve", "como tomar",
                "indicação", "contra indicação", "efeitos colaterais"
            ],
            "entrega": [
                "entrega", "frete", "prazo", "chegada", "demora", "envio",
                "transportadora", "retirada", "entrega em domicílio"
            ],
            "receita": [
                "receita", "prescrição", "médico", "receitar", "receitado",
                "precisa de receita", "com receita", "sem receita"
            ]
        }

        for categoria, palavras in padroes.items():
            if any(p in mensagem_lower for p in palavras):
                confianca = 0.9 if len(palavras) > 3 else 0.7
                return {"categoria": categoria, "confianca": confianca}

        return {"categoria": "geral", "confianca": 0.4}

    def extrair_produtos(self, mensagem: str, produtos_especificos: Optional[List[str]] = None) -> List[Dict[str, Any]]:
        """Extrai nomes de produtos da mensagem"""
        if produtos_especificos:
            return [self.produtos.get(p) for p in produtos_especificos if p in self.produtos]

        produtos_encontrados = []
        mensagem_lower = mensagem.lower()

        for key, produto in self.produtos.items():
            if key in mensagem_lower or produto["nome"].lower() in mensagem_lower:
                produtos_encontrados.append(produto)

        return produtos_encontrados

    def calcular_orcamento(self, produtos: List[Dict[str, Any]]) -> Optional[Dict[str, Any]]:
        """Calcula orçamento com base nos produtos"""
        if not produtos:
            return None

        itens = []
        total = 0.0

        for p in produtos:
            quantidade = p.get("quantidade", 1)
            subtotal = p["preco"] * quantidade
            itens.append({
                "nome": p["nome"],
                "preco": p["preco"],
                "quantidade": quantidade,
                "subtotal": subtotal,
                "categoria": p["categoria"]
            })
            total += subtotal

        return {
            "itens": itens,
            "total": total,
            "total_formatado": f"R$ {total:.2f}",
            "quantidade": len(produtos),
            "precisa_receita": any(p["categoria"] == "Antibiótico" for p in produtos),
            "produtos_unicos": len(set(p["nome"] for p in produtos))
        }

    def gerar_resposta(self, mensagem: str, orcamento: Optional[Dict[str, Any]],
                       classificacao: Dict[str, Any]) -> str:
        """Gera resposta baseada no contexto"""
        if not orcamento:
            return self._gerar_resposta_sem_produto(mensagem, classificacao)

        if orcamento["precisa_receita"]:
            return self._gerar_resposta_com_receita(orcamento)

        return self._gerar_resposta_padrao(orcamento)

    def _gerar_resposta_sem_produto(self, mensagem: str, classificacao: Dict[str, Any]) -> str:
        """Gera resposta quando nenhum produto foi identificado"""
        if classificacao["categoria"] == "pedido":
            return "Entendi que você deseja fazer um pedido. Poderia me informar qual produto você gostaria de comprar? 🛒"

        if classificacao["categoria"] == "duvida":
            return "Claro! Vou ajudar com sua dúvida. Poderia me informar melhor qual produto você está se referindo? 🤔"

        if classificacao["categoria"] == "entrega":
            return "Sobre a entrega, podemos verificar o prazo após você escolher os produtos. Qual produto você está interessado? 📦"

        return "Desculpe, não entendi sobre qual produto você gostaria de informação. Poderia me informar o nome do medicamento? 💊"

    def _gerar_resposta_com_receita(self, orcamento: Dict[str, Any]) -> str:
        """Gera resposta para produtos que precisam de receita"""
        itens_texto = self._formatar_itens(orcamento["itens"])

        return f"""📋 *Orçamento para {orcamento['quantidade']} produto(s)*

{itens_texto}

💰 *Total: {orcamento['total_formatado']}*

⚠️ *ATENÇÃO:* Este medicamento requer receita médica.
Um de nossos atendentes vai entrar em contato para dar continuidade.

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*
📝 *Para agilizar, tenha sua receita em mãos.*"""

    def _gerar_resposta_padrao(self, orcamento: Dict[str, Any]) -> str:
        """Gera resposta padrão para orçamento"""
        itens_texto = self._formatar_itens(orcamento["itens"])

        return f"""📋 *Orçamento para {orcamento['quantidade']} produto(s)*

{itens_texto}

💰 *Total: {orcamento['total_formatado']}*

✅ *Todos os produtos estão disponíveis em estoque.*

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*
🛒 Deseja que eu reserve os produtos para você?

💡 *Dica:* Se quiser finalizar a compra, um atendente vai te ajudar!"""

    def _formatar_itens(self, itens: List[Dict[str, Any]]) -> str:
        """Formata lista de itens para resposta"""
        return "\n".join([
            f"• {item['nome']} - R$ {item['preco']:.2f} (x{item['quantidade']}) = R$ {item['subtotal']:.2f}"
            for item in itens
        ])

    def processar_mensagem(self, request: OrcamentoRequest) -> OrcamentoResponse:
        """Processa a mensagem e retorna resposta"""
        start_time = time.time()

        try:
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

            return OrcamentoResponse(
                orcamento=orcamento,
                precisa_humano=precisa_humano,
                motivo=motivo,
                resposta_ia=resposta,
                produtos=produtos if produtos else None,
                tempo_processamento=time.time() - start_time
            )

        except Exception as e:
            self.errors_count += 1
            logger.error(f"Erro ao processar mensagem: {str(e)}")
            raise

    def _decidir_intervencao_humana(self, classificacao: Dict[str, Any],
                                   orcamento: Optional[Dict[str, Any]],
                                   mensagem: str) -> tuple:
        """Decide se é necessária intervenção humana"""
        precisa_humano = False
        motivo = None

        # Casos que precisam de humano
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

    def get_status(self) -> IAStatusResponse:
        """Retorna o status do serviço"""
        return IAStatusResponse(
            status="online",
            service="IA Service",
            version="2.0.0",
            produtos_disponiveis=len(self.produtos),
            limite_orcamento_auto=self.limite_orcamento_auto,
            timestamp=datetime.now().isoformat(),
            uptime=self.get_uptime()
        )

    def get_produtos(self) -> Dict[str, Any]:
        """Retorna a lista de produtos"""
        return {"produtos": self.produtos}

# ============================================
# INSTÂNCIA DO SERVIÇO
# ============================================


ia_service = IAService()

# ============================================
# LIFESPAN
# ============================================


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("🚀 IA Service iniciando...")
    logger.info(f"📦 Produtos disponíveis: {len(ia_service.produtos)}")
    logger.info(f"💰 Limite orçamento automático: R$ {ia_service.limite_orcamento_auto:.2f}")
    yield
    # Shutdown
    logger.info("🛑 IA Service desligando...")
    logger.info(f"📊 Estatísticas finais: {ia_service.requests_count} requisições, {ia_service.errors_count} erros")

# ============================================
# FASTAPI APP
# ============================================

app = FastAPI(
    title="SaaS IA Service",
    description="Serviço de IA para orçamentos e atendimento",
    version="2.0.0",
    lifespan=lifespan
)

# ============================================
# MIDDLEWARES
# ============================================

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# Logging middleware
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


# Error handler
@app.exception_handler(Exception)
async def generic_exception_handler(request: Request, exc: Exception):
    logger.error(f"Erro não tratado: {str(exc)}")
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={"error": "Erro interno do servidor", "detail": str(exc)}
    )

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
        "health": "/api/ia/health"
    }


@app.get("/api/ia/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        status="healthy",
        service="IA Service",
        timestamp=datetime.now().isoformat(),
        uptime=ia_service.get_uptime(),
        version="2.0.0"
    )


@app.get("/api/ia/status", response_model=IAStatusResponse)
async def get_status():
    """Retorna o status detalhado do serviço"""
    return ia_service.get_status()


@app.get("/api/ia/produtos")
async def listar_produtos():
    """Lista todos os produtos disponíveis"""
    return ia_service.get_produtos()


@app.get("/api/ia/produtos/search")
async def buscar_produtos(q: Optional[str] = None):
    """Busca produtos pelo nome"""
    if not q:
        return {"produtos": []}

    q_lower = q.lower()
    resultados = [
        p for p in ia_service.produtos.values()
        if q_lower in p["nome"].lower() or q_lower in p["categoria"].lower()
    ]

    return {
        "query": q,
        "total": len(resultados),
        "produtos": resultados
    }


@app.post("/api/ia/orcamento", response_model=OrcamentoResponse)
async def processar_orcamento(request: OrcamentoRequest):
    """Processa uma requisição de orçamento"""
    try:
        logger.info(f"Processando orçamento: cliente={request.cliente_id}, loja={request.loja_id}")
        response = ia_service.processar_mensagem(request)

        if response.precisa_humano:
            logger.info(f"🔄 Intervenção humana necessária: {response.motivo}")

        return response

    except ValueError as e:
        logger.error(f"Erro de validação: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))

    except Exception as e:
        logger.error(f"Erro ao processar orçamento: {str(e)}")
        raise HTTPException(status_code=500, detail="Erro interno ao processar orçamento")


@app.get("/api/ia/metrics")
async def get_metrics():
    """Retorna métricas do serviço"""
    return {
        "total_requests": ia_service.requests_count,
        "total_errors": ia_service.errors_count,
        "error_rate": (ia_service.errors_count / max(ia_service.requests_count, 1)) * 100,
        "uptime": ia_service.get_uptime(),
        "produtos_disponiveis": len(ia_service.produtos)
    }

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
# MAIN
# ============================================

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=8001,
        log_level=LOG_LEVEL.lower(),
        workers=2
    )
