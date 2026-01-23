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
	updateSessionName "github.com/PatiharnKam/AiLaw/app/update_session_name"
	feedback "github.com/PatiharnKam/AiLaw/app/feedback"
	messageshistory "github.com/PatiharnKam/AiLaw/app/messages_history"
	sessionshistory "github.com/PatiharnKam/AiLaw/app/sessions_history"
	"github.com/PatiharnKam/AiLaw/config"
	database "github.com/PatiharnKam/AiLaw/db"
	"github.com/PatiharnKam/AiLaw/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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
	r := gin.New()
	r.Use(gin.Recovery())

	corsConfig := cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	})

	r.Use(corsConfig)

	db, err := database.NewPostgresDB(cfg.Database.PostgresURL, database.DBConnectionConfig{
		ConnMaxLifetime:   &cfg.Database.PostgresConnMaxLifetime,
		ConnMaxIdleTime:   &cfg.Database.PostgresConnMaxIdleTime,
		MaxOpenConns:      &cfg.Database.PostgresMaxOpenConns,
		HealthCheckPeriod: &cfg.Database.PostgresHealthCheckPeriod,
	})
	defer db.Close()

	r.GET("/health", health())

	apiKey := cfg.APIkey.GeminiAPIkey
	if apiKey == "" {
		slog.Error("GEMINI_API_KEY environment variable is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		slog.Error("Failed to create Gemini client ", "error :", err.Error())
	}

	model := client.GenerativeModel("gemini-2.5-flash")

	model.GenerationConfig = genai.GenerationConfig{
		Temperature:     genai.Ptr(float32(0.7)),
		TopK:            genai.Ptr(int32(40)),
		TopP:            genai.Ptr(float32(0.95)),
		MaxOutputTokens: genai.Ptr(int32(1024)),
	}

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
			createChatSessionService := service.NewService(cfg, createChatSessionStorage, client)
			createChatSessionHandler := service.NewHandler(createChatSessionService)
			api.POST("/session", createChatSessionHandler.CreateChatSessionHandler)
		}

		{
			getMessageStorage := service.NewStorage(db)
			getMessageService := service.NewService(cfg, getMessageStorage, client)
			getMessageHandler := service.NewHandler(getMessageService)
			api.POST("/model", getMessageHandler.ChatbotProcessModelHandler)
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
		r.GET("/token", authHandler.GetToken)
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
