package service

import "context"

type Service interface {
	CreateChatSessionService(ctx context.Context, req CreateChatSessionRequest) (CreateChatSessionResponse, error)
	ChatbotProcess(ctx context.Context, req ChatbotProcessRequest) (*GetMessageResponse, error)
}

type Storage interface {
	CreateSession(ctx context.Context, req CreateChatSessionRequest) (*CreateChatSessionResponse, error)
	UpdateLastMessageAt(ctx context.Context, userId string, sessionId string) error
	SaveUserMessage(ctx context.Context, sessionId, userMessage string, promptTokens int) error
	SaveModelMessage(ctx context.Context, sessionId string, modelDetail ModelMessageDetail) (string, error)
}

type CreateChatSessionRequest struct {
	UserId string `json:"userId" validate:"required,uuid"`
	Title  string `json:"title"`
}

type CreateChatSessionResponse struct {
	SessionId string `json:"sessionId"`
}

type ModelMessageDetail struct {
	Content          string  `json:"message"`
	Feedback         *string `json:"feedback"`
	PromptTokens     *int    `json:"promptTokens"`
	CompletionTokens *int    `json:"completionTokens"`
	ResponseTime     *int    `json:"responseTime"`
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
	Role         string `json:"role"`
	Content      string `json:"content"`
	Desicion     string `json:"desicion"`
	Memory       Memory `json:"memory"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
}

type Memory struct {
	Agent    string `json:"agent"`
	Sections string `json:"sections"`
}
