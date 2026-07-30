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
}

type OrcamentoRequest struct {
	ClienteID int      `json:"cliente_id"`
	LojaID    int      `json:"loja_id"`
	Mensagem  string   `json:"mensagem"`
	Historico []string `json:"historico,omitempty"`
}

type OrcamentoResponse struct {
	Orcamento     *OrcamentoData `json:"orcamento,omitempty"`
	PrecisaHumano bool           `json:"precisa_humano"`
	Motivo        string         `json:"motivo,omitempty"`
	RespostaIA    string         `json:"resposta_ia"`
}

type OrcamentoData struct {
	Itens          []ProdutoOrcamento `json:"itens"`
	Total          float64            `json:"total"`
	TotalFormatado string             `json:"total_formatado"`
	Quantidade     int                `json:"quantidade"`
	PrecisaReceita bool               `json:"precisa_receita"`
}

type ProdutoOrcamento struct {
	Nome      string  `json:"nome"`
	Preco     float64 `json:"preco"`
	Categoria string  `json:"categoria"`
}

func NewIAService(baseURL string) *IAService {
	if baseURL == "" {
		baseURL = "http://ia-service:8001"
	}
	return &IAService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *IAService) ProcessarOrcamento(ctx context.Context, req *OrcamentoRequest) (*OrcamentoResponse, error) {
	url := fmt.Sprintf("%s/api/ia/orcamento", s.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar IA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro na IA: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response OrcamentoResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &response, nil
}

// ListarProdutos retorna a lista de produtos da IA
func (s *IAService) ListarProdutos(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/ia/produtos", s.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
