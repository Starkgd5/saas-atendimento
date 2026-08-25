from sqlalchemy import Boolean, Column, DateTime, Float, Integer, String, Text
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.sql import func

Base = declarative_base()


class Produto(Base):
    __tablename__ = "produtos"

    id = Column(Integer, primary_key=True, index=True)
    loja_id = Column(Integer, nullable=False)
    codigo_barras = Column(String(50), unique=True)
    nome = Column(String(200), nullable=False)
    descricao = Column(Text)
    categoria = Column(String(50), nullable=False)
    sub_categoria = Column(String(50))
    fabricante = Column(String(100))
    fornecedor_id = Column(Integer)
    registro_anvisa = Column(String(50))
    classe_terapeutica = Column(String(100))
    tarja = Column(String(20), default="Sem Tarja")
    requere_receita = Column(Boolean, default=False)
    tipo_receita = Column(String(10))
    preco_custo = Column(Float, default=0)
    preco_venda = Column(Float, default=0)
    margem_lucro = Column(Float)
    preco_minimo = Column(Float)
    estoque_minimo = Column(Integer, default=5)
    estoque_maximo = Column(Integer, default=100)
    ponto_pedido = Column(Integer, default=10)
    ativo = Column(Boolean, default=True)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())


class Lote(Base):
    __tablename__ = "lotes"

    id = Column(Integer, primary_key=True, index=True)
    produto_id = Column(Integer, nullable=False)
    loja_id = Column(Integer, nullable=False)
    numero_lote = Column(String(50), nullable=False)
    data_fabricacao = Column(DateTime)
    data_validade = Column(DateTime, nullable=False)
    quantidade = Column(Integer, nullable=False)
    quantidade_inicial = Column(Integer, nullable=False)
    preco_custo = Column(Float)
    preco_venda = Column(Float)
    status = Column(String(20), default="Ativo")
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())


class Cliente(Base):
    __tablename__ = "clientes"

    id = Column(Integer, primary_key=True, index=True)
    loja_id = Column(Integer, nullable=False)
    nome = Column(String(100), nullable=False)
    telefone = Column(String(20))
    email = Column(String(100))
    cpf = Column(String(14))
    ativo = Column(Boolean, default=True)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())


class Orcamento(Base):
    __tablename__ = "orcamentos"

    id = Column(Integer, primary_key=True, index=True)
    cliente_id = Column(Integer, nullable=False)
    loja_id = Column(Integer, nullable=False)
    atendimento_id = Column(Integer)
    status = Column(String(20), default="pendente")
    total = Column(Float, default=0)
    desconto = Column(Float, default=0)
    total_com_desconto = Column(Float)
    observacao = Column(Text)
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())
