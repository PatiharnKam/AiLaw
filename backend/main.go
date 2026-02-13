// cmd/main.go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PatiharnKam/AiLaw/app/auth"
	service "github.com/PatiharnKam/AiLaw/app/chatbot"
	deleteChatSession "github.com/PatiharnKam/AiLaw/app/delete_session"
	feedback "github.com/PatiharnKam/AiLaw/app/feedback"
	messageshistory "github.com/PatiharnKam/AiLaw/app/messages_history"
	"github.com/PatiharnKam/AiLaw/app/quota"
	sessionshistory "github.com/PatiharnKam/AiLaw/app/sessions_history"
	updateSessionName "github.com/PatiharnKam/AiLaw/app/update_session_name"
	"github.com/PatiharnKam/AiLaw/config"
	"github.com/PatiharnKam/AiLaw/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	gracefulShutdownDuration = 10 * time.Second
	serverReadHeaderTimeout  = 300 * time.Second
	serverReadTimeout        = 300 * time.Second
	serverWriteTimeout       = 300 * time.Second
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		slog.Error("unable to parse specific config", " error :", err.Error())
		return
	}

	corsConfig := cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigin,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})
	r := gin.New()
	r.Use(
		gin.Recovery(),
		corsConfig,
		middleware.LoggerMiddleware(),
	)
	r.GET("/health", health())

	db, err := config.NewPostgresDB(cfg.Database.PostgresURL, config.DBConnectionConfig{
		ConnMaxLifetime:   &cfg.Database.PostgresConnMaxLifetime,
		ConnMaxIdleTime:   &cfg.Database.PostgresConnMaxIdleTime,
		MaxOpenConns:      &cfg.Database.PostgresMaxOpenConns,
		HealthCheckPeriod: &cfg.Database.PostgresHealthCheckPeriod,
	})
	defer db.Close()
	if err != nil {
		slog.Error("Failed to connect to Postgres", "error", err.Error())
		return
	}

	redisClient, err := config.NewRedisClient(cfg.Redis)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err.Error())
	}
	defer redisClient.Close()

	quotaService := quota.NewQuotaService(redisClient, &cfg.Quota)

	api := r.Group("/api")
	api.Use(middleware.GinJWTMiddleware(cfg))
	{
		{
			getMessageHistoryStorage := messageshistory.NewStorage(db)
			getMessageHistoryService := messageshistory.NewService(getMessageHistoryStorage)
			getMessageHistoryHandler := messageshistory.NewHandler(getMessageHistoryService)
			api.GET("/messages-history/:sessionID", getMessageHistoryHandler.GetMessageHistory)
		}

		{
			getSessionHistoryStorage := sessionshistory.NewStorage(db)
			getSessionHistoryService := sessionshistory.NewService(getSessionHistoryStorage)
			getSessionHistoryHandler := sessionshistory.NewHandler(getSessionHistoryService)
			api.GET("/sessions-history", getSessionHistoryHandler.GetSessionHistory)
		}

		{
			deleteChatSessionStorage := deleteChatSession.NewStorage(db)
			deleteChatSessionService := deleteChatSession.NewService(deleteChatSessionStorage)
			deleteChatSessionHandler := deleteChatSession.NewHandler(deleteChatSessionService)
			api.DELETE("/session/:sessionID", deleteChatSessionHandler.DeleteChatSessionHandler)
		}

		{
			updateSessionNameStorage := updateSessionName.NewStorage(db)
			updateSessionNameService := updateSessionName.NewService(updateSessionNameStorage)
			updateSessionNameHandler := updateSessionName.NewHandler(updateSessionNameService)
			api.PATCH("/name/session/:sessionID", updateSessionNameHandler.UpdateSessionNameHandler)
		}

		{
			createChatSessionStorage := service.NewStorage(db)
			createChatSessionService := service.NewService(cfg, createChatSessionStorage, quotaService)
			createChatSessionHandler := service.NewHandler(createChatSessionService, cfg)
			api.POST("/session", createChatSessionHandler.CreateChatSessionHandler)
		}

		{
			getMessageStorage := service.NewStorage(db)
			getMessageService := service.NewService(cfg, getMessageStorage, quotaService)
			getMessageHandler := service.NewHandler(getMessageService, cfg)
			api.POST("/model", getMessageHandler.ChatbotProcessModelHandler)
			api.GET("/ws", getMessageHandler.WebSocketHandler)
		}

		{
			feedbackStorage := feedback.NewStorage(db)
			feedbackService := feedback.NewService(feedbackStorage)
			feedbackHandler := feedback.NewHandler(feedbackService)
			api.PATCH("/feedback/:messageID", feedbackHandler.FeedbackHandler)
		}

	}

	{
		authStorage := auth.NewStorage(db)
		authService := auth.NewService(cfg, authStorage)
		authHandler := auth.NewHandler(authService)
		r.GET("/auth/google/login", authHandler.GoogleLogin)
		r.GET("/auth/google/callback", authHandler.GoogleCallback)
		r.POST("/auth/refresh", authHandler.RefreshTokenProcess)
		r.POST("/auth/logout", authHandler.Logout)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           r,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	go gracefully(srv)

	slog.Info("run at : " + cfg.Server.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("HTTP server ListenAndServe: " + err.Error())
		return
	}
}

func gracefully(srv *http.Server) {
	{
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		<-ctx.Done()
	}

	d := time.Duration(gracefulShutdownDuration)
	slog.Info(fmt.Sprintf("shutting down in %d ... \n", d))

	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Info("HTTP Server Shutdown: " + err.Error())
	}
}

func health() func(c *gin.Context) {
	h, err := os.Hostname()
	if err != nil {
		h = fmt.Sprintf("unknown host err: %s", err.Error())
	}
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"hostname": h,
			// "version":  strings.ReplaceAll(version, "\n", ""),
			// "commit":   commit,
		})
	}
}
