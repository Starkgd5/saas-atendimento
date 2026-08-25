package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	socketio "github.com/googollee/go-socket.io"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Starkgd5/saas-atendimento/internal/cache"
	"github.com/Starkgd5/saas-atendimento/internal/config"
	"github.com/Starkgd5/saas-atendimento/internal/middleware"
	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
	"github.com/Starkgd5/saas-atendimento/internal/services"
	"github.com/Starkgd5/saas-atendimento/internal/websocket"
)

// Variáveis globais
var (
	db               *sql.DB
	redisClient      *redis.Client
	cacheService     *cache.CacheService
	cacheInvalidation *services.CacheInvalidationService
	filaService      *services.FilaService
	wsManager        *websocket.Manager
	clienteRepo      *repository.ClienteRepository
	atendimentoRepo  *repository.AtendimentoRepository
	dashboardService *services.DashboardService
	jwtService       *services.JWTService
	whatsappService  *services.WhatsAppService
	iaService        *services.IAService
	socketServer     *socketio.Server
	
	// Repositórios
	produtoRepo      *repository.ProdutoRepository
	loteRepo         *repository.LoteRepository
	movimentoRepo    *repository.MovimentoEstoqueRepository
	vendaRepo        *repository.VendaRepository
	fornecedorRepo   *repository.FornecedorRepository
	compraRepo       *repository.CompraRepository
	receitaRepo      *repository.ReceitaRepository
	orcamentoRepo    *repository.OrcamentoRepository
	reclamacaoRepo   *repository.ReclamacaoRepository
	configRepo       *repository.ConfiguracaoRepository
	
	// Serviços
	estoqueService   *services.EstoqueService
	vendaService     *services.VendaService
	fornecedorService *services.FornecedorService
	compraService    *services.CompraService
	receitaService   *services.ReceitaService
	orcamentoService *services.OrcamentoService
	reclamacaoService *services.ReclamacaoService
	configService    *services.ConfiguracaoService
	cacheMiddleware  *middleware.CacheMiddleware
)

