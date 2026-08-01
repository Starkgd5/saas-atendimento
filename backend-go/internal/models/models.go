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

// Cliente representa um cliente da farmácia
type Cliente struct {
	ID                int        `json:"id"`
	LojaID            int        `json:"loja_id"`
	Nome              string     `json:"nome"`
	Telefone          string     `json:"telefone"`
	Email             string     `json:"email,omitempty"`
	UltimoAtendimento *time.Time `json:"ultimo_atendimento,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
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
	ID          int     `json:"id"`
	Nome        string  `json:"nome"`
	Descricao   string  `json:"descricao,omitempty"`
	Categoria   string  `json:"categoria"`
	Preco       float64 `json:"preco"`
	PrecoCusto  float64 `json:"preco_custo,omitempty"`
	Estoque     int     `json:"estoque"`
	EstoqueMin  int     `json:"estoque_min"`
	LojaID      int     `json:"loja_id"`
	RequereReceita bool `json:"requere_receita"`
	Ativo       bool    `json:"ativo"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
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