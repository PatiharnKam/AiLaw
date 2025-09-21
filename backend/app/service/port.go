package service

import "context"

type Service interface {
	GetMessageService(ctx context.Context, req GetMessageRequest) (*GetMessageResponse, error)
	GetMessageModelService(ctx context.Context, req GetMessageModelRequest) (*GetMessageModelResponse, error)
}

type GetMessageRequest struct {
	UserMessage string `json:"userMessage" binding:"required"`
}

type GetMessageResponse struct {
	Message string `json:"message"`
}

type GetMessageModelRequest struct {
	// UserMessage string `json:"userMessage" binding:"required"`
	Input Input `json:"input"`
}

type Input struct {
	Message []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GetMessageModelResponse struct {
	DelayTime     int    `json:"delayTime"`
	ExecutionTime int    `json:"executionTime"`
	Id            string `json:"id"`
	Output        Output `json:"output"`
	Status        string `json:"status"`
	WorkerId      string `json:"workerId"`
}

type Output struct {
	Content string `json:"content"`
	Memory  Memory `json:"memory"`
	Role    string `json:"role"`
}

type Memory struct {
	Agent         string `json:"agent"`
	GuardDecision string `json:"guard_decision"`
}