func main() {
	// Carregar .env
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis do sistema")
	}

	// Logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// Configuração
	cfg := config.Load()

	// Conectar ao MariaDB
	var err error
	db, err = sql.Open("mysql", cfg.DB.DSN)
	if err != nil {
		zap.L().Fatal("Erro ao conectar ao MariaDB", zap.Error(err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		zap.L().Fatal("Erro ao pingar MariaDB", zap.Error(err))
	}
	zap.L().Info("✅ Conectado ao MariaDB")

	// Conectar ao Redis com autenticação
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.URL,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Testar conexão com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = redisClient.Ping(ctx).Err(); err != nil {
		zap.L().Fatal("Erro ao conectar ao Redis", zap.Error(err))
	}
	zap.L().Info("✅ Conectado ao Redis")

	// Inicializar Cache
	cacheService = cache.NewCacheService(redisClient)
	cacheInvalidation = services.NewCacheInvalidationService(cacheService)
	cacheMiddleware = middleware.NewCacheMiddleware(cacheService, 30*time.Second)
	zap.L().Info("✅ Cache Service inicializado")

	// Inicializar repositórios
	clienteRepo = repository.NewClienteRepository(db)
	atendimentoRepo = repository.NewAtendimentoRepository(db)
	produtoRepo = repository.NewProdutoRepository(db)
	loteRepo = repository.NewLoteRepository(db)
	movimentoRepo = repository.NewMovimentoEstoqueRepository(db)
	vendaRepo = repository.NewVendaRepository(db)
	fornecedorRepo = repository.NewFornecedorRepository(db)
	compraRepo = repository.NewCompraRepository(db)
	receitaRepo = repository.NewReceitaRepository(db)
	orcamentoRepo = repository.NewOrcamentoRepository(db)
	reclamacaoRepo = repository.NewReclamacaoRepository(db)
	configRepo = repository.NewConfiguracaoRepository(db)

	// Inicializar serviços
	filaService = services.NewFilaService(redisClient, cfg.MaxClients)
	jwtService = services.NewJWTService(cfg.JWTSecret)
	whatsappService = services.NewWhatsAppService(&cfg.WhatsApp)
	dashboardService = services.NewDashboardService(db)
	iaService = services.NewIAService(os.Getenv("IA_URL"))
	
	// Serviços de negócio
	estoqueService = services.NewEstoqueService(produtoRepo, loteRepo, movimentoRepo, db)
	vendaService = services.NewVendaService(vendaRepo, produtoRepo, loteRepo, estoqueService, db)
	fornecedorService = services.NewFornecedorService(fornecedorRepo)
	compraService = services.NewCompraService(compraRepo, fornecedorRepo, produtoRepo, loteRepo, estoqueService, db)
	receitaService = services.NewReceitaService(receitaRepo, clienteRepo)
	orcamentoService = services.NewOrcamentoService(orcamentoRepo, produtoRepo, clienteRepo, loteRepo)
	reclamacaoService = services.NewReclamacaoService(reclamacaoRepo, clienteRepo)
	configService = services.NewConfiguracaoService(configRepo)

	// Inicializar WebSocket Manager
	wsManager = websocket.NewManager()
	go wsManager.Run()
	zap.L().Info("✅ WebSocket Manager iniciado")

	// Configurar rotas
	router := setupRouter()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		zap.L().Info("🚀 Servidor iniciado na porta 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Falha ao iniciar servidor", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("🛑 Desligando servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Erro no shutdown", zap.Error(err))
	}

	zap.L().Info("✅ Servidor desligado com sucesso")
}

// ============================================
// SETUP ROUTER
// ============================================

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health Check
	router.GET("/health", healthCheck)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	// WebSocket
	router.GET("/ws", wsHandler)

	api := router.Group("/api/v1")
	{
		// ============================================
		// ROTAS PÚBLICAS
		// ============================================
		api.POST("/auth/login", authLogin)
		api.POST("/auth/refresh", authRefresh)
		api.POST("/auth/register", registerUser)
		
		// Webhook WhatsApp
		api.GET("/webhook/whatsapp", webhookWhatsAppVerify)
		api.POST("/webhook/whatsapp", webhookWhatsApp)

		// ============================================
		// ROTAS PROTEGIDAS (JWT)
		// ============================================
		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			// ==========================================
			// DASHBOARD
			// ==========================================
			auth.GET("/dashboard", cacheMiddleware.Cache(30*time.Second), getDashboard)
			auth.GET("/dashboard/metricas", cacheMiddleware.Cache(30*time.Second), getMetricasDetalhadas)
			auth.GET("/dashboard/farmacia", cacheMiddleware.Cache(30*time.Second), dashboardFarmaciaHandler)

			// ==========================================
			// FILA
			// ==========================================
			auth.GET("/fila/status", getFilaStatus)
			auth.GET("/fila/clientes", getFilaClientes)
			auth.POST("/fila/proximo", puxarProximoCliente)
			auth.PUT("/fila/config", configurarLimite)

			// ==========================================
			// CLIENTES
			// ==========================================
			auth.GET("/clientes", cacheMiddleware.Cache(5*time.Minute), listarClientes)
			auth.GET("/clientes/:id", cacheMiddleware.Cache(5*time.Minute), getCliente)
			auth.POST("/clientes", criarCliente)
			auth.PUT("/clientes/:id", atualizarCliente)
			auth.DELETE("/clientes/:id", deletarCliente)

			// ==========================================
			// ATENDIMENTOS
			// ==========================================
			auth.GET("/atendimentos", listarAtendimentos)
			auth.GET("/atendimentos/:id", getAtendimento)
			auth.POST("/atendimentos/:id/finalizar", finalizarAtendimentoHandler)
			auth.POST("/atendimentos/:id/abandonar", abandonarAtendimentoHandler)
			auth.POST("/atendimentos/:id/enviar", enviarMensagem)

			// ==========================================
			// MENSAGENS
			// ==========================================
			auth.GET("/mensagens", listarMensagens)
			auth.GET("/mensagens/:id", getMensagem)

			// ==========================================
			// PRODUTOS
			// ==========================================
			auth.GET("/produtos", cacheMiddleware.Cache(5*time.Minute), listarProdutosHandler)
			auth.GET("/produtos/:id", cacheMiddleware.Cache(5*time.Minute), getProdutoHandler)
			auth.GET("/produtos/codigo/:codigo", cacheMiddleware.Cache(5*time.Minute), getProdutoPorCodigoHandler)
			auth.POST("/produtos", criarProdutoHandler)
			auth.PUT("/produtos/:id", atualizarProdutoHandler)
			// auth.DELETE("/produtos/:id", deletarProdutoHandler)

			// ==========================================
			// ESTOQUE
			// ==========================================
			auth.GET("/estoque", cacheMiddleware.Cache(2*time.Minute), listarEstoqueHandler)
			auth.GET("/estoque/produtos", cacheMiddleware.Cache(2*time.Minute), listarProdutosEstoqueHandler)
			auth.GET("/estoque/produtos/:id", cacheMiddleware.Cache(2*time.Minute), getEstoqueProdutoHandler)
			auth.GET("/estoque/lotes/:produto_id", cacheMiddleware.Cache(2*time.Minute), listarLotesHandler)
			auth.GET("/estoque/alertas", cacheMiddleware.Cache(1*time.Minute), alertasEstoqueHandler)
			auth.GET("/estoque/movimentacoes", cacheMiddleware.Cache(1*time.Minute), movimentacoesEstoqueHandler)
			auth.GET("/estoque/resumo", cacheMiddleware.Cache(2*time.Minute), resumoEstoqueHandler)
			auth.POST("/estoque/entrada", entradaEstoqueHandler)
			auth.POST("/estoque/saida", saidaEstoqueHandler)
			auth.POST("/estoque/ajuste", ajusteEstoqueHandler)


			// ==========================================
			// VENDAS (PDV)
			// ==========================================
			auth.POST("/vendas", criarVendaHandler)
			auth.GET("/vendas", cacheMiddleware.Cache(1*time.Minute), listarVendasHandler)
			auth.GET("/vendas/:id", cacheMiddleware.Cache(1*time.Minute), getVendaHandler)
			auth.PUT("/vendas/:id", atualizarVendaHandler)
			auth.POST("/vendas/:id/cancelar", cancelarVendaHandler)
			auth.GET("/vendas/relatorio/diario", cacheMiddleware.Cache(1*time.Minute), relatorioVendasHandler)

			// ==========================================
			// ORÇAMENTOS
			// ==========================================
			auth.GET("/orcamentos", cacheMiddleware.Cache(5*time.Minute), listarOrcamentosHandler)
			auth.GET("/orcamentos/:id", cacheMiddleware.Cache(5*time.Minute), getOrcamentoHandler)
			auth.POST("/orcamentos", criarOrcamentoHandler)
			auth.PUT("/orcamentos/:id", atualizarOrcamentoHandler)
			auth.POST("/orcamentos/:id/aprovar", aprovarOrcamentoHandler)
			auth.POST("/orcamentos/:id/rejeitar", rejeitarOrcamentoHandler)

			// ==========================================
			// RECEITAS MÉDICAS
			// ==========================================
			auth.GET("/receitas", cacheMiddleware.Cache(2*time.Minute), listarReceitasHandler)
			auth.GET("/receitas/:id", cacheMiddleware.Cache(2*time.Minute), getReceitaHandler)
			auth.POST("/receitas", criarReceitaHandler)
			auth.PUT("/receitas/:id", atualizarReceitaHandler)
			auth.POST("/receitas/:id/validar", validarReceitaHandler)
			auth.POST("/receitas/:id/rejeitar", rejeitarReceitaHandler)

			// ==========================================
			// RECLAMAÇÕES
			// ==========================================
			auth.GET("/reclamacoes", cacheMiddleware.Cache(2*time.Minute), listarReclamacoesHandler)
			auth.GET("/reclamacoes/:id", cacheMiddleware.Cache(2*time.Minute), getReclamacaoHandler)
			auth.POST("/reclamacoes", criarReclamacaoHandler)
			auth.PUT("/reclamacoes/:id", atualizarReclamacaoHandler)
			auth.POST("/reclamacoes/:id/resolver", resolverReclamacaoHandler)

			// ==========================================
			// FORNECEDORES
			// ==========================================
			auth.GET("/fornecedores", cacheMiddleware.Cache(5*time.Minute), listarFornecedoresHandler)
			auth.GET("/fornecedores/:id", cacheMiddleware.Cache(5*time.Minute), getFornecedorHandler)
			auth.POST("/fornecedores", criarFornecedorHandler)
			auth.PUT("/fornecedores/:id", atualizarFornecedorHandler)
			auth.DELETE("/fornecedores/:id", deletarFornecedorHandler)

			// ==========================================
			// COMPRAS
			// ==========================================
			auth.GET("/compras", cacheMiddleware.Cache(2*time.Minute), listarComprasHandler)
			auth.GET("/compras/:id", cacheMiddleware.Cache(2*time.Minute), getCompraHandler)
			auth.POST("/compras", criarCompraHandler)
			auth.PUT("/compras/:id", atualizarCompraHandler)
			// auth.POST("/compras/:id/receber", receberCompraHandler)

			// ==========================================
			// FINANCEIRO
			// ==========================================
			auth.GET("/financeiro/contas-receber", cacheMiddleware.Cache(2*time.Minute), listarContasReceberHandler)
			auth.GET("/financeiro/contas-pagar", cacheMiddleware.Cache(2*time.Minute), listarContasPagarHandler)
			auth.POST("/financeiro/contas-receber", criarContaReceberHandler)
			auth.POST("/financeiro/contas-pagar", criarContaPagarHandler)
			auth.PUT("/financeiro/contas/:id/baixar", baixarContaHandler)

			// ==========================================
			// IA
			// ==========================================
			auth.GET("/ia/produtos", listarProdutosIA)
			auth.POST("/ia/orcamento", processarOrcamentoIA)
			auth.GET("/ia/cache", getIACacheStatus)
			auth.DELETE("/ia/cache", clearIACache)

			// ==========================================
			// USUÁRIOS
			// ==========================================
			auth.GET("/usuarios", cacheMiddleware.Cache(15*time.Minute), listarUsuariosHandler)
			auth.GET("/usuarios/:id", cacheMiddleware.Cache(15*time.Minute), getUsuarioHandler)
			auth.POST("/usuarios", criarUsuarioHandler)
			auth.PUT("/usuarios/:id", atualizarUsuarioHandler)
			auth.PATCH("/usuarios/:id/toggle", toggleUsuarioHandler)
			auth.DELETE("/usuarios/:id", deletarUsuarioHandler)

			// ==========================================
			// CONFIGURAÇÕES
			// ==========================================
			auth.GET("/configuracoes", cacheMiddleware.Cache(30*time.Minute), listarConfiguracoesHandler)
			auth.PUT("/configuracoes", atualizarConfiguracoesHandler)
		}
	}

	return router
}

