package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Starkgd5/saas-atendimento/internal/cache"
	"github.com/gin-gonic/gin"
)

type CacheMiddleware struct {
	cacheService *cache.CacheService
	defaultTTL   time.Duration
}

func NewCacheMiddleware(cacheService *cache.CacheService, defaultTTL time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cacheService: cacheService,
		defaultTTL:   defaultTTL,
	}
}

// CacheMiddleware cria um middleware de cache para requisições GET
func (m *CacheMiddleware) Cache(ttl ...time.Duration) gin.HandlerFunc {
	duration := m.defaultTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		duration = ttl[0]
	}

	return func(c *gin.Context) {
		// Apenas cache para GET requests
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Ignorar URLs com query params que não devem ser cacheadas
		if m.shouldSkipCache(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Gerar chave do cache
		cacheKey := m.generateCacheKey(c)

		// Tentar buscar do cache
		ctx := context.Background()
		cachedData, err := m.cacheService.Get(ctx, cacheKey)
		if err == nil {
			// Retornar resposta do cache
			var response map[string]interface{}
			if err := json.Unmarshal(cachedData, &response); err == nil {
				c.JSON(http.StatusOK, response)
				c.Abort()
				return
			}
			// Se não conseguir decodificar, retorna como string
			c.String(http.StatusOK, string(cachedData))
			c.Abort()
			return
		}

		// Capturar resposta
		writer := &responseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = writer

		// Processar request
		c.Next()

		// Salvar no cache se status for 200
		if c.Writer.Status() == http.StatusOK {
			responseBody := writer.body.String()
			if responseBody != "" {
				_ = m.cacheService.Set(ctx, cacheKey, responseBody, duration)
			}
		}
	}
}

// shouldSkipCache verifica se a URL deve ser ignorada pelo cache
func (m *CacheMiddleware) shouldSkipCache(path string) bool {
	skipPaths := []string{
		"/health",
		"/ping",
		"/metrics",
		"/socket.io",
		"/ws",
	}

	for _, skip := range skipPaths {
		if strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}

// generateCacheKey gera uma chave única para o cache
func (m *CacheMiddleware) generateCacheKey(c *gin.Context) string {
	// Base: método + path
	key := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.Path)

	// Adicionar query params relevantes
	query := c.Request.URL.Query()
	if len(query) > 0 {
		// Ordenar para consistência
		queryParams := []string{}
		for k, v := range query {
			if m.isCacheableQueryParam(k) {
				queryParams = append(queryParams, fmt.Sprintf("%s=%s", k, strings.Join(v, ",")))
			}
		}
		if len(queryParams) > 0 {
			key += "?" + strings.Join(queryParams, "&")
		}
	}

	// Adicionar loja_id se disponível
	if lojaID, exists := c.Get("lojaID"); exists {
		key += fmt.Sprintf(":loja:%d", lojaID)
	}

	return key
}

// isCacheableQueryParam verifica se um parâmetro de query deve fazer parte da chave do cache
func (m *CacheMiddleware) isCacheableQueryParam(param string) bool {
	cacheableParams := []string{
		"limit", "offset", "page", "sort", "order",
		"categoria", "status", "ativo",
		"data_inicio", "data_fim",
		"cliente_id", "produto_id",
	}

	for _, p := range cacheableParams {
		if param == p {
			return true
		}
	}
	return false
}

// responseWriter captura a resposta para cache
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
