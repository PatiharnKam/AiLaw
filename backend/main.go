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

	"github.com/PatiharnKam/AiLaw/app/service"
	"github.com/PatiharnKam/AiLaw/config"
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

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}

		c.Next()
	})

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

	model := client.GenerativeModel("gemini-1.5-flash")

	model.GenerationConfig = genai.GenerationConfig{
		Temperature:     genai.Ptr(float32(0.7)),
		TopK:            genai.Ptr(int32(40)),
		TopP:            genai.Ptr(float32(0.95)),
		MaxOutputTokens: genai.Ptr(int32(1024)),
	}

	{
		getMessageService := service.NewService(cfg, client, model)
		getMessageHandler := service.NewHandler(getMessageService)
		r.POST("/message", getMessageHandler.GetMessage)
	}

	{
		getMessageService := service.NewService(cfg, client, model)
		getMessageHandler := service.NewHandler(getMessageService)
		r.POST("/message/model", getMessageHandler.GetMessageModel)
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
