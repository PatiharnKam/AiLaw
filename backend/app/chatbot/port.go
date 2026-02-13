package chatbot

import (
	"context"

	"github.com/PatiharnKam/AiLaw/app"
)

type Service interface {
	CreateChatSessionService(ctx context.Context, req CreateChatSessionRequest) (*CreateChatSessionResponse, error)
	// ChatbotProcess(ctx context.Context, req ChatbotProcessRequest) (*GetMessageResponse, error)
	ChatbotProcess(ctx context.Context, req ChatbotProcessRequest) (app.Response, error)
	// ChatbotProcessWithStream(ctx context.Context, req ChatbotProcessRequest, onChunk StreamCallback) (*StreamingMessageResponse, error)
	ChatbotProcessWithStream(ctx context.Context, req ChatbotProcessRequest, onChunk StreamCallback) (app.Response, error)
}

type Storage interface {
	CreateSession(ctx context.Context, req CreateChatSessionRequest) (*CreateChatSessionResponse, error)
	UpdateLastMessageAt(ctx context.Context, userId string, sessionId string) error
	SaveUserMessage(ctx context.Context, userId, sessionId, modelMessageId, userMessage string, userPromptTokens int) error
	SaveModelMessage(ctx context.Context, userId, sessionId, modelMessageId string, modelDetail ModelMessageDetail) error
}

type CreateChatSessionRequest struct {
	UserId string `json:"userId" validate:"required,uuid"`
	Title  string `json:"title"`
}

type CreateChatSessionResponse struct {
	SessionId string `json:"sessionId"`
}

type ModelMessageDetail struct {
	ModelType         string  `json:"modelType"`
	Content           string  `json:"message"`
	Feedback          *string `json:"feedback"`
	TotalInputTokens  int     `json:"totalInputTokens"`
	TotalOutputTokens int     `json:"totalOutputTokens"`
	FinalOutputTokens int     `json:"finalOutputTokens"`
	TotalUsedTokens   int     `json:"totalUsedTokens"`
	ResponseTime      *int    `json:"responseTime"`
}

type GetMessageResponse struct {
	Message        string `json:"message"`
	ModelMessageID string `json:"modelMessageID"`
}

type ChatbotProcessRequest struct {
	UserId    string `json:"userId" validate:"required,uuid"`
	SessionId string `json:"sessionId" binding:"required" validate:"required,uuid"`
	ModelType string `json:"modelType" binding:"required"`
	Input     Input  `json:"input"`
}

type Input struct {
	Messages Messages `json:"messages"`
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content" binding:"required"`
}

type ChatbotRequest struct {
	Messages []Messages `json:"messages"`
}

type ChatbotResponse struct {
	Role              string `json:"role"`
	Content           string `json:"content"`
	Desicion          string `json:"desicion"`
	Memory            Memory `json:"memory"`
	TotalInputTokens  int    `json:"totalInputTokens"`
	TotalOutputTokens int    `json:"totalOutputTokens"`
	FinalOutputTokens int    `json:"finalOutputTokens"`
	TotalUsedTokens   int    `json:"totalUsedTokens"`
}

type Memory struct {
	Agent    string `json:"agent"`
	Sections string `json:"sections"`
}
//========================== Web Socker ========================================

// WebSocket Message Types
type WSMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId,omitempty"`
	Content   string `json:"content,omitempty"`
	ModelType string `json:"modelType,omitempty"` // "default" or "COT"
}

type WSResponse struct {
	Type           string `json:"type"`
	Content        string `json:"content,omitempty"`
	SessionID      string `json:"sessionId,omitempty"`
	ModelMessageID string `json:"modelMessageId,omitempty"`

	// COT specific fields
	Steps       []string `json:"steps,omitempty"`
	Rationale   string   `json:"rationale,omitempty"`
	CurrentStep int      `json:"currentStep,omitempty"`
	TotalSteps  int      `json:"totalSteps,omitempty"`
	StepDesc    string   `json:"stepDescription,omitempty"`
	Status      string   `json:"status,omitempty"`

	// Error fields
	Error *WSError `json:"error,omitempty"`
}

type WSError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// StreamCallback is called for each chunk
type StreamCallback func(chunk StreamEvent)

// StreamEvent represents different event types from SSE
type StreamEvent struct {
	Type string `json:"type"`

	// For content chunks
	Text string `json:"text,omitempty"`

	// For status updates
	Message string `json:"message,omitempty"`

	// For COT plan
	Steps     []string `json:"steps,omitempty"`
	Rationale string   `json:"rationale,omitempty"`

	// For COT step progress
	Step        int    `json:"step,omitempty"`
	Total       int    `json:"total,omitempty"`
	Description string `json:"description,omitempty"`

	// For completion
	TotalInputTokens  int    `json:"totalInputTokens,omitempty"`
	TotalOutputTokens int    `json:"totalOutputTokens,omitempty"`
	FinalOutputTokens int    `json:"finalOutputTokens,omitempty"`
	TotalUsedTokens   int    `json:"totalUsedTokens,omitempty"`
	FullContent       string `json:"fullContent,omitempty"`

	// For errors
	Error string `json:"error,omitempty"`
}

// StreamingMessageResponse for streaming completion
type StreamingMessageResponse struct {
	Message        string `json:"message"`
	ModelMessageID string `json:"modelMessageId"`
}