// ============================================
// MIDDLEWARES
// ============================================

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept")
		c.Header("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token não fornecido"})
			return
		}

		var tokenString string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "formato de token inválido"})
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("lojaID", claims.LojaID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// ============================================
// WEBSOCKET
// ============================================

func wsHandler(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido"})
		return
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
		return
	}

	wsManager.ServeWS(c.Writer, c.Request, claims.UserID, claims.LojaID)
}

// ============================================
// AUTH HANDLERS
// ============================================

func authLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	var userID int
	var nome, role, senhaHash string
	var lojaID sql.NullInt64

	err := db.QueryRow(`
		SELECT id, nome, role, senha_hash, loja_id 
		FROM usuarios 
		WHERE email = ? AND ativo = 1
	`, req.Email).Scan(&userID, &nome, &role, &senhaHash, &lojaID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar usuário"})
		return
	}

	// Verificar senha (simplificado para demo)
	if senhaHash != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Senha incorreta"})
		return
	}

	lojaIDInt := 0
	if lojaID.Valid {
		lojaIDInt = int(lojaID.Int64)
	}

	token, err := jwtService.GenerateTokenWithDetails(userID, lojaIDInt, role, req.Email, nome)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":      userID,
			"nome":    nome,
			"email":   req.Email,
			"role":    role,
			"loja_id": lojaIDInt,
		},
	})
}

func authRefresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "new-token"})
}

func registerUser(c *gin.Context) {
	var req struct {
		Nome  string `json:"nome"`
		Email string `json:"email"`
		Senha string `json:"password"`
		Role  string `json:"role"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if req.Role == "" {
		req.Role = "atendente"
	}

	_, err := db.Exec(`
		INSERT INTO usuarios (nome, email, senha_hash, role) 
		VALUES (?, ?, ?, ?)
	`, req.Nome, req.Email, req.Senha, req.Role)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuário registrado com sucesso"})
}

// ============================================
// DASHBOARD HANDLERS
// ============================================

func getDashboard(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	ctx := context.Background()

	metrics, err := dashboardService.GetMetrics(ctx, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filaStatus, err := filaService.GetStatusFila(ctx)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"metrics": metrics})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics":   metrics,
		"fila":      filaStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func getMetricasDetalhadas(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	ctx := context.Background()

	metricasDiarias, _ := dashboardService.GetMetricasDiarias(ctx, lojaID.(int), 30)
	metricasPorHora, _ := dashboardService.GetMetricasPorHora(ctx, lojaID.(int))
	satisfacao, _ := dashboardService.GetSatisfacaoCliente(ctx, lojaID.(int))

	c.JSON(http.StatusOK, gin.H{
		"metricas_diarias":  metricasDiarias,
		"metricas_por_hora": metricasPorHora,
		"satisfacao":        satisfacao,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

func dashboardFarmaciaHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	ctx := c.Request.Context()

	var vendasHoje int
	var faturamentoHoje float64
	db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total), 0)
		FROM vendas
		WHERE loja_id = ? AND DATE(created_at) = CURDATE() AND status = 'Pago'
	`, lojaID).Scan(&vendasHoje, &faturamentoHoje)

	var ticketMedio float64
	db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(total), 0)
		FROM vendas
		WHERE loja_id = ? AND DATE(created_at) = CURDATE() AND status = 'Pago'
	`, lojaID).Scan(&ticketMedio)

	var receitasPendentes int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM receitas_medicas
		WHERE loja_id = ? AND status = 'Pendente'
	`, lojaID).Scan(&receitasPendentes)

	c.JSON(http.StatusOK, gin.H{
		"vendas_hoje":      vendasHoje,
		"faturamento_hoje": faturamentoHoje,
		"ticket_medio":     ticketMedio,
		"receitas_pendentes": receitasPendentes,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// ============================================
// FILA HANDLERS
// ============================================

func getFilaStatus(c *gin.Context) {
	ctx := context.Background()
	status, err := filaService.GetStatusFila(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func getFilaClientes(c *gin.Context) {
	ctx := context.Background()
	clientes, err := filaService.GetClientesNaFila(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, clientes)
}

func puxarProximoCliente(c *gin.Context) {
	ctx := context.Background()
	clienteID, err := filaService.ProximoClienteAtendimento(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if clienteID == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "fila vazia"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cliente_id": clienteID})
}

func configurarLimite(c *gin.Context) {
	var req struct {
		Limite int `json:"limite"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limite inválido"})
		return
	}
	ctx := context.Background()
	if err := filaService.SetLimite(ctx, req.Limite); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Limite atualizado", "limite": req.Limite})
}

// ============================================
// CLIENTE HANDLERS
// ============================================

func listarClientes(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	clientes, total, err := clienteRepo.ListarClientes(lojaID.(int), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": clientes,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getCliente(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	cliente, err := clienteRepo.BuscarClientePorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cliente == nil || cliente.LojaID != lojaID.(int) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente não encontrado"})
		return
	}

	c.JSON(http.StatusOK, cliente)
}

func criarCliente(c *gin.Context) {
	var req models.Cliente
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	req.LojaID = lojaID.(int)
	req.Ativo = true

	if err := clienteRepo.CriarCliente(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Cliente criado com sucesso", "cliente": req})
}

func atualizarCliente(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req models.Cliente
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	req.ID = id
	req.LojaID = lojaID.(int)

	if err := clienteRepo.AtualizarCliente(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente atualizado com sucesso"})
}

func deletarCliente(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	if err := clienteRepo.DeletarCliente(id, lojaID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente deletado com sucesso"})
}

// ============================================
// ATENDIMENTO HANDLERS
// ============================================

func listarAtendimentos(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	atendimentos, err := atendimentoRepo.ListarAtendimentos(lojaID.(int), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": atendimentos,
		"total": len(atendimentos),
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getAtendimento(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	atendimento, err := atendimentoRepo.BuscarAtendimentoPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if atendimento == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Atendimento não encontrado"})
		return
	}
	c.JSON(http.StatusOK, atendimento)
}

func finalizarAtendimentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	usuarioID, _ := c.Get("userID")
	atendenteID := usuarioID.(int)

	// Buscar atendimento
	atendimento, err := atendimentoRepo.BuscarAtendimentoPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if atendimento == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Atendimento não encontrado"})
		return
	}

	// Finalizar na fila
	ctx := context.Background()
	if err := filaService.FinalizarAtendimento(ctx, atendimento.ClienteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Atualizar no banco
	if err := atendimentoRepo.AtualizarStatusAtendimento(id, "finalizado", atendenteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Atendimento finalizado com sucesso"})
}

func abandonarAtendimentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	atendimento, err := atendimentoRepo.BuscarAtendimentoPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if atendimento == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Atendimento não encontrado"})
		return
	}

	ctx := context.Background()
	if err := filaService.AbandonarAtendimento(ctx, atendimento.ClienteID, id, db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Atendimento abandonado com sucesso"})
}

func enviarMensagem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Mensagem string `json:"mensagem"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mensagem inválida"})
		return
	}

	// Inserir mensagem
	_, err := db.Exec(`
		INSERT INTO mensagens (atendimento_id, remetente, conteudo)
		VALUES (?, 'atendente', ?)
	`, id, req.Mensagem)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notificar via WebSocket
	msg, _ := json.Marshal(gin.H{
		"type": "nova_mensagem",
		"payload": gin.H{
			"atendimento_id": id,
			"mensagem":       req.Mensagem,
		},
	})
	wsManager.EnviarParaTodos(msg)

	c.JSON(http.StatusOK, gin.H{"message": "Mensagem enviada com sucesso"})
}

// ============================================
// MENSAGEM HANDLERS
// ============================================

func listarMensagens(c *gin.Context) {
	atendimentoID := c.DefaultQuery("atendimento_id", "")
	if atendimentoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "atendimento_id é obrigatório"})
		return
	}

	id, _ := strconv.Atoi(atendimentoID)
	mensagens, err := atendimentoRepo.ListarMensagens(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mensagens)
}

func getMensagem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	mensagem, err := atendimentoRepo.BuscarMensagem(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if mensagem == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mensagem não encontrada"})
		return
	}
	c.JSON(http.StatusOK, mensagem)
}

// ============================================
// PRODUTO HANDLERS
// ============================================

func listarProdutosHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	categoria := c.DefaultQuery("categoria", "")
	ativo := c.DefaultQuery("ativo", "true")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	ativoBool := ativo == "true"
	produtos, total, err := produtoRepo.ListarProdutos(lojaID.(int), categoria, &ativoBool, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": produtos,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getProdutoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	produto, err := produtoRepo.BuscarProdutoPorID(id, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if produto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produto não encontrado"})
		return
	}
	c.JSON(http.StatusOK, produto)
}

