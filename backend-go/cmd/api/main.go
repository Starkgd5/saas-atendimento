package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	socketio "github.com/googollee/go-socket.io"
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
	socketServer     *socketio.Server
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

	// Configurar pool de conexões
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Testar conexão com retry
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		if i < maxRetries-1 {
			zap.L().Warn("Falha ao pingar MariaDB, tentando novamente...",
				zap.Int("tentativa", i+1),
				zap.Error(err))
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	}
	if err != nil {
		zap.L().Fatal("Erro ao pingar MariaDB após retries", zap.Error(err))
	}
	zap.L().Info("✅ Conectado ao MariaDB",
		zap.Int("max_open_conns", cfg.DB.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.DB.MaxIdleConns),
		zap.Duration("conn_max_lifetime", cfg.DB.ConnMaxLifetime),
	)

	// Adicionar função para verificar saúde da conexão
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := db.Ping(); err != nil {
				zap.L().Error("Perda de conexão com o banco", zap.Error(err))
				// Tentar reconectar
				db.Close()
				db, err = sql.Open("mysql", cfg.DB.DSN)
				if err != nil {
					zap.L().Error("Falha ao reconectar", zap.Error(err))
					continue
				}
				db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
				db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
				db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
				db.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)
				if err = db.Ping(); err == nil {
					zap.L().Info("✅ Reconectado ao MariaDB")
				}
			}
		}
	}()

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

	// Inicializar WebSocket Manager (Gorilla)
	wsManager = websocket.NewManager()
	go wsManager.Run()
	zap.L().Info("✅ WebSocket Manager iniciado")

	// Inicializar Socket.IO Server
	socketServer, err = initSocketIOServer()
	if err != nil {
		zap.L().Fatal("Erro ao iniciar Socket.IO", zap.Error(err))
	}
	zap.L().Info("✅ Socket.IO Server iniciado")

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

	// Desligar Socket.IO
	if socketServer != nil {
		socketServer.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Erro no shutdown", zap.Error(err))
	}

	zap.L().Info("✅ Servidor desligado com sucesso")
}

// ============================================
// SOCKET.IO SERVER
// ============================================

func initSocketIOServer() (*socketio.Server, error) {
	server := socketio.NewServer(nil)

	// Conexão
	server.OnConnect("/", func(s socketio.Conn) error {
		zap.L().Info("🔌 Cliente Socket.IO conectado", zap.String("id", s.ID()))

		// Extrair token da query
		url := s.URL()
		query := url.Query()
		token := query.Get("token")

		// Validar token
		if token != "" {
			claims, err := jwtService.ValidateToken(token)
			if err == nil {
				s.SetContext(map[string]interface{}{
					"user_id": claims.UserID,
					"loja_id": claims.LojaID,
					"role":    claims.Role,
				})
				zap.L().Info("✅ Token validado", zap.Int("user_id", claims.UserID))
			} else {
				zap.L().Warn("⚠️ Token inválido", zap.Error(err))
			}
		}

		return nil
	})

	// Desconexão
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		zap.L().Info("🔌 Cliente Socket.IO desconectado", zap.String("id", s.ID()), zap.String("reason", reason))
	})

	// Evento: nova_mensagem
	server.OnEvent("/", "nova_mensagem", func(s socketio.Conn, msg map[string]interface{}) {
		zap.L().Info("📩 Nova mensagem recebida", zap.Any("msg", msg))

		// Reenviar para todos os clientes na sala da loja
		lojaID := 1
		if id, ok := msg["loja_id"]; ok {
			if val, ok := id.(float64); ok {
				lojaID = int(val)
			}
		}

		room := getRoomName(lojaID)
		server.BroadcastToRoom("/", room, "nova_mensagem", msg)

		// Também enviar via Gorilla WebSocket
		payload, _ := json.Marshal(msg)
		wsManager.EnviarParaClientes(lojaID, payload)
	})

	// Evento: puxar_cliente
	server.OnEvent("/", "puxar_cliente", func(s socketio.Conn, data interface{}) {
		zap.L().Info("🎯 Puxar cliente solicitado")

		// Processar a fila
		ctx := context.Background()
		clienteID, err := filaService.ProximoClienteAtendimento(ctx)
		if err != nil {
			zap.L().Error("Erro ao puxar cliente", zap.Error(err))
			s.Emit("error", map[string]interface{}{
				"message": "Erro ao puxar cliente",
				"error":   err.Error(),
			})
			return
		}

		if clienteID == 0 {
			s.Emit("fila_vazia", map[string]interface{}{
				"message": "Fila vazia",
			})
			return
		}

		// Buscar cliente
		cliente, err := clienteRepo.BuscarClientePorID(clienteID)
		if err != nil {
			zap.L().Error("Erro ao buscar cliente", zap.Error(err))
			return
		}

		// Notificar todos
		msg := map[string]interface{}{
			"cliente_id": clienteID,
			"nome":       cliente.Nome,
			"telefone":   cliente.Telefone,
			"status":     "em_atendimento",
		}

		room := getRoomName(1)
		server.BroadcastToRoom("/", room, "fila_atualizada", msg)

		// Também enviar via Gorilla
		payload, _ := json.Marshal(map[string]interface{}{
			"type":    "fila_atualizada",
			"payload": msg,
		})
		wsManager.EnviarParaClientes(1, payload)
	})

	// Evento: finalizar_atendimento
	server.OnEvent("/", "atendimento_finalizado", func(s socketio.Conn, data map[string]interface{}) {
		zap.L().Info("✅ Atendimento finalizado", zap.Any("data", data))

		clienteID := 0
		if id, ok := data["cliente_id"]; ok {
			if val, ok := id.(float64); ok {
				clienteID = int(val)
			}
		}

		if clienteID > 0 {
			// Finalizar na fila
			ctx := context.Background()
			filaService.FinalizarAtendimento(ctx, clienteID)

			// Notificar todos
			msg := map[string]interface{}{
				"cliente_id": clienteID,
				"status":     "finalizado",
			}

			room := getRoomName(1)
			server.BroadcastToRoom("/", room, "atendimento_finalizado", msg)
		}
	})

	// Evento: typing
	server.OnEvent("/", "typing", func(s socketio.Conn, data map[string]interface{}) {
		lojaID := 1
		if id, ok := data["loja_id"]; ok {
			if val, ok := id.(float64); ok {
				lojaID = int(val)
			}
		}

		room := getRoomName(lojaID)
		server.BroadcastToRoom("/", room, "typing", data)
	})

	// Evento: join_room
	server.OnEvent("/", "join_room", func(s socketio.Conn, room string) {
		if room != "" {
			s.Join(room)
			zap.L().Info("📥 Cliente entrou na sala", zap.String("room", room), zap.String("id", s.ID()))
		}
	})

	// Evento: leave_room
	server.OnEvent("/", "leave_room", func(s socketio.Conn, room string) {
		if room != "" {
			s.Leave(room)
			zap.L().Info("📤 Cliente saiu da sala", zap.String("room", room), zap.String("id", s.ID()))
		}
	})

	// Evento: ping
	server.OnEvent("/", "ping", func(s socketio.Conn) {
		s.Emit("pong", map[string]interface{}{
			"time": time.Now().Unix(),
		})
	})

	go func() {
		if err := server.Serve(); err != nil {
			zap.L().Error("Erro no Socket.IO server", zap.Error(err))
		}
	}()

	return server, nil
}

