package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type IAService struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

// ============================================
// REQUISIÇÕES
// ============================================

type OrcamentoRequest struct {
	ClienteID int      `json:"cliente_id"`
	LojaID    int      `json:"loja_id"`
	Mensagem  string   `json:"mensagem"`
	Historico []string `json:"historico,omitempty"`
	Produtos  []string `json:"produtos,omitempty"` // Lista de produtos para orçamento
}

type OrcamentoResponse struct {
	Orcamento     *OrcamentoData `json:"orcamento,omitempty"`
	PrecisaHumano bool           `json:"precisa_humano"`
	Motivo        string         `json:"motivo,omitempty"`
	RespostaIA    string         `json:"resposta_ia"`
	Produtos      []ProdutoOrcamento `json:"produtos,omitempty"`
}

type OrcamentoData struct {
	Itens          []ProdutoOrcamento `json:"itens"`
	Total          float64            `json:"total"`
	TotalFormatado string             `json:"total_formatado"`
	Quantidade     int                `json:"quantidade"`
	PrecisaReceita bool               `json:"precisa_receita"`
}

type ProdutoOrcamento struct {
	ID         int     `json:"id,omitempty"`
	Nome       string  `json:"nome"`
	Preco      float64 `json:"preco"`
	Categoria  string  `json:"categoria"`
	Quantidade int     `json:"quantidade,omitempty"`
}

// ============================================
// RESPOSTAS
// ============================================

type IAStatusResponse struct {
	Status             string  `json:"status"`
	Service            string  `json:"service"`
	Version            string  `json:"version"`
	ProdutosDisponiveis int    `json:"produtos_disponiveis"`
	LimiteOrcamentoAuto float64 `json:"limite_orcamento_auto"`
	Timestamp          string  `json:"timestamp"`
}

type IAProdutosResponse struct {
	Produtos map[string]ProdutoOrcamento `json:"produtos"`
}

// ============================================
// CONSTRUTOR
// ============================================

func NewIAService(baseURL string) *IAService {
	if baseURL == "" {
		baseURL = "http://ia-service:8001"
	}
	return &IAService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout:    30 * time.Second,
		maxRetries: 3,
	}
}

// ============================================
// MÉTODOS PRINCIPAIS
// ============================================

// ProcessarOrcamento processa um orçamento com a IA
func (s *IAService) ProcessarOrcamento(ctx context.Context, req *OrcamentoRequest) (*OrcamentoResponse, error) {
	url := fmt.Sprintf("%s/api/ia/orcamento", s.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %w", err)
	}

	// Tentar com retry
	var lastErr error
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		if attempt > 0 {
			// Aguardar antes de tentar novamente
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		resp, err := s.doRequest(ctx, "POST", url, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("falha após %d tentativas: %w", s.maxRetries, lastErr)
}

// ListarProdutos retorna a lista de produtos da IA
func (s *IAService) ListarProdutos(ctx context.Context) (*IAProdutosResponse, error) {
	url := fmt.Sprintf("%s/api/ia/produtos", s.baseURL)

	body, err := s.doRequestRaw(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result IAProdutosResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar produtos: %w", err)
	}

	return &result, nil
}

// GetStatus retorna o status do serviço IA
func (s *IAService) GetStatus(ctx context.Context) (*IAStatusResponse, error) {
	url := fmt.Sprintf("%s/api/ia/status", s.baseURL)

	body, err := s.doRequestRaw(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result IAStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar status: %w", err)
	}

	return &result, nil
}

// HealthCheck verifica se o serviço IA está saudável
func (s *IAService) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/ia/health", s.baseURL)

	_, err := s.doRequestRaw(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("serviço IA indisponível: %w", err)
	}

	return nil
}

// ============================================
// MÉTODOS AUXILIARES
// ============================================

// doRequest realiza uma requisição HTTP e retorna a resposta decodificada
func (s *IAService) doRequest(ctx context.Context, method, url string, body []byte) (*OrcamentoResponse, error) {
	respBody, err := s.doRequestRaw(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	var response OrcamentoResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &response, nil
}

// doRequestRaw realiza uma requisição HTTP e retorna o body bruto
func (s *IAService) doRequestRaw(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SaaS-Atendimento/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar IA: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro na IA: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ============================================
// MÉTODOS DE CONVENIÊNCIA
// ============================================

// ProcessarMensagem processa uma mensagem simples (wrapper)
func (s *IAService) ProcessarMensagem(ctx context.Context, clienteID, lojaID int, mensagem string) (*OrcamentoResponse, error) {
	req := &OrcamentoRequest{
		ClienteID: clienteID,
		LojaID:    lojaID,
		Mensagem:  mensagem,
	}
	return s.ProcessarOrcamento(ctx, req)
}

// ProcessarOrcamentoComProdutos processa orçamento com lista de produtos específica
func (s *IAService) ProcessarOrcamentoComProdutos(ctx context.Context, clienteID, lojaID int, produtos []string) (*OrcamentoResponse, error) {
	req := &OrcamentoRequest{
		ClienteID: clienteID,
		LojaID:    lojaID,
		Produtos:  produtos,
		Mensagem:  "Orçamento para os produtos: " + fmt.Sprintf("%v", produtos),
	}
	return s.ProcessarOrcamento(ctx, req)
}

// ============================================
// CONFIGURAÇÃO
// ============================================

// SetTimeout define o timeout do cliente HTTP
func (s *IAService) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
	s.httpClient.Timeout = timeout
}

// SetMaxRetries define o número máximo de tentativas
func (s *IAService) SetMaxRetries(maxRetries int) {
	if maxRetries < 1 {
		maxRetries = 1
	}
	s.maxRetries = maxRetries
}

// ============================================
// SIMULAÇÃO (PARA TESTES SEM IA)
// ============================================

// SimularOrcamento simula uma resposta da IA para testes
func (s *IAService) SimularOrcamento(mensagem string) *OrcamentoResponse {
	// Simular produtos encontrados
	produtos := []ProdutoOrcamento{
		{
			Nome:      "Dipirona Sódica 500mg",
			Preco:     15.90,
			Categoria: "Analgésico",
		},
		{
			Nome:      "Paracetamol 750mg",
			Preco:     12.50,
			Categoria: "Analgésico",
		},
	}

	total := 0.0
	for _, p := range produtos {
		total += p.Preco
	}

	return &OrcamentoResponse{
		Orcamento: &OrcamentoData{
			Itens:          produtos,
			Total:          total,
			TotalFormatado: fmt.Sprintf("R$ %.2f", total),
			Quantidade:     len(produtos),
			PrecisaReceita: false,
		},
		PrecisaHumano: total > 500.00,
		RespostaIA: fmt.Sprintf(
			"📋 *Orçamento para %d produto(s)*\n\n• %s - R$ %.2f\n• %s - R$ %.2f\n\n💰 *Total: R$ %.2f*\n\n✅ Produtos disponíveis em estoque.",
			len(produtos),
			produtos[0].Nome, produtos[0].Preco,
			produtos[1].Nome, produtos[1].Preco,
			total,
		),
	}
}