func getProdutoPorCodigoHandler(c *gin.Context) {
	codigo := c.Param("codigo")
	lojaID, _ := c.Get("lojaID")

	produto, err := produtoRepo.BuscarProdutoPorCodigoBarras(codigo, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if produto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produto não encontrado"})
		return
	}
	c.JSON(http.StatusOK, produto)
}

func criarProdutoHandler(c *gin.Context) {
	var req models.Produto
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	req.LojaID = lojaID.(int)
	req.Ativo = true

	if err := produtoRepo.CriarProduto(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateProduto(context.Background(), req.ID, lojaID.(int))

	c.JSON(http.StatusCreated, gin.H{"message": "Produto criado com sucesso", "produto": req})
}

func atualizarProdutoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req models.Produto
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	req.ID = id
	req.LojaID = lojaID.(int)

	if err := produtoRepo.AtualizarProduto(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateProduto(context.Background(), id, lojaID.(int))

	c.JSON(http.StatusOK, gin.H{"message": "Produto atualizado com sucesso"})
}

// func deletarProdutoHandler(c *gin.Context) {
// 	id, _ := strconv.Atoi(c.Param("id"))
// 	lojaID, _ := c.Get("lojaID")

// 	if err := produtoRepo.DeletarProduto(id, lojaID.(int)); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	cacheInvalidation.InvalidateProduto(context.Background(), id, lojaID.(int))

// 	c.JSON(http.StatusOK, gin.H{"message": "Produto deletado com sucesso"})
// }

// ============================================
// ESTOQUE HANDLERS
// ============================================

// listarEstoqueHandler lista todos os produtos com informações de estoque
func listarEstoqueHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	
	// Parâmetros de paginação e filtro
	categoria := c.DefaultQuery("categoria", "")
	status := c.DefaultQuery("status", "todos") // todos, baixo, critico, vencido
	search := c.DefaultQuery("search", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	// Buscar produtos da loja
	produtos, _, err := produtoRepo.ListarProdutos(lojaID.(int), categoria, nil, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Construir resposta com informações de estoque
	var estoqueItems []map[string]interface{}
	
	for _, produto := range produtos {
		// Buscar lotes do produto
		lotes, err := loteRepo.BuscarLotesPorProduto(produto.ID, lojaID.(int))
		if err != nil {
			continue
		}
		
		// Calcular estoque total
		quantidadeTotal := 0
		var lotesAtivos []map[string]interface{}
		var lotesVencidos []map[string]interface{}
		
		for _, lote := range lotes {
			if lote.Status == "Ativo" {
				quantidadeTotal += lote.Quantidade
				lotesAtivos = append(lotesAtivos, map[string]interface{}{
					"id":          lote.ID,
					"numero_lote": lote.NumeroLote,
					"quantidade":  lote.Quantidade,
					"data_validade": lote.DataValidade.Format("2006-01-02"),
					"preco_venda": lote.PrecoVenda,
				})
			} else if lote.Status == "Vencido" {
				lotesVencidos = append(lotesVencidos, map[string]interface{}{
					"id":          lote.ID,
					"numero_lote": lote.NumeroLote,
					"quantidade":  lote.Quantidade,
					"data_validade": lote.DataValidade.Format("2006-01-02"),
				})
			}
		}
		
		// Determinar status do estoque
		statusEstoque := "normal"
		statusMessage := ""
		
		if quantidadeTotal <= produto.EstoqueMinimo {
			if quantidadeTotal == 0 {
				statusEstoque = "critico"
				statusMessage = "⚠️ Produto sem estoque"
			} else {
				statusEstoque = "baixo"
				statusMessage = "⚠️ Estoque abaixo do mínimo"
			}
		} else if len(lotesVencidos) > 0 {
			statusEstoque = "vencido"
			statusMessage = "⚠️ Produto com lotes vencidos"
		}
		
		// Filtrar por status se especificado
		if status != "todos" && status != statusEstoque {
			continue
		}
		
		// Filtrar por search
		if search != "" && !containsString(produto.Nome, search) && !containsString(produto.CodigoBarras, search) {
			continue
		}
		
		// Verificar produtos vencendo (próximos 30 dias)
		vencendoEmBreve := false
		for _, lote := range lotes {
			if lote.Status == "Ativo" {
				dias := int(time.Until(lote.DataValidade).Hours() / 24)
				if dias <= 30 && dias > 0 {
					vencendoEmBreve = true
					break
				}
			}
		}
		
		// Construir item de estoque
		item := map[string]interface{}{
			"id":              produto.ID,
			"nome":            produto.Nome,
			"codigo_barras":   produto.CodigoBarras,
			"categoria":       produto.Categoria,
			"fabricante":      produto.Fabricante,
			"preco_venda":     produto.PrecoVenda,
			"preco_custo":     produto.PrecoCusto,
			"estoque_minimo":  produto.EstoqueMinimo,
			"estoque_maximo":  produto.EstoqueMaximo,
			"quantidade_total": quantidadeTotal,
			"status":          statusEstoque,
			"status_message":  statusMessage,
			"vencendo_em_breve": vencendoEmBreve,
			"lotes_ativos":    lotesAtivos,
			"lotes_vencidos":  lotesVencidos,
			"total_lotes":     len(lotes),
			"ativo":           produto.Ativo,
		}
		
		estoqueItems = append(estoqueItems, item)
	}
	
	// Retornar resposta
	c.JSON(http.StatusOK, gin.H{
		"items": estoqueItems,
		"total": len(estoqueItems),
		"page":  offset/limit + 1,
		"limit": limit,
		"filters": gin.H{
			"categoria": categoria,
			"status":    status,
			"search":    search,
		},
	})
}

// ============================================
// FUNÇÕES AUXILIARES
// ============================================

// containsString verifica se uma string contém outra (case insensitive)
func containsString(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// listarProdutosEstoqueHandler lista produtos com estoque (versão simplificada)
func listarProdutosEstoqueHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	
	produtos, total, err := produtoRepo.ListarProdutos(lojaID.(int), "", nil, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	var items []map[string]interface{}
	for _, p := range produtos {
		// Calcular estoque total
		lotes, err := loteRepo.BuscarLotesPorProduto(p.ID, lojaID.(int))
		if err != nil {
			continue
		}
		
		quantidadeTotal := 0
		for _, l := range lotes {
			if l.Status == "Ativo" {
				quantidadeTotal += l.Quantidade
			}
		}
		
		items = append(items, map[string]interface{}{
			"id":              p.ID,
			"nome":            p.Nome,
			"codigo_barras":   p.CodigoBarras,
			"categoria":       p.Categoria,
			"preco_venda":     p.PrecoVenda,
			"quantidade_total": quantidadeTotal,
			"estoque_minimo":  p.EstoqueMinimo,
			"ativo":           p.Ativo,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

// getEstoqueProdutoHandler busca estoque de um produto específico
func getEstoqueProdutoHandler(c *gin.Context) {
	produtoID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	
	lojaID, _ := c.Get("lojaID")
	
	// Buscar produto
	produto, err := produtoRepo.BuscarProdutoPorID(produtoID, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if produto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produto não encontrado"})
		return
	}
	
	// Buscar lotes
	lotes, err := loteRepo.BuscarLotesPorProduto(produtoID, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Calcular estoque total e organizar lotes
	quantidadeTotal := 0
	var lotesAtivos []map[string]interface{}
	var lotesVencidos []map[string]interface{}
	
	for _, lote := range lotes {
		loteInfo := map[string]interface{}{
			"id":             lote.ID,
			"numero_lote":    lote.NumeroLote,
			"quantidade":     lote.Quantidade,
			"data_fabricacao": lote.DataFabricacao.Format("2006-01-02"),
			"data_validade":  lote.DataValidade.Format("2006-01-02"),
			"preco_custo":    lote.PrecoCusto,
			"preco_venda":    lote.PrecoVenda,
		}
		
		if lote.Status == "Ativo" {
			quantidadeTotal += lote.Quantidade
			lotesAtivos = append(lotesAtivos, loteInfo)
		} else if lote.Status == "Vencido" {
			lotesVencidos = append(lotesVencidos, loteInfo)
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"produto": gin.H{
			"id":              produto.ID,
			"nome":            produto.Nome,
			"codigo_barras":   produto.CodigoBarras,
			"categoria":       produto.Categoria,
			"preco_venda":     produto.PrecoVenda,
			"preco_custo":     produto.PrecoCusto,
			"estoque_minimo":  produto.EstoqueMinimo,
			"ativo":           produto.Ativo,
		},
		"estoque": gin.H{
			"quantidade_total": quantidadeTotal,
			"lotes_ativos":     lotesAtivos,
			"lotes_vencidos":   lotesVencidos,
			"total_lotes":      len(lotes),
		},
	})
}

// movimentacoesEstoqueHandler lista movimentações de estoque
func movimentacoesEstoqueHandler(c *gin.Context) {
	produtoID := c.DefaultQuery("produto_id", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	lojaID, _ := c.Get("lojaID")
	
	if produtoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produto_id é obrigatório"})
		return
	}
	
	id, err := strconv.Atoi(produtoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produto_id inválido"})
		return
	}
	
	movimentos, err := movimentoRepo.ListarMovimentosPorProduto(id, lojaID.(int), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"items": movimentos,
		"total": len(movimentos),
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

// resumoEstoqueHandler retorna um resumo do estoque
func resumoEstoqueHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	
	// Total de produtos
	var totalProdutos int
	db.QueryRow(`
		SELECT COUNT(*) FROM produtos WHERE loja_id = ? AND ativo = 1
	`, lojaID).Scan(&totalProdutos)
	
	// Produtos com estoque baixo
	var produtosBaixo int
	db.QueryRow(`
		SELECT COUNT(DISTINCT p.id)
		FROM produtos p
		LEFT JOIN lotes l ON p.id = l.produto_id AND l.status = 'Ativo'
		WHERE p.loja_id = ? AND p.ativo = 1
		GROUP BY p.id
		HAVING COALESCE(SUM(l.quantidade), 0) <= p.estoque_minimo
	`, lojaID).Scan(&produtosBaixo)
	
	// Produtos com estoque zerado
	var produtosZerados int
	db.QueryRow(`
		SELECT COUNT(DISTINCT p.id)
		FROM produtos p
		LEFT JOIN lotes l ON p.id = l.produto_id AND l.status = 'Ativo'
		WHERE p.loja_id = ? AND p.ativo = 1
		GROUP BY p.id
		HAVING COALESCE(SUM(l.quantidade), 0) = 0
	`, lojaID).Scan(&produtosZerados)
	
	// Produtos vencendo (próximos 30 dias)
	var produtosVencendo int
	db.QueryRow(`
		SELECT COUNT(DISTINCT produto_id)
		FROM lotes
		WHERE loja_id = ? AND status = 'Ativo'
		AND data_validade BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL 30 DAY)
	`, lojaID).Scan(&produtosVencendo)
	
	c.JSON(http.StatusOK, gin.H{
		"total_produtos":    totalProdutos,
		"produtos_baixo":    produtosBaixo,
		"produtos_zerados":  produtosZerados,
		"produtos_vencendo": produtosVencendo,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

func listarLotesHandler(c *gin.Context) {
	produtoID, _ := strconv.Atoi(c.Param("produto_id"))
	lojaID, _ := c.Get("lojaID")

	lotes, err := loteRepo.BuscarLotesPorProduto(produtoID, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lotes)
}

func alertasEstoqueHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	alertas, err := estoqueService.ObterAlertasEstoque(c.Request.Context(), lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alertas)
}

func entradaEstoqueHandler(c *gin.Context) {
	var req struct {
		ProdutoID     int     `json:"produto_id"`
		NumeroLote    string  `json:"numero_lote"`
		DataFabricacao string `json:"data_fabricacao"`
		DataValidade  string  `json:"data_validade"`
		Quantidade    int     `json:"quantidade"`
		PrecoCusto    float64 `json:"preco_custo"`
		PrecoVenda    float64 `json:"preco_venda"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	usuarioID, _ := c.Get("userID")

	dataFabricacao, _ := time.Parse("2006-01-02", req.DataFabricacao)
	dataValidade, _ := time.Parse("2006-01-02", req.DataValidade)

	lote := &models.Lote{
		ProdutoID:     req.ProdutoID,
		LojaID:        lojaID.(int),
		NumeroLote:    req.NumeroLote,
		DataFabricacao: dataFabricacao,
		DataValidade:  dataValidade,
		Quantidade:    req.Quantidade,
		QuantidadeInicial: req.Quantidade,
		PrecoCusto:    req.PrecoCusto,
		PrecoVenda:    req.PrecoVenda,
		Status:        "Ativo",
	}

	if err := estoqueService.EntradaEstoque(c.Request.Context(), req.ProdutoID, lojaID.(int), lote, usuarioID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateProduto(c.Request.Context(), req.ProdutoID, lojaID.(int))
	cacheInvalidation.InvalidateDashboard(c.Request.Context(), lojaID.(int))

	c.JSON(http.StatusCreated, gin.H{"message": "Entrada de estoque realizada com sucesso", "lote": lote})
}

func saidaEstoqueHandler(c *gin.Context) {
	var req struct {
		ProdutoID  int    `json:"produto_id"`
		LoteID     int    `json:"lote_id"`
		Quantidade int    `json:"quantidade"`
		Motivo     string `json:"motivo"`
		Documento  string `json:"documento"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	usuarioID, _ := c.Get("userID")

	if err := estoqueService.SaidaEstoque(c.Request.Context(), req.ProdutoID, lojaID.(int), req.Quantidade, req.LoteID, usuarioID.(int), req.Motivo, req.Documento); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateProduto(c.Request.Context(), req.ProdutoID, lojaID.(int))
	cacheInvalidation.InvalidateDashboard(c.Request.Context(), lojaID.(int))

	c.JSON(http.StatusOK, gin.H{"message": "Saída de estoque realizada com sucesso"})
}

func ajusteEstoqueHandler(c *gin.Context) {
	var req struct {
		ProdutoID  int    `json:"produto_id"`
		LoteID     int    `json:"lote_id"`
		Quantidade int    `json:"quantidade"`
		Motivo     string `json:"motivo"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	// Implementar ajuste de estoque
	c.JSON(http.StatusOK, gin.H{"message": "Ajuste de estoque realizado com sucesso"})
}

// ============================================
// VENDA HANDLERS (PDV)
// ============================================

func criarVendaHandler(c *gin.Context) {
	var req struct {
		ClienteID     *int               `json:"cliente_id"`
		TipoPagamento string             `json:"tipo_pagamento"`
		Desconto      float64            `json:"desconto"`
		ReceitaAnexada bool              `json:"receita_anexada"`
		Observacao    string             `json:"observacao"`
		Itens         []models.ItemVenda `json:"itens"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	usuarioID, _ := c.Get("userID")

	if len(req.Itens) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Venda deve ter pelo menos um item"})
		return
	}

	venda := &models.Venda{
		LojaID:        lojaID.(int),
		ClienteID:     req.ClienteID,
		TipoPagamento: req.TipoPagamento,
		Desconto:      req.Desconto,
		ReceitaAnexada: req.ReceitaAnexada,
		Observacao:    req.Observacao,
		Itens:         req.Itens,
	}

	result, err := vendaService.ProcessarVenda(c.Request.Context(), venda, req.Itens, usuarioID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateVenda(c.Request.Context(), lojaID.(int))
	cacheInvalidation.InvalidateDashboard(c.Request.Context(), lojaID.(int))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Venda realizada com sucesso",
		"venda":   result,
	})
}

func listarVendasHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	vendas, total, err := vendaRepo.ListarVendas(lojaID.(int), status, "", "", limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": vendas,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getVendaHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	venda, err := vendaRepo.BuscarVendaPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if venda == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Venda não encontrada"})
		return
	}
	c.JSON(http.StatusOK, venda)
}

func atualizarVendaHandler(c *gin.Context) {
	// Implementar atualização de venda
	c.JSON(http.StatusOK, gin.H{"message": "Venda atualizada com sucesso"})
}

func cancelarVendaHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	usuarioID, _ := c.Get("userID")

	if err := vendaService.CancelarVenda(c.Request.Context(), id, usuarioID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheInvalidation.InvalidateVenda(c.Request.Context(), 1)

	c.JSON(http.StatusOK, gin.H{"message": "Venda cancelada com sucesso"})
}

func relatorioVendasHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	data := c.DefaultQuery("data", time.Now().Format("2006-01-02"))

	relatorio, err := vendaService.RelatorioVendasDiarias(c.Request.Context(), lojaID.(int), data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, relatorio)
}

// ============================================
// ORÇAMENTO HANDLERS
// ============================================

func listarOrcamentosHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orcamentos, total, err := orcamentoRepo.ListarOrcamentos(lojaID.(int), status, nil, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": orcamentos,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getOrcamentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	orcamento, err := orcamentoRepo.BuscarOrcamentoPorID(id, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if orcamento == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Orçamento não encontrado"})
		return
	}
	c.JSON(http.StatusOK, orcamento)
}

func criarOrcamentoHandler(c *gin.Context) {
	var req struct {
		ClienteID  int     `json:"cliente_id"`
		Total      float64 `json:"total"`
		Desconto   float64 `json:"desconto"`
		Observacao string  `json:"observacao"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")

	orcamento := &models.Orcamento{
		ClienteID:  req.ClienteID,
		LojaID:     lojaID.(int),
		Total:      req.Total,
		Desconto:   req.Desconto,
		Observacao: req.Observacao,
		Status:     "pendente",
	}

	if err := orcamentoRepo.CriarOrcamento(orcamento); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Orçamento criado com sucesso", "orcamento": orcamento})
}

func atualizarOrcamentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req models.Orcamento
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	req.ID = id
	req.LojaID = lojaID.(int)

	if err := orcamentoRepo.AtualizarOrcamento(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento atualizado com sucesso"})
}

func aprovarOrcamentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	if err := orcamentoRepo.AprovarOrcamento(id, lojaID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento aprovado com sucesso"})
}

func rejeitarOrcamentoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	if err := orcamentoRepo.RejeitarOrcamento(id, lojaID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento rejeitado com sucesso"})
}

// ============================================
// RECEITA HANDLERS
// ============================================

func listarReceitasHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	receitas, err := receitaRepo.ListarReceitas(lojaID.(int), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": receitas,
		"total": len(receitas),
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getReceitaHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	receita, err := receitaRepo.BuscarReceitaPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if receita == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Receita não encontrada"})
		return
	}
	c.JSON(http.StatusOK, receita)
}

func criarReceitaHandler(c *gin.Context) {
	// Implementar criação de receita
	c.JSON(http.StatusCreated, gin.H{"message": "Receita criada com sucesso"})
}

func atualizarReceitaHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Receita atualizada com sucesso"})
}

func validarReceitaHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Observacao string `json:"observacao"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if err := receitaRepo.AtualizarStatus(id, "Validada", req.Observacao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Receita validada com sucesso"})
}

func rejeitarReceitaHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Observacao string `json:"observacao"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if err := receitaRepo.AtualizarStatus(id, "Rejeitada", req.Observacao); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Receita rejeitada com sucesso"})
}

// ============================================
// RECLAMAÇÃO HANDLERS
// ============================================

func listarReclamacoesHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	status := c.DefaultQuery("status", "")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	reclamacoes, total, err := reclamacaoRepo.ListarReclamacoes(lojaID.(int), status, "", limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": reclamacoes,
		"total": total,
		"page":  offset/limit + 1,
		"limit": limit,
	})
}

func getReclamacaoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	reclamacao, err := reclamacaoRepo.BuscarReclamacaoPorID(id, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if reclamacao == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reclamação não encontrada"})
		return
	}
	c.JSON(http.StatusOK, reclamacao)
}

func criarReclamacaoHandler(c *gin.Context) {
	var req models.Reclamacao
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	req.LojaID = lojaID.(int)

	if err := reclamacaoRepo.CriarReclamacao(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Reclamação registrada com sucesso"})
}

func atualizarReclamacaoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req models.Reclamacao
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	req.ID = id
	req.LojaID = lojaID.(int)

	if err := reclamacaoRepo.AtualizarReclamacao(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reclamação atualizada com sucesso"})
}

func resolverReclamacaoHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req struct {
		Resposta string `json:"resposta"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if err := reclamacaoRepo.ResolverReclamacao(id, lojaID.(int), req.Resposta); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reclamação resolvida com sucesso"})
}

// ============================================
// FORNECEDOR HANDLERS
// ============================================

func listarFornecedoresHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	fornecedores, err := fornecedorRepo.ListarFornecedores(lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, fornecedores)
}

func getFornecedorHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	fornecedor, err := fornecedorRepo.BuscarFornecedorPorID(id, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if fornecedor == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fornecedor não encontrado"})
		return
	}
	c.JSON(http.StatusOK, fornecedor)
}

func criarFornecedorHandler(c *gin.Context) {
	var req models.Fornecedor
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	lojaID, _ := c.Get("lojaID")
	req.LojaID = lojaID.(int)
	req.Ativo = true

	if err := fornecedorRepo.CriarFornecedor(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Fornecedor criado com sucesso"})
}

func atualizarFornecedorHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	var req models.Fornecedor
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	req.ID = id
	req.LojaID = lojaID.(int)

	if err := fornecedorRepo.AtualizarFornecedor(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fornecedor atualizado com sucesso"})
}

func deletarFornecedorHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	lojaID, _ := c.Get("lojaID")

	if err := fornecedorRepo.DeletarFornecedor(id, lojaID.(int)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fornecedor deletado com sucesso"})
}

// ============================================
// COMPRA HANDLERS COMPLETOS
// ============================================

// listarComprasHandler lista todas as compras
func listarComprasHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	
	compras, err := compraRepo.ListarCompras(lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Buscar fornecedores para cada compra
	var comprasCompletas []map[string]interface{}
	for _, compra := range compras {
		fornecedor, err := fornecedorRepo.BuscarFornecedorPorID(compra.FornecedorID, lojaID.(int))
		if err != nil {
			continue
		}
		
		compraCompleta := map[string]interface{}{
			"id":            compra.ID,
			"numero_nf":     compra.NumeroNF,
			"fornecedor_id": compra.FornecedorID,
			"fornecedor":    fornecedor.RazaoSocial,
			"total":         compra.Total,
			"status":        compra.Status,
			"data_emissao":  compra.DataEmissao.Format("2006-01-02"),
			"created_at":    compra.CreatedAt,
		}
		
		if compra.DataRecebimento != nil {
			compraCompleta["data_recebimento"] = compra.DataRecebimento.Format("2006-01-02 15:04:05")
		}
		
		comprasCompletas = append(comprasCompletas, compraCompleta)
	}
	
	c.JSON(http.StatusOK, comprasCompletas)
}

// getCompraHandler busca uma compra específica
func getCompraHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	
	compra, err := compraRepo.BuscarCompraPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if compra == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Compra não encontrada"})
		return
	}
	
	// Buscar fornecedor
	fornecedor, err := fornecedorRepo.BuscarFornecedorPorID(compra.FornecedorID, compra.LojaID)
	if err != nil {
		// Continua sem fornecedor
	}
	
	// Buscar itens com produtos
	var itensCompletos []map[string]interface{}
	for _, item := range compra.Itens {
		produto, err := produtoRepo.BuscarProdutoPorID(item.ProdutoID, compra.LojaID)
		if err != nil {
			continue
		}
		
		itensCompletos = append(itensCompletos, map[string]interface{}{
			"id":            item.ID,
			"produto_id":    item.ProdutoID,
			"produto_nome":  produto.Nome,
			"produto_codigo": produto.CodigoBarras,
			"quantidade":    item.Quantidade,
			"preco_unit":    item.PrecoUnit,
			"total":         item.Total,
		})
	}
	
	response := map[string]interface{}{
		"id":             compra.ID,
		"numero_nf":      compra.NumeroNF,
		"fornecedor_id":  compra.FornecedorID,
		"fornecedor":     fornecedor,
		"data_emissao":   compra.DataEmissao.Format("2006-01-02"),
		"status":         compra.Status,
		"subtotal":       compra.Subtotal,
		"desconto":       compra.Desconto,
		"total":          compra.Total,
		"observacao":     compra.Observacao,
		"itens":          itensCompletos,
		"created_at":     compra.CreatedAt,
	}
	
	if compra.DataRecebimento != nil {
		response["data_recebimento"] = compra.DataRecebimento.Format("2006-01-02 15:04:05")
	}
	
	c.JSON(http.StatusOK, response)
}

// criarCompraHandler cria uma nova compra
func criarCompraHandler(c *gin.Context) {
	var req struct {
		FornecedorID int `json:"fornecedor_id"`
		NumeroNF     string `json:"numero_nf"`
		DataEmissao  string `json:"data_emissao"`
		Desconto     float64 `json:"desconto"`
		Observacao   string `json:"observacao"`
		Itens        []struct {
			ProdutoID  int `json:"produto_id"`
			Quantidade int `json:"quantidade"`
		} `json:"itens"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}
	
	lojaID, _ := c.Get("lojaID")
	
	// Validar fornecedor
	fornecedor, err := fornecedorRepo.BuscarFornecedorPorID(req.FornecedorID, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if fornecedor == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fornecedor não encontrado"})
		return
	}
	
	// Validar itens
	if len(req.Itens) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Compra deve ter pelo menos um item"})
		return
	}
	
	// Parse data
	dataEmissao, err := time.Parse("2006-01-02", req.DataEmissao)
	if err != nil {
		dataEmissao = time.Now()
	}
	
	// Criar compra
	compra := &models.Compra{
		LojaID:       lojaID.(int),
		FornecedorID: req.FornecedorID,
		NumeroNF:     req.NumeroNF,
		DataEmissao:  dataEmissao,
		Desconto:     req.Desconto,
		Observacao:   req.Observacao,
		Status:       "Em Andamento",
	}
	
	// Criar itens
	var itens []models.ItemCompra
	var subtotal float64
	
	for _, itemReq := range req.Itens {
		// Validar produto
		produto, err := produtoRepo.BuscarProdutoPorID(itemReq.ProdutoID, lojaID.(int))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if produto == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Produto %d não encontrado", itemReq.ProdutoID)})
			return
		}
		
		item := models.ItemCompra{
			ProdutoID:  itemReq.ProdutoID,
			Quantidade: itemReq.Quantidade,
			PrecoUnit:  produto.PrecoCusto,
			Total:      produto.PrecoCusto * float64(itemReq.Quantidade),
		}
		
		subtotal += item.Total
		itens = append(itens, item)
	}
	
	compra.Subtotal = subtotal
	compra.Total = subtotal - req.Desconto
	compra.Itens = itens
	
	// Salvar compra
	if err := compraRepo.CriarCompra(compra); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Compra criada com sucesso",
		"compra":  compra,
	})
}

// atualizarCompraHandler atualiza uma compra
func atualizarCompraHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	
	lojaID, _ := c.Get("lojaID")
	
	// Buscar compra
	compra, err := compraRepo.BuscarCompraPorID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if compra == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Compra não encontrada"})
		return
	}
	
	if compra.LojaID != lojaID.(int) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Compra não pertence a esta loja"})
		return
	}
	
	if compra.Status == "Recebido" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Compra já foi recebida e não pode ser alterada"})
		return
	}
	
	var req struct {
		NumeroNF   string  `json:"numero_nf"`
		DataEmissao string `json:"data_emissao"`
		Desconto   float64 `json:"desconto"`
		Observacao string  `json:"observacao"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}
	
	// Atualizar campos
	if req.NumeroNF != "" {
		compra.NumeroNF = req.NumeroNF
	}
	if req.DataEmissao != "" {
		if data, err := time.Parse("2006-01-02", req.DataEmissao); err == nil {
			compra.DataEmissao = data
		}
	}
	compra.Desconto = req.Desconto
	compra.Total = compra.Subtotal - req.Desconto
	if req.Observacao != "" {
		compra.Observacao = req.Observacao
	}
	
	// Atualizar compra
	if err := compraRepo.AtualizarCompra(compra); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Compra atualizada com sucesso",
		"compra":  compra,
	})
}

