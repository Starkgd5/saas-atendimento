package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Starkgd5/saas-atendimento/internal/config"
	"github.com/Starkgd5/saas-atendimento/internal/models"
	"github.com/Starkgd5/saas-atendimento/internal/repository"
	"github.com/Starkgd5/saas-atendimento/internal/services"
	"github.com/Starkgd5/saas-atendimento/internal/websocket"
)

// Variáveis globais
var (
	db               *sql.DB
	redisClient      *redis.Client
	filaService      *services.FilaService
	wsManager        *websocket.Manager
	clienteRepo      *repository.ClienteRepository
	atendimentoRepo  *repository.AtendimentoRepository
	dashboardService *services.DashboardService
	jwtService       *services.JWTService
	whatsappService  *services.WhatsAppService
	iaService        *services.IAService
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

	// Conectar ao Redis
	redisClient = redis.NewClient(&redis.Options{
		Addr: cfg.Redis.URL,
		DB:   0,
	})

	if err = redisClient.Ping(context.Background()).Err(); err != nil {
		zap.L().Fatal("Erro ao conectar ao Redis", zap.Error(err))
	}
	zap.L().Info("✅ Conectado ao Redis")

	// Inicializar repositórios
	clienteRepo = repository.NewClienteRepository(db)
	atendimentoRepo = repository.NewAtendimentoRepository(db)

	// Inicializar serviços
	filaService = services.NewFilaService(redisClient, cfg.MaxClients)
	jwtService = services.NewJWTService(cfg.JWTSecret)
	whatsappService = services.NewWhatsAppService(&cfg.WhatsApp)
	dashboardService = services.NewDashboardService(db)
	iaService = services.NewIAService(os.Getenv("IA_URL"))

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

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health Check
	router.GET("/health", healthCheck)

	// WebSocket
	router.GET("/ws", wsHandler)

	api := router.Group("/api/v1")
	{
		// Rotas públicas
		api.GET("/webhook/whatsapp", webhookWhatsAppVerify)
		api.POST("/webhook/whatsapp", webhookWhatsApp)
		api.POST("/auth/login", authLogin)
		api.POST("/auth/refresh", authRefresh)

		// Rotas protegidas (JWT)
		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			auth.GET("/dashboard", getDashboard)
			auth.GET("/fila/status", getFilaStatus)
			auth.POST("/fila/proximo", puxarProximoCliente)
			auth.PUT("/fila/config", configurarLimite)
			auth.GET("/clientes", listarClientes)
			auth.GET("/clientes/:id", getCliente)
			auth.GET("/atendimentos", listarAtendimentos)
			auth.POST("/atendimentos/:id/enviar", enviarMensagem)
			// Adicionar no setupRouter() dentro de api.Group
			// Rotas de IA
			auth.GET("/ia/produtos", listarProdutosIA)
			auth.POST("/ia/orcamento", processarOrcamentoIA)

			// Rotas de Usuários (Gestor)
			auth.GET("/usuarios", listarUsuarios)
			auth.POST("/usuarios", criarUsuario)
			auth.PUT("/usuarios/:id", atualizarUsuario)
			auth.PATCH("/usuarios/:id/toggle", toggleUsuario)
			auth.DELETE("/usuarios/:id", deletarUsuario)

			// Rotas de Reclamações
			auth.GET("/reclamacoes", listarReclamacoes)
			auth.PUT("/reclamacoes/:id", atualizarReclamacao)

			// Rotas de Atendimentos (para métricas)
			auth.GET("/atendimentos/metricas", getMetricasAtendimento)
		}
	}

	return router
}

// ============ HANDLERS BÁSICOS ============

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func wsHandler(c *gin.Context) {
	userID := 1
	lojaID := 1
	wsManager.ServeWS(c.Writer, c.Request, userID, lojaID)
}

// ============ IA HANDLERS ============

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