func getRoomName(lojaID int) string {
	return "loja_" + string(rune(lojaID))
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

	// Socket.IO - rotas compatíveis
	router.GET("/socket.io/*any", gin.WrapH(socketServer))
	router.POST("/socket.io/*any", gin.WrapH(socketServer))
	router.OPTIONS("/socket.io/*any", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// WebSocket nativo (Gorilla) - para compatibilidade
	router.GET("/ws", wsHandler)

	api := router.Group("/api/v1")
	{
		// Rotas públicas
		api.GET("/webhook/whatsapp", webhookWhatsAppVerify)
		api.POST("/webhook/whatsapp", webhookWhatsApp)
		api.POST("/auth/login", authLogin)
		api.POST("/auth/refresh", authRefresh)
		api.POST("/auth/register", registerUser)

		// Rotas protegidas (JWT)
		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			// Dashboard
			auth.GET("/dashboard", getDashboard)
			auth.GET("/dashboard/metricas", getMetricasDetalhadas)
			auth.GET("/fila/status", getFilaStatus)
			auth.GET("/fila/clientes", getFilaClientes)
			auth.POST("/fila/proximo", puxarProximoCliente)
			auth.PUT("/fila/config", configurarLimite)
			auth.GET("/clientes", listarClientes)
			auth.GET("/clientes/:id", getCliente)
			auth.POST("/clientes", criarCliente)
			auth.PUT("/clientes/:id", atualizarCliente)
			auth.GET("/atendimentos", listarAtendimentos)
			auth.GET("/atendimentos/:id", getAtendimento)
			auth.POST("/atendimentos/:id/finalizar", finalizarAtendimento)
			auth.POST("/atendimentos/:id/enviar", enviarMensagem)
			auth.GET("/mensagens", listarMensagens)
			auth.GET("/mensagens/:id", getMensagem)
			auth.GET("/produtos", listarProdutos)
			auth.GET("/produtos/:id", getProduto)
			auth.POST("/produtos", criarProduto)
			auth.PUT("/produtos/:id", atualizarProduto)
			auth.DELETE("/produtos/:id", deletarProduto)
			auth.GET("/orcamentos", listarOrcamentos)
			auth.GET("/orcamentos/:id", getOrcamento)
			auth.POST("/orcamentos", criarOrcamento)
			auth.PUT("/orcamentos/:id", atualizarOrcamento)
			auth.POST("/orcamentos/:id/aprovar", aprovarOrcamento)
			auth.POST("/orcamentos/:id/rejeitar", rejeitarOrcamento)
			auth.GET("/ia/produtos", listarProdutosIA)
			auth.POST("/ia/orcamento", processarOrcamentoIA)
			auth.GET("/usuarios", listarUsuarios)
			auth.GET("/usuarios/:id", getUsuario)
			auth.POST("/usuarios", criarUsuario)
			auth.PUT("/usuarios/:id", atualizarUsuario)
			auth.PATCH("/usuarios/:id/toggle", toggleUsuario)
			auth.DELETE("/usuarios/:id", deletarUsuario)
			auth.GET("/reclamacoes", listarReclamacoes)
			auth.GET("/reclamacoes/:id", getReclamacao)
			auth.POST("/reclamacoes", criarReclamacao)
			auth.PUT("/reclamacoes/:id", atualizarReclamacao)
			auth.POST("/reclamacoes/:id/resolver", resolverReclamacao)
			auth.GET("/configuracoes", listarConfiguracoes)
			auth.PUT("/configuracoes", atualizarConfiguracao)
		}
	}

	return router
}