// ============================================
// FINANCEIRO HANDLERS
// ============================================

func listarContasReceberHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Listar contas a receber"})
}

func listarContasPagarHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Listar contas a pagar"})
}

func criarContaReceberHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Conta a receber criada com sucesso"})
}

func criarContaPagarHandler(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Conta a pagar criada com sucesso"})
}

func baixarContaHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Conta baixada com sucesso"})
}

// ============================================
// IA HANDLERS
// ============================================

func listarProdutosIA(c *gin.Context) {
	ctx := context.Background()
	produtos, err := iaService.ListarProdutos(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, produtos)
}

func processarOrcamentoIA(c *gin.Context) {
	var req services.OrcamentoRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	ctx := context.Background()
	resp, err := iaService.ProcessarOrcamento(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func getIACacheStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"message": "Cache do IA Service",
	})
}

func clearIACache(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Cache do IA Service limpo com sucesso"})
}

// ============================================
// USUÁRIO HANDLERS
// ============================================

func listarUsuariosHandler(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, nome, email, role, ativo, loja_id, created_at
		FROM usuarios
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var usuarios []gin.H
	for rows.Next() {
		var id int
		var nome, email, role string
		var ativo bool
		var lojaID sql.NullInt64
		var createdAt time.Time

		if err := rows.Scan(&id, &nome, &email, &role, &ativo, &lojaID, &createdAt); err != nil {
			continue
		}

		usuario := gin.H{
			"id":         id,
			"nome":       nome,
			"email":      email,
			"role":       role,
			"ativo":      ativo,
			"created_at": createdAt,
		}
		if lojaID.Valid {
			usuario["loja_id"] = int(lojaID.Int64)
		}
		usuarios = append(usuarios, usuario)
	}

	c.JSON(http.StatusOK, usuarios)
}

