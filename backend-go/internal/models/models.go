package models

import (
	"time"
)

// Loja representa uma farmácia
type Loja struct {
	ID        int       `json:"id"`
	Codigo    string    `json:"codigo"`
	Nome      string    `json:"nome"`
	CreatedAt time.Time `json:"created_at"`
}

// Cliente representa um cliente da farmácia
type Cliente struct {
	ID               int       `json:"id"`
	LojaID           int       `json:"loja_id"`
	Nome             string    `json:"nome"`
	Telefone         string    `json:"telefone"`
	Email            string    `json:"email,omitempty"`
	UltimoAtendimento *time.Time `json:"ultimo_atendimento,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// Atendimento representa uma sessão de atendimento
type Atendimento struct {
	ID          int       `json:"id"`
	ClienteID   int       `json:"cliente_id"`
	LojaID      int       `json:"loja_id"`
	Status      string    `json:"status"` // aguardando, em_atendimento, finalizado
	IniciadoEm  time.Time `json:"iniciado_em"`
	FinalizadoEm *time.Time `json:"finalizado_em,omitempty"`
	AtendenteID *int      `json:"atendente_id,omitempty"`
	Cliente     *Cliente  `json:"cliente,omitempty"`
}

// Mensagem representa uma mensagem trocada no chat
type Mensagem struct {
	ID           int       `json:"id"`
	AtendimentoID int      `json:"atendimento_id"`
	Remetente    string    `json:"remetente"` // cliente, atendente, ia
	Conteudo     string    `json:"conteudo"`
	Tipo         string    `json:"tipo"` // texto, imagem, documento
	ArquivoURL   string    `json:"arquivo_url,omitempty"`
	EnviadoEm    time.Time `json:"enviado_em"`
}

// Usuario representa um atendente/administrador
type Usuario struct {
	ID        int       `json:"id"`
	LojaID    *int      `json:"loja_id,omitempty"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	Role      string    `json:"role"` // admin, gerente, atendente
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
}

// FilaStatus representa o status atual da fila
type FilaStatus struct {
	EmAtendimento int `json:"em_atendimento"`
	EmEspera      int `json:"em_espera"`
	Limite        int `json:"limite"`
}