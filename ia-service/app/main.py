import json
import re
from datetime import datetime
from typing import Optional

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

app = FastAPI(title="IA Service - Orçamentos")


class OrcamentoRequest(BaseModel):
    cliente_id: int
    loja_id: int
    mensagem: str
    historico: Optional[list] = []


class OrcamentoResponse(BaseModel):
    orcamento: Optional[dict]
    precisa_humano: bool
    motivo: Optional[str]
    resposta_ia: str


# Base de dados de produtos (simulada)
PRODUTOS = {
    "dipirona": {"nome": "Dipirona Sódica", "preco": 15.90, "categoria": "Analgésico"},
    "paracetamol": {"nome": "Paracetamol 500mg", "preco": 12.50, "categoria": "Analgésico"},
    "amoxicilina": {"nome": "Amoxicilina 500mg", "preco": 45.00, "categoria": "Antibiótico"},
    "losartana": {"nome": "Losartana Potássica", "preco": 38.90, "categoria": "Anti-hipertensivo"},
    "metformina": {"nome": "Metformina 850mg", "preco": 22.30, "categoria": "Antidiabético"},
}


class IAService:
    def __init__(self):
        self.limite_orcamento_auto = 500.00
    
    def classificar_intencao(self, mensagem: str) -> dict:
        """Classifica a intenção do cliente"""
        mensagem_lower = mensagem.lower()
        
        # Padrões de busca
        padroes = {
            "orcamento": ["quanto custa", "preço", "valor", "orçamento", "precisa de receita"],
            "pedido": ["quero comprar", "pedir", "encomendar", "gostaria de"],
            "duvida": ["como funciona", "o que é", "para que serve", "como tomar"],
            "entrega": ["entrega", "frete", "prazo", "chegada"],
        }
        
        for categoria, palavras in padroes.items():
            if any(p in mensagem_lower for p in palavras):
                return {"categoria": categoria, "confianca": 0.8}
        
        return {"categoria": "geral", "confianca": 0.5}
    
    def extrair_produtos(self, mensagem: str) -> list:
        """Extrai nomes de produtos da mensagem"""
        produtos_encontrados = []
        mensagem_lower = mensagem.lower()
        
        for key, produto in PRODUTOS.items():
            if key in mensagem_lower or produto["nome"].lower() in mensagem_lower:
                produtos_encontrados.append(produto)
        
        return produtos_encontrados
    
    def calcular_orcamento(self, produtos: list) -> dict:
        """Calcula orçamento com base nos produtos"""
        if not produtos:
            return None
        
        total = sum(p["preco"] for p in produtos)
        
        return {
            "itens": produtos,
            "total": total,
            "total_formatado": f"R$ {total:.2f}",
            "quantidade": len(produtos),
            "precisa_receita": any(p["categoria"] == "Antibiótico" for p in produtos),
        }
    
    def gerar_resposta(self, mensagem: str, orcamento: dict, classificacao: dict) -> str:
        """Gera resposta baseada no contexto"""
        if not orcamento:
            return "Desculpe, não entendi sobre qual produto você gostaria de orçamento. Poderia me informar o nome do medicamento?"
        
        if orcamento["precisa_receita"]:
            return f"""📋 *Orçamento para {len(orcamento['itens'])} produto(s)*

{self._formatar_itens(orcamento['itens'])}

💰 *Total: {orcamento['total_formatado']}*

⚠️ *ATENÇÃO:* Este medicamento requer receita médica.
Um de nossos atendentes vai entrar em contato para dar continuidade.

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*"""
        
        return f"""📋 *Orçamento para {len(orcamento['itens'])} produto(s)*

{self._formatar_itens(orcamento['itens'])}

💰 *Total: {orcamento['total_formatado']}*

✅ *Todos os produtos estão disponíveis em estoque.*

🔹 *Este orçamento foi gerado automaticamente e está sujeito à confirmação.*
🛒 Deseja que eu reserve os produtos para você?"""
    
    def _formatar_itens(self, itens: list) -> str:
        """Formata lista de itens para resposta"""
        return "\n".join([f"• {p['nome']} - {p['preco']:.2f}" for p in itens])
    
    def processar_mensagem(self, request: OrcamentoRequest) -> OrcamentoResponse:
        """Processa a mensagem e retorna resposta"""
        # Classificar intenção
        classificacao = self.classificar_intencao(request.mensagem)
        
        # Extrair produtos
        produtos = self.extrair_produtos(request.mensagem)
        
        # Calcular orçamento
        orcamento = self.calcular_orcamento(produtos)
        
        # Decidir se precisa de humano
        precisa_humano = False
        motivo = None
        
        if classificacao["categoria"] == "pedido":
            # Pedidos sempre precisam de humano
            precisa_humano = True
            motivo = "Pedido de compra requer atendimento humano"
        
        elif orcamento and orcamento["precisa_receita"]:
            precisa_humano = True
            motivo = "Produto requer receita médica"
        
        elif orcamento and orcamento["total"] > self.limite_orcamento_auto:
            precisa_humano = True
            motivo = f"""Orçamento acima do limite automático
             (R$ {self.limite_orcamento_auto:.2f})"""
        
        elif classificacao["confianca"] < 0.6:
            precisa_humano = True
            motivo = "Baixa confiança na classificação da mensagem"
        
        # Gerar resposta
        resposta = self.gerar_resposta(request.mensagem, orcamento, classificacao)
        
        return OrcamentoResponse(
            orcamento=orcamento,
            precisa_humano=precisa_humano,
            motivo=motivo,
            resposta_ia=resposta
        )


ia_service = IAService()


@app.post("/api/ia/orcamento")
async def processar_orcamento(request: OrcamentoRequest):
    try:
        response = ia_service.processar_mensagem(request)
        return response
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/api/ia/health")
async def health_check():
    return {"status": "ok", "service": "IA", "timestamp": datetime.now().isoformat()}


@app.get("/api/ia/produtos")
async def listar_produtos():
    return {"produtos": PRODUTOS}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8001)