func getUsuarioHandler(c *gin.Context) {
	id := c.Param("id")
	var usuario gin.H
	var lojaID sql.NullInt64

	err := db.QueryRow(`
		SELECT id, nome, email, role, ativo, loja_id, created_at
		FROM usuarios
		WHERE id = ?
	`, id).Scan(
		usuario["id"], usuario["nome"], usuario["email"],
		usuario["role"], usuario["ativo"], lojaID, usuario["created_at"],
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if lojaID.Valid {
		usuario["loja_id"] = int(lojaID.Int64)
	}

	c.JSON(http.StatusOK, usuario)
}

func criarUsuarioHandler(c *gin.Context) {
	var req struct {
		Nome   string `json:"nome"`
		Email  string `json:"email"`
		Senha  string `json:"password"`
		Role   string `json:"role"`
		LojaID *int   `json:"loja_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if req.Role == "" {
		req.Role = "atendente"
	}

	var lojaID interface{}
	if req.LojaID != nil {
		lojaID = *req.LojaID
	} else {
		lojaID = nil
	}

	result, err := db.Exec(`
		INSERT INTO usuarios (nome, email, senha_hash, role, loja_id)
		VALUES (?, ?, ?, ?, ?)
	`, req.Nome, req.Email, req.Senha, req.Role, lojaID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Usuário criado com sucesso"})
}

func atualizarUsuarioHandler(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Nome   string `json:"nome"`
		Email  string `json:"email"`
		Senha  string `json:"password,omitempty"`
		Role   string `json:"role"`
		LojaID *int   `json:"loja_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	query := `UPDATE usuarios SET nome = ?, email = ?, role = ?, loja_id = ?`
	args := []interface{}{req.Nome, req.Email, req.Role, req.LojaID}

	if req.Senha != "" {
		query += `, senha_hash = ?`
		args = append(args, req.Senha)
	}

	query += ` WHERE id = ?`
	args = append(args, id)

	_, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuário atualizado com sucesso"})
}

func toggleUsuarioHandler(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Ativo bool `json:"ativo"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`UPDATE usuarios SET ativo = ? WHERE id = ?`, req.Ativo, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status do usuário atualizado"})
}

func deletarUsuarioHandler(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`DELETE FROM usuarios WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuário deletado com sucesso"})
}

// ============================================
// CONFIGURAÇÃO HANDLERS
// ============================================

func listarConfiguracoesHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	configs, err := configRepo.ListarConfiguracoes(lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]string)
	for _, cfg := range configs {
		result[cfg.Chave] = cfg.Valor
	}

	c.JSON(http.StatusOK, result)
}