// ============ USUÁRIOS (GESTOR) ============

func listarUsuarios(c *gin.Context) {
	rows, err := db.Query(`
		SELECT u.id, u.nome, u.email, u.role, u.ativo, u.loja_id, 
		       COALESCE(l.nome, '') as loja_nome, u.created_at
		FROM usuarios u
		LEFT JOIN lojas l ON u.loja_id = l.id
		ORDER BY u.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var usuarios []map[string]interface{}
	for rows.Next() {
		var id int
		var nome, email, role, lojaNome string
		var ativo bool
		var lojaID sql.NullInt64
		var createdAt string

		if err := rows.Scan(&id, &nome, &email, &role, &ativo, &lojaID, &lojaNome, &createdAt); err != nil {
			continue
		}

		usuario := map[string]interface{}{
			"id":         id,
			"nome":       nome,
			"email":      email,
			"role":       role,
			"ativo":      ativo,
			"loja_id":    nil,
			"loja_nome":  lojaNome,
			"created_at": createdAt,
		}
		if lojaID.Valid {
			usuario["loja_id"] = int(lojaID.Int64)
		}
		usuarios = append(usuarios, usuario)
	}

	c.JSON(http.StatusOK, usuarios)
}

func criarUsuario(c *gin.Context) {
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

	// Hash da senha (simplificado - use bcrypt em produção)
	// Aqui usamos um hash simples para demonstração
	senhaHash := req.Senha // TODO: Implementar hash real

	var lojaID interface{}
	if req.LojaID != nil {
		lojaID = *req.LojaID
	} else {
		lojaID = nil
	}

	query := `INSERT INTO usuarios (nome, email, senha_hash, role, loja_id) VALUES (?, ?, ?, ?, ?)`
	result, err := db.Exec(query, req.Nome, req.Email, senhaHash, req.Role, lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Usuário criado com sucesso"})
}

func atualizarUsuario(c *gin.Context) {
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
		args = append(args, req.Senha) // TODO: Hash
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

func toggleUsuario(c *gin.Context) {
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

func deletarUsuario(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`DELETE FROM usuarios WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuário excluído com sucesso"})
}

// ============ RECLAMAÇÕES ============

func listarReclamacoes(c *gin.Context) {
	// Simulação - em produção, buscar do banco
	reclamacoes := []map[string]interface{}{
		{
			"id":         1,
			"cliente":    "João Silva",
			"mensagem":   "Produto com atraso na entrega",
			"status":     "pendente",
			"created_at": time.Now().Add(-2 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			"id":         2,
			"cliente":    "Maria Santos",
			"mensagem":   "Medicamento com preço diferente do informado",
			"status":     "resolvido",
			"created_at": time.Now().Add(-5 * time.Hour).Format("2006-01-02 15:04:05"),
		},
	}
	c.JSON(http.StatusOK, reclamacoes)
}

func atualizarReclamacao(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	// TODO: Atualizar no banco
	c.JSON(http.StatusOK, gin.H{"id": id, "status": req.Status, "message": "Reclamação atualizada"})
}

// ============ MÉTRICAS DE ATENDIMENTO ============

func getMetricasAtendimento(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	ctx := context.Background()

	// Tempo médio de atendimento
	var tempoMedio sql.NullFloat64
	if err := db.QueryRowContext(ctx, `
		SELECT AVG(TIMESTAMPDIFF(SECOND, iniciado_em, finalizado_em))
		FROM atendimentos
		WHERE loja_id = ? AND status = 'finalizado' AND finalizado_em IS NOT NULL
	`, lojaID).Scan(&tempoMedio); err != nil && err != sql.ErrNoRows {
		zap.L().Error("erro ao calcular tempo medio de atendimento", zap.Error(err))
	}

	tempoMedioAtendimento := 0.0
	if tempoMedio.Valid {
		tempoMedioAtendimento = tempoMedio.Float64
	}

	// Total de atendimentos finalizados
	var totalFinalizados int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = 'finalizado'
	`, lojaID).Scan(&totalFinalizados)

	// Abandonos (clientes que saíram da fila)
	var abandonos int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = 'abandonado'
	`, lojaID).Scan(&abandonos)

	c.JSON(http.StatusOK, gin.H{
		"tempo_medio_atendimento": tempoMedioAtendimento,
		"total_finalizados":       totalFinalizados,
		"abandonos":               abandonos,
		"taxa_abandono":           float64(abandonos) / float64(totalFinalizados+abandonos) * 100,
	})
}

// ============ WEBHOOK WHATSAPP ============

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
	var webhook services.WhatsAppWebhook
	if err := c.BindJSON(&webhook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages, err := whatsappService.ProcessWebhook(&webhook)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	for _, msg := range messages {
		go processWhatsAppMessage(ctx, msg)
	}

	c.Status(http.StatusOK)
}

func processWhatsAppMessage(ctx context.Context, msg services.WebhookMessage) {
	zap.L().Info("Mensagem recebida",
		zap.String("from", msg.From),
		zap.String("type", msg.Type),
	)

	cliente, err := clienteRepo.BuscarClientePorTelefone(msg.From)
	if err != nil {
		zap.L().Error("Erro ao buscar cliente", zap.Error(err))
		return
	}

	if cliente == nil {
		cliente = &models.Cliente{
			LojaID:   1,
			Nome:     "Cliente WhatsApp",
			Telefone: msg.From,
		}
		if err := clienteRepo.CriarCliente(cliente); err != nil {
			zap.L().Error("Erro ao criar cliente", zap.Error(err))
			return
		}
	}

	if err := filaService.AdicionarClienteFila(ctx, cliente.ID, cliente.LojaID); err != nil {
		zap.L().Error("Erro ao adicionar à fila", zap.Error(err))
	}

	notificacao, _ := json.Marshal(gin.H{
		"type": "nova_mensagem",
		"payload": gin.H{
			"cliente_id": cliente.ID,
			"cliente":    cliente.Nome,
			"mensagem":   msg.Text,
		},
	})
	wsManager.EnviarParaClientes(cliente.LojaID, notificacao)
}

// ============ AUTENTICAÇÃO ============

func authLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if req.Email == "admin@saas.com" && req.Password == "admin123" {
		token, err := jwtService.GenerateToken(1, 1, "admin")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"id":   1,
				"nome": "Admin Master",
				"role": "admin",
			},
		})
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciais inválidas"})
}

func authRefresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": "new-token"})
}

// ============ DASHBOARD ============

func getDashboard(c *gin.Context) {
	lojaID, exists := c.Get("lojaID")
	if !exists {
		lojaID = 1
	}
	ctx := context.Background()

	metrics, err := dashboardService.GetMetrics(ctx, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filaStatus, err := filaService.GetStatusFila(ctx)
	if err == nil {
		// Adicionar fila status ao metrics (precisa ser adicionado no models)
		c.JSON(http.StatusOK, gin.H{
			"metrics": metrics,
			"fila":    filaStatus,
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// ============ FILA ============

func getFilaStatus(c *gin.Context) {
	ctx := context.Background()
	status, err := filaService.GetStatusFila(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
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

	msg, _ := json.Marshal(gin.H{
		"type":    "fila_atualizada",
		"payload": gin.H{"cliente_id": clienteID},
	})
	wsManager.EnviarParaClientes(1, msg)

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

// ============ CLIENTES ============

func listarClientes(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{})
}

func getCliente(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
}

// ============ ATENDIMENTOS ============

func listarAtendimentos(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{})
}

func enviarMensagem(c *gin.Context) {
	var req struct {
		Mensagem string `json:"mensagem"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mensagem inválida"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mensagem enviada"})
}

// ============ MIDDLEWARES ============

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

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