// ============ HANDLERS BÁSICOS ============

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"service": "saas-atendimento",
		"version": "1.0.0",
	})
}

func wsHandler(c *gin.Context) {
	userID, _ := strconv.Atoi(c.Query("user_id"))
	if userID == 0 {
		userID = 1
	}
	lojaID, _ := strconv.Atoi(c.Query("loja_id"))
	if lojaID == 0 {
		lojaID = 1
	}
	wsManager.ServeWS(c.Writer, c.Request, userID, lojaID)
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

	// Verificar se o banco está conectado
	if db == nil {
		zap.L().Error("Banco de dados não está conectado")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno no servidor"})
		return
	}

	// Buscar usuário no banco
	var userID int
	var nome, role, senhaHash string
	var lojaID sql.NullInt64

	query := `
		SELECT id, nome, role, senha_hash, loja_id 
		FROM usuarios 
		WHERE email = ? AND ativo = 1
	`

	err := db.QueryRow(query, req.Email).Scan(&userID, &nome, &role, &senhaHash, &lojaID)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não encontrado ou inativo"})
			return
		}
		zap.L().Error("Erro ao buscar usuário", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao processar login"})
		return
	}

	// Verificar senha
	// Em produção, use bcrypt.CompareHashAndPassword
	// Para demo, comparamos o hash diretamente
	validPassword := false

	// Se a senha hash for igual à senha fornecida
	if senhaHash == req.Password {
		validPassword = true
	}

	// Senhas padrão para demo (caso o hash não corresponda)
	if !validPassword {
		defaultPasswords := map[string]string{
			"admin@saas.com":     "admin123",
			"gerente@saas.com":   "gerente123",
			"atendente@saas.com": "atendente123",
		}

		if defaultPass, ok := defaultPasswords[req.Email]; ok && defaultPass == req.Password {
			validPassword = true
		}
	}

	if !validPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Senha incorreta"})
		return
	}

	// Definir lojaID
	lojaIDInt := 0
	if lojaID.Valid {
		lojaIDInt = int(lojaID.Int64)
	}

	// Gerar token
	token, err := jwtService.GenerateTokenWithDetails(userID, lojaIDInt, role, req.Email, nome)
	if err != nil {
		zap.L().Error("Erro ao gerar token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
		return
	}

	// Registrar login
	zap.L().Info("Login realizado com sucesso",
		zap.Int("user_id", userID),
		zap.String("email", req.Email),
		zap.String("role", role),
	)

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

	// Hash da senha (simplificado)
	senhaHash := req.Senha
	if req.Role == "" {
		req.Role = "atendente"
	}

	_, err := db.Exec(`
		INSERT INTO usuarios (nome, email, senha_hash, role) 
		VALUES (?, ?, ?, ?)
	`, req.Nome, req.Email, senhaHash, req.Role)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Usuário registrado com sucesso"})
}

// ============ DASHBOARD ============

