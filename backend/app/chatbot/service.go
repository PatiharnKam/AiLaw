package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/PatiharnKam/AiLaw/app/quota"
	"github.com/PatiharnKam/AiLaw/config"
)

type MessageService struct {
	cfg          *config.Config
	storage      Storage
	quotaService quota.QuotaService
	httpClient   *http.Client
}

func NewService(cfg *config.Config, storage Storage, quotaService quota.QuotaService) *MessageService {
	return &MessageService{
		cfg:          cfg,
		storage:      storage,
		quotaService: quotaService,
		httpClient:   &http.Client{Timeout: 0},
	}

}

func (s *MessageService) CreateChatSessionService(ctx context.Context, req CreateChatSessionRequest) (CreateChatSessionResponse, error) {
	resp, err := s.storage.CreateSession(ctx, req)
	if err != nil {
		return CreateChatSessionResponse{}, fmt.Errorf("error when create session : %w", err)
	}
	return *resp, nil
}

func (s *MessageService) ChatbotProcess(ctx context.Context, req ChatbotProcessRequest) (*GetMessageResponse, error) {
	userPromptTokens, err := s.quotaService.CheckPromptLength(req.Input.Messages.Content)
	if err != nil {
		return nil, fmt.Errorf("prompt length exceeded : %w", err)
	}

	quotaStatus, err := s.quotaService.CheckQuota(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to check quota: %w", err)
	}

	fmt.Println()
	slog.Info("quota status", "status", quotaStatus)

	if quotaStatus.IsExceeded {
		return nil, fmt.Errorf("daily quota exceeded")
	}

	err = s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error when update last message at session : %w", err)
	}

	resp, err := s.callChatbot(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed when call chatbot : %w", err)
	}

	err = s.storage.SaveUserMessage(ctx, req.SessionId, req.Input.Messages.Content, userPromptTokens)
	if err != nil {
		return nil, fmt.Errorf("error when save user message at session : %w", err)
	}

	modelmessageDetail := ModelMessageDetail{
		Content:          resp.Content,
		PromptTokens:     &resp.InputTokens,
		CompletionTokens: &resp.TotalTokens,
	}

	messageID, err := s.storage.SaveModelMessage(ctx, req.SessionId, modelmessageDetail)
	if err != nil {
		return nil, fmt.Errorf("error when save model message at session : %w", err)
	}

	err = s.quotaService.ConsumeTokens(ctx, req.UserId, int64(resp.TotalTokens))
	if err != nil {
		return nil, fmt.Errorf("failed to consume tokens: %w", err)
	}

	fmt.Println()
	slog.Info("resp is", "response", resp)
	fmt.Println()

	return &GetMessageResponse{
		Message:        modelmessageDetail.Content,
		ModelMessageID: messageID,
	}, nil
}

func (s *MessageService) callChatbot(ctx context.Context, req ChatbotProcessRequest) (*ChatbotResponse, error) {

	data := ChatbotRequest{
		Messages: []Messages{
			{
				Role:    req.Input.Messages.Role,
				Content: req.Input.Messages.Content,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %w", err)
	}

	modelURL := s.cfg.Model.ModelURL
	if req.ModelType == "COT" {
		modelURL = s.cfg.Model.ModelCOTURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", modelURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error when creating API request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.Model.ModelAPIkey)

	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error whell calling API Model : %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", httpResp.StatusCode)
	}

	httpRespBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from model: %w", err)
	}

	var response ChatbotResponse
	if err := json.Unmarshal(httpRespBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return &response, nil
}
