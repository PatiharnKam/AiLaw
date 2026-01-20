package service

import "context"

type Service interface {
	CreateChatSessionService(ctx context.Context, req CreateChatSessionRequest) (CreateChatSessionResponse, error)

	ChatbotProcess(ctx context.Context, req ChatbotProcessRequest) (*GetMessageResponse, error)
	ChatbotModelProcess(ctx context.Context, req ChatbotProcessModelRequest) (*GetMessageResponse, error)
}

type Storage interface {
	CreateSession(ctx context.Context, req CreateChatSessionRequest) (*CreateChatSessionResponse, error)
	UpdateLastMessageAt(ctx context.Context, userId string, sessionId string) error
	SaveUserMessage(ctx context.Context, sessionId, userMessage string) error
	SaveModelMessage(ctx context.Context, sessionId string, modelDetail ModelMessageDetail) error
}

type CreateChatSessionRequest struct {
	UserId string `json:"userId" validate:"required,uuid"`
	Title  string `json:"title"`
}

type CreateChatSessionResponse struct {
	SessionId string `json:"sessionId"`
}

type ChatbotProcessRequest struct {
	UserId      string `json:"userId" validate:"required,uuid"`
	SessionId   string `json:"sessionId" binding:"required" validate:"required,uuid"`
	UserMessage string `json:"userMessage" binding:"required"`
}

type ModelMessageDetail struct {
	Content          string  `json:"message"`
	Feedback         *string `json:"feedback"`
	PromptTokens     *int    `json:"promptTokens"`
	CompletionTokens *int    `json:"completionTokens"`
	ResponseTime     *int    `json:"responseTime"`
}

type GetMessageResponse struct {
	Message string `json:"message"`
}

type ChatbotProcessModelRequest struct {
	UserId    string `json:"userId" validate:"required,uuid"`
	SessionId string `json:"sessionId" binding:"required" validate:"required,uuid"`
	// UserMessage string `json:"userMessage" binding:"required"`
	Input Input `json:"input"`
}

type Input struct {
	Message []Messages `json:"messages"`
}

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content" binding:"required"`
}

type GetMessageModelResponse struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Desicion string `json:"desicion"`
	Memory   Memory `json:"memory"`
}

type Memory struct {
	Agent    string `json:"agent"`
	Sections string `json:"sections"`
}

// type GetMessageModelResponse struct {
// 	DelayTime     int    `json:"delayTime"`
// 	ExecutionTime int    `json:"executionTime"`
// 	Id            string `json:"id"`
// 	Output        Output `json:"output"`
// 	Status        string `json:"status"`
// 	WorkerId      string `json:"workerId"`
// }

// type Output struct {
// 	Content string `json:"content"`
// 	Memory  Memory `json:"memory"`
// 	Role    string `json:"role"`
// }

// type Memory struct {
// 	Agent         string `json:"agent"`
// 	GuardDecision string `json:"guard_decision"`
// }