func getDashboard(c *gin.Context) {
	// Pegar lojaID do contexto
	lojaID, exists := c.Get("lojaID")
	if !exists {
		lojaID = 1
	}

	ctx := context.Background()

	// Verificar se o banco está conectado
	if db == nil {
		zap.L().Error("Banco de dados não está conectado")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Banco de dados não disponível"})
		return
	}

	// Buscar métricas
	metrics, err := dashboardService.GetMetrics(ctx, lojaID.(int))
	if err != nil {
		zap.L().Error("Erro ao buscar métricas", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Buscar status da fila
	filaStatus, err := filaService.GetStatusFila(ctx)
	if err != nil {
		zap.L().Warn("Erro ao buscar status da fila", zap.Error(err))
		// Não falha a requisição, apenas não inclui a fila
		c.JSON(http.StatusOK, gin.H{
			"metrics": metrics,
		})
		return
	}

	// Buscar métricas adicionais
	metricasDiarias, err := dashboardService.GetMetricasDiarias(ctx, lojaID.(int), 7)
	if err != nil {
		zap.L().Warn("Erro ao buscar métricas diárias", zap.Error(err))
		metricasDiarias = []map[string]interface{}{}
	}

	satisfacao, err := dashboardService.GetSatisfacaoCliente(ctx, lojaID.(int))
	if err != nil {
		zap.L().Warn("Erro ao buscar satisfação", zap.Error(err))
		satisfacao = map[string]interface{}{
			"total_reclamacoes": 0,
			"resolvidas":        0,
			"taxa_resolucao":    0,
			"score":             100,
		}
	}

	crescimento, err := dashboardService.GetCrescimentoMensal(ctx, lojaID.(int))
	if err != nil {
		zap.L().Warn("Erro ao buscar crescimento", zap.Error(err))
		crescimento = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics":          metrics,
		"fila":             filaStatus,
		"metricas_diarias": metricasDiarias,
		"satisfacao":       satisfacao,
		"crescimento":      crescimento,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// ============ HANDLER PARA MÉTRICAS DETALHADAS ============

func getMetricasDetalhadas(c *gin.Context) {
	lojaID, exists := c.Get("lojaID")
	if !exists {
		lojaID = 1
	}

	ctx := context.Background()

	// Buscar métricas por hora
	metricasPorHora, err := dashboardService.GetMetricasPorHora(ctx, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Buscar métricas diárias (30 dias)
	metricasDiarias, err := dashboardService.GetMetricasDiarias(ctx, lojaID.(int), 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Buscar satisfação
	satisfacao, err := dashboardService.GetSatisfacaoCliente(ctx, lojaID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Buscar relatório completo
	relatorio, err := dashboardService.GetRelatorioCompleto(ctx, lojaID.(int), "mensal")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metricas_por_hora": metricasPorHora,
		"metricas_diarias":  metricasDiarias,
		"satisfacao":        satisfacao,
		"relatorio":         relatorio,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
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

func getFilaClientes(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")

	// Buscar clientes em atendimento e espera
	rows, err := db.Query(`
		SELECT c.id, c.nome, c.telefone, a.status, a.iniciado_em
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		WHERE a.loja_id = ? AND a.status IN ('aguardando', 'em_atendimento')
		ORDER BY a.iniciado_em ASC
	`, lojaID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var clientes []gin.H
	for rows.Next() {
		var id int
		var nome, telefone, status string
		var iniciadoEm time.Time

		if err := rows.Scan(&id, &nome, &telefone, &status, &iniciadoEm); err != nil {
			continue
		}

		clientes = append(clientes, gin.H{
			"id":          id,
			"nome":        nome,
			"telefone":    telefone,
			"status":      status,
			"iniciado_em": iniciadoEm,
		})
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

	// Buscar dados do cliente
	var nome, telefone string
	err = db.QueryRow(`
		SELECT nome, telefone FROM clientes WHERE id = ?
	`, clienteID).Scan(&nome, &telefone)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notificar via WebSocket
	msg, _ := json.Marshal(gin.H{
		"type": "fila_atualizada",
		"payload": gin.H{
			"cliente_id": clienteID,
			"nome":       nome,
			"telefone":   telefone,
		},
	})
	wsManager.EnviarParaClientes(1, msg)

	c.JSON(http.StatusOK, gin.H{
		"cliente_id": clienteID,
		"nome":       nome,
		"telefone":   telefone,
	})
}

func configurarLimite(c *gin.Context) {
	var req struct {
		Limite int `json:"limite"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limite inválido"})
		return
	}

	if req.Limite < 1 || req.Limite > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limite deve ser entre 1 e 10"})
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
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	query := `
		SELECT id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE loja_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.Query(query, lojaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var clientes []gin.H
	for rows.Next() {
		var id int
		var nome, telefone, email string
		var ultimoAtendimento sql.NullTime
		var createdAt time.Time

		if err := rows.Scan(&id, &nome, &telefone, &email, &ultimoAtendimento, &createdAt); err != nil {
			continue
		}

		cliente := gin.H{
			"id":         id,
			"nome":       nome,
			"telefone":   telefone,
			"email":      email,
			"created_at": createdAt,
		}
		if ultimoAtendimento.Valid {
			cliente["ultimo_atendimento"] = ultimoAtendimento.Time
		}
		clientes = append(clientes, cliente)
	}

	c.JSON(http.StatusOK, clientes)
}

func getCliente(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	var cliente models.Cliente
	var ultimoAtendimento sql.NullTime

	err := db.QueryRow(`
		SELECT id, loja_id, nome, telefone, email, ultimo_atendimento, created_at
		FROM clientes
		WHERE id = ? AND loja_id = ?
	`, id, lojaID).Scan(
		&cliente.ID, &cliente.LojaID, &cliente.Nome, &cliente.Telefone,
		&cliente.Email, &ultimoAtendimento, &cliente.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "cliente não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if ultimoAtendimento.Valid {
		cliente.UltimoAtendimento = &ultimoAtendimento.Time
	}

	c.JSON(http.StatusOK, cliente)
}

func criarCliente(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var req struct {
		Nome     string `json:"nome"`
		Telefone string `json:"telefone"`
		Email    string `json:"email"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	result, err := db.Exec(`
		INSERT INTO clientes (loja_id, nome, telefone, email)
		VALUES (?, ?, ?, ?)
	`, lojaID, req.Nome, req.Telefone, req.Email)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Cliente criado"})
}

func atualizarCliente(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")
	var req struct {
		Nome     string `json:"nome"`
		Telefone string `json:"telefone"`
		Email    string `json:"email"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`
		UPDATE clientes SET nome = ?, telefone = ?, email = ?
		WHERE id = ? AND loja_id = ?
	`, req.Nome, req.Telefone, req.Email, id, lojaID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente atualizado"})
}

// ============ ATENDIMENTOS ============

func listarAtendimentos(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.DefaultQuery("status", "")

	query := `
		SELECT a.id, a.cliente_id, c.nome, c.telefone, a.status, 
		       a.iniciado_em, a.finalizado_em, a.atendente_id, u.nome as atendente_nome
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		LEFT JOIN usuarios u ON a.atendente_id = u.id
		WHERE a.loja_id = ?
	`
	args := []interface{}{lojaID}

	if status != "" {
		query += " AND a.status = ?"
		args = append(args, status)
	}

	query += " ORDER BY a.iniciado_em DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var atendimentos []gin.H
	for rows.Next() {
		var id, clienteID, atendenteID sql.NullInt64
		var nome, telefone, status, atendenteNome string
		var iniciadoEm, finalizadoEm sql.NullTime

		if err := rows.Scan(&id, &clienteID, &nome, &telefone, &status,
			&iniciadoEm, &finalizadoEm, &atendenteID, &atendenteNome); err != nil {
			continue
		}

		atendimento := gin.H{
			"id":         id.Int64,
			"cliente_id": clienteID.Int64,
			"cliente":    nome,
			"telefone":   telefone,
			"status":     status,
		}
		if iniciadoEm.Valid {
			atendimento["iniciado_em"] = iniciadoEm.Time
		}
		if finalizadoEm.Valid {
			atendimento["finalizado_em"] = finalizadoEm.Time
		}
		if atendenteID.Valid {
			atendimento["atendente_id"] = atendenteID.Int64
			atendimento["atendente"] = atendenteNome
		}
		atendimentos = append(atendimentos, atendimento)
	}

	c.JSON(http.StatusOK, atendimentos)
}

func getAtendimento(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	var atendimento gin.H
	var clienteID, atendenteID sql.NullInt64
	var clienteNome, telefone, status, atendenteNome string
	var iniciadoEm, finalizadoEm sql.NullTime

	err := db.QueryRow(`
		SELECT a.id, a.cliente_id, c.nome, c.telefone, a.status,
		       a.iniciado_em, a.finalizado_em, a.atendente_id, u.nome
		FROM atendimentos a
		JOIN clientes c ON a.cliente_id = c.id
		LEFT JOIN usuarios u ON a.atendente_id = u.id
		WHERE a.id = ? AND a.loja_id = ?
	`, id, lojaID).Scan(
		&id, &clienteID, &clienteNome, &telefone, &status,
		&iniciadoEm, &finalizadoEm, &atendenteID, &atendenteNome,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "atendimento não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	atendimento = gin.H{
		"id":          id,
		"cliente_id":  clienteID.Int64,
		"cliente":     clienteNome,
		"telefone":    telefone,
		"status":      status,
		"iniciado_em": iniciadoEm.Time,
	}
	if finalizadoEm.Valid {
		atendimento["finalizado_em"] = finalizadoEm.Time
	}
	if atendenteID.Valid {
		atendimento["atendente_id"] = atendenteID.Int64
		atendimento["atendente"] = atendenteNome
	}

	c.JSON(http.StatusOK, atendimento)
}

func finalizarAtendimento(c *gin.Context) {
	id := c.Param("id")
	atendenteID, _ := c.Get("userID")

	_, err := db.Exec(`
		UPDATE atendimentos 
		SET status = 'finalizado', finalizado_em = NOW(), atendente_id = ?
		WHERE id = ?
	`, atendenteID, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Atendimento finalizado"})
}

func enviarMensagem(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Mensagem string `json:"mensagem"`
		Arquivo  string `json:"arquivo,omitempty"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mensagem inválida"})
		return
	}

	_, err := db.Exec(`
		INSERT INTO mensagens (atendimento_id, remetente, conteudo, tipo)
		VALUES (?, 'atendente', ?, 'texto')
	`, id, req.Mensagem)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mensagem enviada com sucesso"})
}

// ============ MENSAGENS ============

func listarMensagens(c *gin.Context) {
	atendimentoID := c.DefaultQuery("atendimento_id", "")
	if atendimentoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "atendimento_id é obrigatório"})
		return
	}

	rows, err := db.Query(`
		SELECT id, remetente, conteudo, tipo, arquivo_url, arquivo_nome, enviado_em, lida
		FROM mensagens
		WHERE atendimento_id = ?
		ORDER BY enviado_em ASC
	`, atendimentoID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var mensagens []gin.H
	for rows.Next() {
		var id int
		var remetente, conteudo, tipo, arquivoURL, arquivoNome string
		var enviadoEm time.Time
		var lida bool

		if err := rows.Scan(&id, &remetente, &conteudo, &tipo, &arquivoURL, &arquivoNome, &enviadoEm, &lida); err != nil {
			continue
		}

		mensagens = append(mensagens, gin.H{
			"id":           id,
			"remetente":    remetente,
			"conteudo":     conteudo,
			"tipo":         tipo,
			"arquivo_url":  arquivoURL,
			"arquivo_nome": arquivoNome,
			"enviado_em":   enviadoEm,
			"lida":         lida,
		})
	}

	c.JSON(http.StatusOK, mensagens)
}

func getMensagem(c *gin.Context) {
	id := c.Param("id")
	var mensagem gin.H

	var mid int
	var remetente, conteudo, tipo, arquivoURL, arquivoNome string
	var enviadoEm time.Time
	var lida bool

	err := db.QueryRow(`
		SELECT id, remetente, conteudo, tipo, arquivo_url, arquivo_nome, enviado_em, lida
		FROM mensagens
		WHERE id = ?
	`, id).Scan(&mid, &remetente, &conteudo, &tipo,
		&arquivoURL, &arquivoNome, &enviadoEm, &lida)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "mensagem não encontrada"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	mensagem = gin.H{
		"id":           mid,
		"remetente":    remetente,
		"conteudo":     conteudo,
		"tipo":         tipo,
		"arquivo_url":  arquivoURL,
		"arquivo_nome": arquivoNome,
		"enviado_em":   enviadoEm,
		"lida":         lida,
	}
	c.JSON(http.StatusOK, mensagem)
}

// ============ PRODUTOS ============

func listarProdutos(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	categoria := c.DefaultQuery("categoria", "")

	query := `
		SELECT id, nome, descricao, categoria, preco, estoque, estoque_min, 
		       requere_receita, ativo, created_at
		FROM produtos
		WHERE loja_id = ?
	`
	args := []interface{}{lojaID}

	if categoria != "" {
		query += " AND categoria = ?"
		args = append(args, categoria)
	}

	query += " ORDER BY nome ASC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var produtos []gin.H
	for rows.Next() {
		var id int
		var nome, descricao, categoria string
		var preco float64
		var estoque, estoqueMin int
		var requereReceita, ativo bool
		var createdAt time.Time

		if err := rows.Scan(&id, &nome, &descricao, &categoria, &preco, &estoque, &estoqueMin,
			&requereReceita, &ativo, &createdAt); err != nil {
			continue
		}

		produtos = append(produtos, gin.H{
			"id":              id,
			"nome":            nome,
			"descricao":       descricao,
			"categoria":       categoria,
			"preco":           preco,
			"estoque":         estoque,
			"estoque_min":     estoqueMin,
			"requere_receita": requereReceita,
			"ativo":           ativo,
			"created_at":      createdAt,
		})
	}

	c.JSON(http.StatusOK, produtos)
}

func getProduto(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	var produto gin.H
	var descricao, nome, categoria string
	var preco float64
	var estoque, estoqueMin int
	var requereReceita, ativo bool
	var createdAt time.Time

	err := db.QueryRow(`
		SELECT nome, descricao, categoria, preco, estoque, estoque_min, 
		       requere_receita, ativo, created_at
		FROM produtos
		WHERE id = ? AND loja_id = ?
	`, id, lojaID).Scan(
		&nome, &descricao, &categoria, &preco, &estoque, &estoqueMin,
		&requereReceita, &ativo, &createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "produto não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	produto = gin.H{
		"id":              id,
		"nome":            nome,
		"descricao":       descricao,
		"categoria":       categoria,
		"preco":           preco,
		"estoque":         estoque,
		"estoque_min":     estoqueMin,
		"requere_receita": requereReceita,
		"ativo":           ativo,
		"created_at":      createdAt,
	}

	c.JSON(http.StatusOK, produto)
}

func criarProduto(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var req struct {
		Nome           string  `json:"nome"`
		Descricao      string  `json:"descricao"`
		Categoria      string  `json:"categoria"`
		Preco          float64 `json:"preco"`
		Estoque        int     `json:"estoque"`
		EstoqueMin     int     `json:"estoque_min"`
		RequereReceita bool    `json:"requere_receita"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	result, err := db.Exec(`
		INSERT INTO produtos (loja_id, nome, descricao, categoria, preco, estoque, estoque_min, requere_receita)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, lojaID, req.Nome, req.Descricao, req.Categoria, req.Preco, req.Estoque, req.EstoqueMin, req.RequereReceita)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Produto criado com sucesso"})
}

func atualizarProduto(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")
	var req struct {
		Nome           string  `json:"nome"`
		Descricao      string  `json:"descricao"`
		Categoria      string  `json:"categoria"`
		Preco          float64 `json:"preco"`
		Estoque        int     `json:"estoque"`
		EstoqueMin     int     `json:"estoque_min"`
		RequereReceita bool    `json:"requere_receita"`
		Ativo          bool    `json:"ativo"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`
		UPDATE produtos SET 
			nome = ?, descricao = ?, categoria = ?, preco = ?, 
			estoque = ?, estoque_min = ?, requere_receita = ?, ativo = ?
		WHERE id = ? AND loja_id = ?
	`, req.Nome, req.Descricao, req.Categoria, req.Preco,
		req.Estoque, req.EstoqueMin, req.RequereReceita, req.Ativo, id, lojaID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Produto atualizado com sucesso"})
}

func deletarProduto(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	_, err := db.Exec(`
		DELETE FROM produtos WHERE id = ? AND loja_id = ?
	`, id, lojaID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Produto excluído com sucesso"})
}

// ============ ORÇAMENTOS ============

func listarOrcamentos(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.DefaultQuery("status", "")

	query := `
		SELECT o.id, o.cliente_id, c.nome, o.total, o.status, o.created_at, o.expirado_em
		FROM orcamentos o
		JOIN clientes c ON o.cliente_id = c.id
		WHERE o.loja_id = ?
	`
	args := []interface{}{lojaID}

	if status != "" {
		query += " AND o.status = ?"
		args = append(args, status)
	}

	query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var orcamentos []gin.H
	for rows.Next() {
		var id, clienteID int
		var nome, status string
		var total float64
		var createdAt, expiradoEm sql.NullTime

		if err := rows.Scan(&id, &clienteID, &nome, &total, &status, &createdAt, &expiradoEm); err != nil {
			continue
		}

		orcamento := gin.H{
			"id":         id,
			"cliente_id": clienteID,
			"cliente":    nome,
			"total":      total,
			"status":     status,
		}
		if createdAt.Valid {
			orcamento["created_at"] = createdAt.Time
		}
		if expiradoEm.Valid {
			orcamento["expirado_em"] = expiradoEm.Time
		}
		orcamentos = append(orcamentos, orcamento)
	}

	c.JSON(http.StatusOK, orcamentos)
}

func getOrcamento(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	var orcamento models.Orcamento
	var clienteNome string
	var expiradoEm sql.NullTime

	err := db.QueryRow(`
		SELECT o.id, o.cliente_id, c.nome, o.total, o.desconto, o.total_com_desconto,
		       o.observacao, o.status, o.created_at, o.expirado_em
		FROM orcamentos o
		JOIN clientes c ON o.cliente_id = c.id
		WHERE o.id = ? AND o.loja_id = ?
	`, id, lojaID).Scan(
		&orcamento.ID, &orcamento.ClienteID, &clienteNome,
		&orcamento.Total, &orcamento.Desconto, &orcamento.TotalComDesconto,
		&orcamento.Observacao, &orcamento.Status, &orcamento.CreatedAt, &expiradoEm,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "orçamento não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if expiradoEm.Valid {
		orcamento.ExpiradoEm = &expiradoEm.Time
	}

	// Buscar itens do orçamento
	rows, err := db.Query(`
		SELECT id, produto_id, produto_nome, quantidade, preco_unit, total
		FROM orcamento_itens
		WHERE orcamento_id = ?
	`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item models.OrcamentoItem
			if err := rows.Scan(&item.ID, &item.ProdutoID, &item.ProdutoNome,
				&item.Quantidade, &item.PrecoUnit, &item.Total); err == nil {
				orcamento.Itens = append(orcamento.Itens, item)
			}
		}
	}

	// Adicionar cliente_nome ao response
	c.JSON(http.StatusOK, gin.H{
		"orcamento":    orcamento,
		"cliente_nome": clienteNome,
	})
}

func criarOrcamento(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var req struct {
		ClienteID int `json:"cliente_id"`
		Produtos  []struct {
			ID         int `json:"id"`
			Quantidade int `json:"quantidade"`
		} `json:"produtos"`
		Observacao string `json:"observacao"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	// Calcular total
	var total float64
	for _, p := range req.Produtos {
		var preco float64
		err := db.QueryRow(`
			SELECT preco FROM produtos WHERE id = ? AND loja_id = ?
		`, p.ID, lojaID).Scan(&preco)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "produto não encontrado: " + strconv.Itoa(p.ID)})
			return
		}
		total += preco * float64(p.Quantidade)
	}

	// Inserir orçamento
	result, err := db.Exec(`
		INSERT INTO orcamentos (cliente_id, loja_id, total, observacao, status)
		VALUES (?, ?, ?, ?, 'pendente')
	`, req.ClienteID, lojaID, total, req.Observacao)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	orcamentoID, _ := result.LastInsertId()

	// Inserir itens
	for _, p := range req.Produtos {
		var preco float64
		var produtoNome string
		err := db.QueryRow(`
			SELECT preco, nome FROM produtos WHERE id = ? AND loja_id = ?
		`, p.ID, lojaID).Scan(&preco, &produtoNome)

		if err == nil {
			db.Exec(`
				INSERT INTO orcamento_itens (orcamento_id, produto_id, produto_nome, quantidade, preco_unit, total)
				VALUES (?, ?, ?, ?, ?, ?)
			`, orcamentoID, p.ID, produtoNome, p.Quantidade, preco, preco*float64(p.Quantidade))
		}
	}

	c.JSON(http.StatusCreated, gin.H{"id": orcamentoID, "total": total, "message": "Orçamento criado com sucesso"})
}

func atualizarOrcamento(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Observacao string  `json:"observacao"`
		Desconto   float64 `json:"desconto"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`
		UPDATE orcamentos SET observacao = ?, desconto = ?, total_com_desconto = total - ?
		WHERE id = ?
	`, req.Observacao, req.Desconto, req.Desconto, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento atualizado com sucesso"})
}

func aprovarOrcamento(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`
		UPDATE orcamentos SET status = 'aprovado', updated_at = NOW()
		WHERE id = ?
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento aprovado com sucesso"})
}

func rejeitarOrcamento(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec(`
		UPDATE orcamentos SET status = 'rejeitado', updated_at = NOW()
		WHERE id = ?
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Orçamento rejeitado"})
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

	var usuarios []gin.H
	for rows.Next() {
		var id int
		var nome, email, role, lojaNome string
		var ativo bool
		var lojaID sql.NullInt64
		var createdAt time.Time

		if err := rows.Scan(&id, &nome, &email, &role, &ativo, &lojaID, &lojaNome, &createdAt); err != nil {
			continue
		}

		usuario := gin.H{
			"id":         id,
			"nome":       nome,
			"email":      email,
			"role":       role,
			"ativo":      ativo,
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

func getUsuario(c *gin.Context) {
	id := c.Param("id")
	var lojaID sql.NullInt64
	var (
		uid      int
		nome     string
		email    string
		role     string
		ativo    bool
		lojaNome string
	)

	err := db.QueryRow(`
		SELECT u.id, u.nome, u.email, u.role, u.ativo, u.loja_id, COALESCE(l.nome, '') as loja_nome
		FROM usuarios u
		LEFT JOIN lojas l ON u.loja_id = l.id
		WHERE u.id = ?
	`, id).Scan(&uid, &nome, &email, &role, &ativo, &lojaID, &lojaNome)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "usuário não encontrado"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	usuario := gin.H{
		"id":        uid,
		"nome":      nome,
		"email":     email,
		"role":      role,
		"ativo":     ativo,
		"loja_nome": lojaNome,
	}
	if lojaID.Valid {
		usuario["loja_id"] = int(lojaID.Int64)
	}

	c.JSON(http.StatusOK, usuario)
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
	lojaID, _ := c.Get("lojaID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	status := c.DefaultQuery("status", "")

	query := `
		SELECT r.id, r.cliente_id, c.nome, r.mensagem, r.status, r.prioridade, 
		       r.categoria, r.created_at, r.resolvido_em
		FROM reclamacoes r
		JOIN clientes c ON r.cliente_id = c.id
		WHERE r.loja_id = ?
	`
	args := []interface{}{lojaID}

	if status != "" {
		query += " AND r.status = ?"
		args = append(args, status)
	}

	query += " ORDER BY r.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var reclamacoes []gin.H
	for rows.Next() {
		var id, clienteID int
		var nome, mensagem, status, prioridade, categoria string
		var createdAt, resolvidoEm sql.NullTime

		if err := rows.Scan(&id, &clienteID, &nome, &mensagem, &status,
			&prioridade, &categoria, &createdAt, &resolvidoEm); err != nil {
			continue
		}

		reclamacao := gin.H{
			"id":         id,
			"cliente_id": clienteID,
			"cliente":    nome,
			"mensagem":   mensagem,
			"status":     status,
			"prioridade": prioridade,
			"categoria":  categoria,
		}
		if createdAt.Valid {
			reclamacao["created_at"] = createdAt.Time
		}
		if resolvidoEm.Valid {
			reclamacao["resolvido_em"] = resolvidoEm.Time
		}
		reclamacoes = append(reclamacoes, reclamacao)
	}

	c.JSON(http.StatusOK, reclamacoes)
}

func getReclamacao(c *gin.Context) {
	id := c.Param("id")
	lojaID, _ := c.Get("lojaID")

	var id_, clienteID, cliente, mensagem, status, prioridade, categoria, resposta interface{}
	var createdAt, resolvidoEm sql.NullTime

	err := db.QueryRow(`
		SELECT r.id, r.cliente_id, c.nome, r.mensagem, r.status, r.prioridade,
		       r.categoria, r.resposta, r.created_at, r.resolvido_em
		FROM reclamacoes r
		JOIN clientes c ON r.cliente_id = c.id
		WHERE r.id = ? AND r.loja_id = ?
	`, id, lojaID).Scan(
		&id_, &clienteID, &cliente,
		&mensagem, &status, &prioridade,
		&categoria, &resposta, &createdAt, &resolvidoEm,
	)

	reclamacao := gin.H{
		"id":         id_,
		"cliente_id": clienteID,
		"cliente":    cliente,
		"mensagem":   mensagem,
		"status":     status,
		"prioridade": prioridade,
		"categoria":  categoria,
		"resposta":   resposta,
	}

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "reclamação não encontrada"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	if createdAt.Valid {
		reclamacao["created_at"] = createdAt.Time
	}
	if resolvidoEm.Valid {
		reclamacao["resolvido_em"] = resolvidoEm.Time
	}

	c.JSON(http.StatusOK, reclamacao)
}

func criarReclamacao(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var req struct {
		ClienteID  int    `json:"cliente_id"`
		Mensagem   string `json:"mensagem"`
		Prioridade string `json:"prioridade"`
		Categoria  string `json:"categoria"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	if req.Prioridade == "" {
		req.Prioridade = "media"
	}

	result, err := db.Exec(`
		INSERT INTO reclamacoes (cliente_id, loja_id, mensagem, prioridade, categoria, status)
		VALUES (?, ?, ?, ?, ?, 'pendente')
	`, req.ClienteID, lojaID, req.Mensagem, req.Prioridade, req.Categoria)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Reclamação registrada com sucesso"})
}

func atualizarReclamacao(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status   string `json:"status"`
		Resposta string `json:"resposta"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`
		UPDATE reclamacoes SET status = ?, resposta = ?, updated_at = NOW()
		WHERE id = ?
	`, req.Status, req.Resposta, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reclamação atualizada com sucesso"})
}

func resolverReclamacao(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Resposta string `json:"resposta"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	_, err := db.Exec(`
		UPDATE reclamacoes SET status = 'resolvido', resposta = ?, resolvido_em = NOW(), updated_at = NOW()
		WHERE id = ?
	`, req.Resposta, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Reclamação resolvida com sucesso"})
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

	// Abandonos
	var abandonos int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND status = 'abandonado'
	`, lojaID).Scan(&abandonos)

	// Atendimentos hoje
	var atendimentosHoje int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM atendimentos WHERE loja_id = ? AND DATE(iniciado_em) = CURDATE()
	`, lojaID).Scan(&atendimentosHoje)

	// Orçamentos hoje
	var orcamentosHoje int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orcamentos WHERE loja_id = ? AND DATE(created_at) = CURDATE()
	`, lojaID).Scan(&orcamentosHoje)

	// Reclamações pendentes
	var reclamacoesPendentes int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reclamacoes WHERE loja_id = ? AND status = 'pendente'
	`, lojaID).Scan(&reclamacoesPendentes)

	c.JSON(http.StatusOK, gin.H{
		"tempo_medio_atendimento": tempoMedioAtendimento,
		"total_finalizados":       totalFinalizados,
		"abandonos":               abandonos,
		"taxa_abandono":           float64(abandonos) / float64(totalFinalizados+abandonos+1) * 100,
		"atendimentos_hoje":       atendimentosHoje,
		"orcamentos_hoje":         orcamentosHoje,
		"reclamacoes_pendentes":   reclamacoesPendentes,
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

// ============ CONFIGURAÇÕES ============

func listarConfiguracoes(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")

	rows, err := db.Query(`
		SELECT chave, valor, descricao FROM configuracoes WHERE loja_id = ?
	`, lojaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	configs := make(map[string]string)
	for rows.Next() {
		var chave, valor, descricao string
		if err := rows.Scan(&chave, &valor, &descricao); err != nil {
			continue
		}
		configs[chave] = valor
	}

	c.JSON(http.StatusOK, configs)
}

func atualizarConfiguracao(c *gin.Context) {
	lojaID, _ := c.Get("lojaID")
	var req map[string]string
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dados inválidos"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	for chave, valor := range req {
		_, err := tx.Exec(`
			INSERT INTO configuracoes (loja_id, chave, valor) VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE valor = ?
		`, lojaID, chave, valor, valor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configurações atualizadas com sucesso"})
}

// ============ MIDDLEWARES ============

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