func atualizarConfiguracoesHandler(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var configs map[string]string
	if err := c.BindJSON(&configs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	for chave, valor := range configs {
		if err := configRepo.AtualizarConfiguracao(lojaID.(int), chave, valor); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	cacheInvalidation.InvalidateByPrefix(c.Request.Context(), "configuracoes")

	c.JSON(http.StatusOK, gin.H{"message": "Configurações atualizadas com sucesso"})
}

// ============================================
// WHATSAPP HANDLERS
// ============================================

func webhookWhatsAppVerify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == os.Getenv("WHATSAPP_VERIFY_TOKEN") {
		c.String(http.StatusOK, challenge)
		return
	}
	c.String(http.StatusForbidden, "Verificação falhou")
}

func webhookWhatsApp(c *gin.Context) {
	// Implementar webhook do WhatsApp
	c.Status(http.StatusOK)
}

// ============================================
// HEALTH CHECK
// ============================================

func healthCheck(c *gin.Context) {
	// Verificar banco
	var dbStatus string
	if err := db.Ping(); err != nil {
		dbStatus = "unhealthy"
	} else {
		dbStatus = "healthy"
	}

	// Verificar Redis
	var redisStatus string
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		redisStatus = "unhealthy"
	} else {
		redisStatus = "healthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "saas-backend",
		"version": "2.0.0",
		"checks": gin.H{
			"database": dbStatus,
			"redis":    redisStatus,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}