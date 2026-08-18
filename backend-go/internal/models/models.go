package models

import (
	"time"
)

// ============================================
// LOJA
// ============================================

// Loja representa uma farmácia
type Loja struct {
	ID        int       `json:"id"`
	Codigo    string    `json:"codigo"`
	Nome      string    `json:"nome"`
	CreatedAt time.Time `json:"created_at"`
}

// ============================================
// CLIENTE
// ============================================

// Cliente representa um cliente da farmácia (expandido)
type Cliente struct {
	ID                int       `json:"id"`
	LojaID            int       `json:"loja_id"`
	Nome              string    `json:"nome"`
	Telefone          string    `json:"telefone"`
	Email             string    `json:"email,omitempty"`
	CPF               string    `json:"cpf,omitempty"`
	DataNascimento    *time.Time `json:"data_nascimento,omitempty"`
	Sexo              string    `json:"sexo,omitempty"`
	Endereco          string    `json:"endereco,omitempty"`
	Numero            string    `json:"numero,omitempty"`
	Complemento       string    `json:"complemento,omitempty"`
	Bairro            string    `json:"bairro,omitempty"`
	Cidade            string    `json:"cidade,omitempty"`
	Estado            string    `json:"estado,omitempty"`
	CEP               string    `json:"cep,omitempty"`
	UltimoAtendimento *time.Time `json:"ultimo_atendimento,omitempty"`
	TotalCompras      float64   `json:"total_compras"`
	QuantidadeCompras int       `json:"quantidade_compras"`
	UltimaCompra      *time.Time `json:"ultima_compra,omitempty"`
	Observacao        string    `json:"observacao,omitempty"`
	Ativo             bool      `json:"ativo"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ClienteFidelidade representa dados de fidelidade
type ClienteFidelidade struct {
	ID              int       `json:"id"`
	ClienteID       int       `json:"cliente_id"`
	LojaID          int       `json:"loja_id"`
	Pontos          int       `json:"pontos"`
	PontosAcumulados int      `json:"pontos_acumulados"`
	PontosUtilizados int      `json:"pontos_utilizados"`
	Nivel           string    `json:"nivel"` // Bronze, Prata, Ouro, Diamante
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Caixa representa uma sessão de caixa
type Caixa struct {
	ID             int       `json:"id"`
	LojaID         int       `json:"loja_id"`
	UsuarioID      int       `json:"usuario_id"`
	DataAbertura   time.Time `json:"data_abertura"`
	DataFechamento *time.Time `json:"data_fechamento,omitempty"`
	SaldoInicial   float64   `json:"saldo_inicial"`
	SaldoFinal     float64   `json:"saldo_final"`
	TotalEntradas  float64   `json:"total_entradas"`
	TotalSaidas    float64   `json:"total_saidas"`
	Status         string    `json:"status"` // Aberto, Fechado, Cancelado
	Observacao     string    `json:"observacao,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MovimentoCaixa representa uma movimentação financeira
type MovimentoCaixa struct {
	ID          int       `json:"id"`
	CaixaID     int       `json:"caixa_id"`
	LojaID      int       `json:"loja_id"`
	Tipo        string    `json:"tipo"` // Entrada, Saída
	Categoria   string    `json:"categoria"` // Venda, Pagamento, Troco, Outro
	Valor       float64   `json:"valor"`
	Descricao   string    `json:"descricao"`
	VendaID     *int      `json:"venda_id,omitempty"`
	UsuarioID   int       `json:"usuario_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================
// ATENDIMENTO
// ============================================

// Atendimento representa uma sessão de atendimento
type Atendimento struct {
	ID           int        `json:"id"`
	ClienteID    int        `json:"cliente_id"`
	LojaID       int        `json:"loja_id"`
	Status       string     `json:"status"` // aguardando, em_atendimento, finalizado, abandonado
	IniciadoEm   time.Time  `json:"iniciado_em"`
	FinalizadoEm *time.Time `json:"finalizado_em,omitempty"`
	AtendenteID  *int       `json:"atendente_id,omitempty"`
	Cliente      *Cliente   `json:"cliente,omitempty"`
}

// ============================================
// MENSAGEM
// ============================================

// Mensagem representa uma mensagem trocada no chat
type Mensagem struct {
	ID            int       `json:"id"`
	AtendimentoID int       `json:"atendimento_id"`
	Remetente     string    `json:"remetente"` // cliente, atendente, ia
	Conteudo      string    `json:"conteudo"`
	Tipo          string    `json:"tipo"` // texto, imagem, documento, audio
	ArquivoURL    string    `json:"arquivo_url,omitempty"`
	ArquivoNome   string    `json:"arquivo_nome,omitempty"`
	ArquivoTamanho int64    `json:"arquivo_tamanho,omitempty"`
	EnviadoEm     time.Time `json:"enviado_em"`
	Lida          bool      `json:"lida"`
}

// ============================================
// USUÁRIO
// ============================================

// Usuario representa um atendente/administrador
type Usuario struct {
	ID        int       `json:"id"`
	LojaID    *int      `json:"loja_id,omitempty"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	SenhaHash string    `json:"-"` // Não serializar
	Role      string    `json:"role"` // admin, gerente, atendente
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ============================================
// FILA
// ============================================

// FilaStatus representa o status atual da fila
type FilaStatus struct {
	EmAtendimento int `json:"em_atendimento"`
	EmEspera      int `json:"em_espera"`
	Limite        int `json:"limite"`
}

// FilaCliente representa um cliente na fila
type FilaCliente struct {
	ClienteID   int       `json:"cliente_id"`
	Nome        string    `json:"nome"`
	Telefone    string    `json:"telefone"`
	Entrada     time.Time `json:"entrada"`
	Posicao     int       `json:"posicao"`
	TempoEspera int       `json:"tempo_espera"` // em segundos
}

// ============================================
// ORÇAMENTO
// ============================================

// Orcamento representa um orçamento gerado
type Orcamento struct {
	ID              int        `json:"id"`
	ClienteID       int        `json:"cliente_id"`
	LojaID          int        `json:"loja_id"`
	AtendimentoID   int        `json:"atendimento_id,omitempty"`
	Status          string     `json:"status"` // pendente, aprovado, rejeitado, expirado
	Total           float64    `json:"total"`
	Desconto        float64    `json:"desconto,omitempty"`
	TotalComDesconto float64   `json:"total_com_desconto,omitempty"`
	Observacao      string     `json:"observacao,omitempty"`
	Itens           []OrcamentoItem `json:"itens,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at,omitempty"`
	ExpiradoEm      *time.Time `json:"expirado_em,omitempty"`
}

// OrcamentoItem representa um item do orçamento
type OrcamentoItem struct {
	ID          int     `json:"id"`
	OrcamentoID int     `json:"orcamento_id"`
	ProdutoID   int     `json:"produto_id"`
	ProdutoNome string  `json:"produto_nome"`
	Quantidade  int     `json:"quantidade"`
	PrecoUnit   float64 `json:"preco_unit"`
	Total       float64 `json:"total"`
}

// ============================================
// PRODUTO
// ============================================

// Produto representa um produto da farmácia
type Produto struct {
	ID             int       `json:"id"`
	LojaID         int       `json:"loja_id"`
	CodigoBarras   string    `json:"codigo_barras"`
	Nome           string    `json:"nome"`
	Descricao      string    `json:"descricao"`
	Categoria      string    `json:"categoria"` // Medicamento, Perfumaria, Conveniência, etc.
	SubCategoria   string    `json:"sub_categoria"`
	Fabricante     string    `json:"fabricante"`
	FornecedorID   int       `json:"fornecedor_id"`
	
	// Controle ANVISA
	RegistroANVISA string    `json:"registro_anvisa"`
	ClasseTerapeutica string `json:"classe_terapeutica"`
	Tarja          string    `json:"tarja"` // Vermelha, Amarela, Preta, Sem Tarja
	RequereReceita bool      `json:"requere_receita"`
	TipoReceita    string    `json:"tipo_receita"` // A1, A2, B1, B2, C1, C2
	
	// Preços
	PrecoCusto     float64   `json:"preco_custo"`
	PrecoVenda     float64   `json:"preco_venda"`
	MargemLucro    float64   `json:"margem_lucro"`
	PrecoMinimo    float64   `json:"preco_minimo"` // Preço mínimo permitido
	
	// Estoque
	EstoqueMinimo  int       `json:"estoque_minimo"`
	EstoqueMaximo  int       `json:"estoque_maximo"`
	PontoPedido    int       `json:"ponto_pedido"` // Quando repor
	
	// Controle
	Ativo          bool      `json:"ativo"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Lote representa um lote de produto
type Lote struct {
	ID            int       `json:"id"`
	ProdutoID     int       `json:"produto_id"`
	LojaID        int       `json:"loja_id"`
	NumeroLote    string    `json:"numero_lote"`
	DataFabricacao time.Time `json:"data_fabricacao"`
	DataValidade  time.Time `json:"data_validade"`
	Quantidade    int       `json:"quantidade"`
	QuantidadeInicial int   `json:"quantidade_inicial"`
	PrecoCusto    float64   `json:"preco_custo"`
	PrecoVenda    float64   `json:"preco_venda"`
	Status        string    `json:"status"` // Ativo, Vencido, Baixado
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// MovimentoEstoque registra todas as movimentações de estoque
type MovimentoEstoque struct {
	ID            int       `json:"id"`
	LojaID        int       `json:"loja_id"`
	ProdutoID     int       `json:"produto_id"`
	LoteID        int       `json:"lote_id"`
	Tipo          string    `json:"tipo"` // Entrada, Saída, Ajuste, Devolução, Perda
	Quantidade    int       `json:"quantidade"`
	SaldoAnterior int       `json:"saldo_anterior"`
	SaldoAtual    int       `json:"saldo_atual"`
	Motivo        string    `json:"motivo"`
	Documento     string    `json:"documento"` // NF, Pedido, etc.
	UsuarioID     int       `json:"usuario_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// Venda representa uma venda realizada
type Venda struct {
	ID            int       `json:"id"`
	LojaID        int       `json:"loja_id"`
	ClienteID     *int      `json:"cliente_id"` // Pode ser nulo para cliente não cadastrado
	AtendenteID   int       `json:"atendente_id"`
	FarmaceuticoID *int     `json:"farmaceutico_id"` // Para medicamentos controlados
	NumeroVenda   string    `json:"numero_venda"`
	TipoPagamento string    `json:"tipo_pagamento"` // Dinheiro, Cartão, Pix
	Status        string    `json:"status"` // Pendente, Pago, Cancelado
	Subtotal      float64   `json:"subtotal"`
	Desconto      float64   `json:"desconto"`
	Total         float64   `json:"total"`
	ReceitaAnexada bool     `json:"receita_anexada"`
	Observacao    string    `json:"observacao"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ItemVenda representa um item de venda
type ItemVenda struct {
	ID          int       `json:"id"`
	VendaID     int       `json:"venda_id"`
	ProdutoID   int       `json:"produto_id"`
	LoteID      int       `json:"lote_id"`
	Quantidade  int       `json:"quantidade"`
	PrecoUnit   float64   `json:"preco_unit"`
	Total       float64   `json:"total"`
}

// ReceitaMedica representa uma receita médica (RDC 527/2021)
type ReceitaMedica struct {
	ID             int       `json:"id"`
	ClienteID      int       `json:"cliente_id"`
	VendaID        *int      `json:"venda_id"`
	NumeroReceita  string    `json:"numero_receita"`
	DataEmissao    time.Time `json:"data_emissao"`
	DataValidade   time.Time `json:"data_validade"`
	MedicoNome     string    `json:"medico_nome"`
	MedicoCRM      string    `json:"medico_crm"`
	MedicoUF       string    `json:"medico_uf"`
	Cid            string    `json:"cid"` // Classificação Internacional de Doenças
	Status         string    `json:"status"` // Pendente, Validada, Rejeitada, Expirada
	DataValidacao  *time.Time `json:"data_validacao"`
	FarmaceuticoID int       `json:"farmaceutico_id"`
	Observacao     string    `json:"observacao"`
	ArquivoURL     string    `json:"arquivo_url"` // URL da imagem da receita
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ItemReceita representa um item de uma receita
type ItemReceita struct {
	ID          int    `json:"id"`
	ReceitaID   int    `json:"receita_id"`
	ProdutoID   int    `json:"produto_id"`
	Quantidade  int    `json:"quantidade"`
	Posologia   string `json:"posologia"`
	Duracao     string `json:"duracao"`
	Observacao  string `json:"observacao"`
}

// Fornecedor representa um fornecedor
type Fornecedor struct {
	ID              int       `json:"id"`
	LojaID          int       `json:"loja_id"`
	RazaoSocial     string    `json:"razao_social"`
	NomeFantasia    string    `json:"nome_fantasia"`
	CNPJ            string    `json:"cnpj"`
	IE              string    `json:"ie"`
	InscricaoMunicipal string `json:"inscricao_municipal"`
	Endereco        string    `json:"endereco"`
	Numero          string    `json:"numero"`
	Complemento     string    `json:"complemento"`
	Bairro          string    `json:"bairro"`
	Cidade          string    `json:"cidade"`
	Estado          string    `json:"estado"`
	CEP             string    `json:"cep"`
	Telefone        string    `json:"telefone"`
	Email           string    `json:"email"`
	ContatoNome     string    `json:"contato_nome"`
	ContatoTelefone string    `json:"contato_telefone"`
	Ativo           bool      `json:"ativo"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Compra representa uma compra de produtos
type Compra struct {
	ID            int       `json:"id"`
	LojaID        int       `json:"loja_id"`
	FornecedorID  int       `json:"fornecedor_id"`
	NumeroNF      string    `json:"numero_nf"`
	DataEmissao   time.Time `json:"data_emissao"`
	DataRecebimento time.Time `json:"data_recebimento"`
	Status        string    `json:"status"` // Em andamento, Recebido, Cancelado
	Subtotal      float64   `json:"subtotal"`
	Desconto      float64   `json:"desconto"`
	Total         float64   `json:"total"`
	Observacao    string    `json:"observacao"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ============================================
// RECLAMAÇÃO
// ============================================

// Reclamacao representa uma reclamação de cliente
type Reclamacao struct {
	ID          int       `json:"id"`
	ClienteID   int       `json:"cliente_id"`
	LojaID      int       `json:"loja_id"`
	AtendimentoID *int    `json:"atendimento_id,omitempty"`
	Mensagem    string    `json:"mensagem"`
	Status      string    `json:"status"` // pendente, em_analise, resolvido, ignorado
	Prioridade  string    `json:"prioridade"` // baixa, media, alta, critica
	Categoria   string    `json:"categoria"` // atendimento, produto, entrega, pagamento, outro
	Resposta    string    `json:"resposta,omitempty"`
	ResolvidoEm *time.Time `json:"resolvido_em,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Cliente     *Cliente  `json:"cliente,omitempty"`
}

// ============================================
// MÉTRICAS
// ============================================

// DashboardMetrics representa as métricas do dashboard
type DashboardMetrics struct {
	TotalClientes      int     `json:"total_clientes"`
	AtendimentosHoje   int     `json:"atendimentos_hoje"`
	AtendimentosMes    int     `json:"atendimentos_mes"`
	TicketMedio        float64 `json:"ticket_medio"`
	TaxaConversao      float64 `json:"taxa_conversao"`
	OrcamentosGerados  int     `json:"orcamentos_gerados"`
	TempoMedioEspera   int     `json:"tempo_medio_espera"` // em segundos
	TempoMedioAtendimento float64 `json:"tempo_medio_atendimento"` // em segundos
	TotalFinalizados   int     `json:"total_finalizados"`
	Abandonos          int     `json:"abandonos"`
	TaxaAbandono       float64 `json:"taxa_abandono"`
	HorarioPico        string  `json:"horario_pico"`
	ProdutosMaisVendidos []ProdutoVendido `json:"produtos_mais_vendidos"`
	ReclamacoesPendentes int `json:"reclamacoes_pendentes"`
	Fila                *FilaStatus `json:"fila,omitempty"`
}

// ProdutoVendido representa um produto com métricas de venda
type ProdutoVendido struct {
	ID         int     `json:"id"`
	Nome       string  `json:"nome"`
	Quantidade int     `json:"quantidade"`
	Total      float64 `json:"total"`
}

// ============================================
// WEBHOOK
// ============================================

// WhatsAppWebhook representa o payload do webhook do WhatsApp
type WhatsAppWebhook struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      *struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
					Document *struct {
						ID       string `json:"id"`
						Filename string `json:"filename"`
						MimeType string `json:"mime_type"`
						URL      string `json:"url,omitempty"`
					} `json:"document,omitempty"`
					Image *struct {
						ID       string `json:"id"`
						Caption  string `json:"caption"`
						MimeType string `json:"mime_type"`
						URL      string `json:"url,omitempty"`
					} `json:"image,omitempty"`
					Audio *struct {
						ID       string `json:"id"`
						MimeType string `json:"mime_type"`
						URL      string `json:"url,omitempty"`
					} `json:"audio,omitempty"`
				} `json:"messages"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// ============================================
// RESPOSTAS DA API
// ============================================

// APIResponse é a estrutura padrão de resposta da API
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// PaginatedResponse é uma resposta paginada
type PaginatedResponse struct {
	Items       interface{} `json:"items"`
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	TotalPages  int         `json:"total_pages"`
	HasNext     bool        `json:"has_next"`
	HasPrevious bool        `json:"has_previous"`
}

// ============================================
// CONSTANTES
// ============================================

// Status de Atendimento
const (
	StatusAguardando    = "aguardando"
	StatusEmAtendimento = "em_atendimento"
	StatusFinalizado    = "finalizado"
	StatusAbandonado    = "abandonado"
)

// Status de Orçamento
const (
	OrcamentoPendente  = "pendente"
	OrcamentoAprovado  = "aprovado"
	OrcamentoRejeitado = "rejeitado"
	OrcamentoExpirado  = "expirado"
)

// Status de Reclamação
const (
	ReclamacaoPendente  = "pendente"
	ReclamacaoEmAnalise = "em_analise"
	ReclamacaoResolvido = "resolvido"
	ReclamacaoIgnorado  = "ignorado"
)

// Prioridade de Reclamação
const (
	PrioridadeBaixa  = "baixa"
	PrioridadeMedia  = "media"
	PrioridadeAlta   = "alta"
	PrioridadeCritica = "critica"
)

// Roles de Usuário
const (
	RoleAdmin     = "admin"
	RoleGerente   = "gerente"
	RoleAtendente = "atendente"
)