package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/PatiharnKam/AiLaw/config"
	"github.com/google/generative-ai-go/genai"
)

type MessageService struct {
	cfg     *config.Config
	storage Storage
	client  *genai.Client
}

func NewService(cfg *config.Config, storage Storage, client *genai.Client) *MessageService {
	return &MessageService{
		cfg:     cfg,
		storage: storage,
		client:  client,
	}

}

func (s *MessageService) CreateChatSessionService(ctx context.Context, req CreateChatSessionRequest) (CreateChatSessionResponse, error) {
	resp, err := s.storage.CreateSession(ctx, req)
	if err != nil {
		return CreateChatSessionResponse{}, fmt.Errorf("error when create session : %w", err)
	}
	return *resp, nil
}

func (s *MessageService) ChatbotProcess(ctx context.Context, req ChatbotProcessModelRequest) (*GetMessageResponse, error) {
	err := s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error when update last message at session : %w", err)
	}

	resp, err := s.callChatbot(req)
	if err != nil {
		return nil, fmt.Errorf("failed when call chatbot : %w", err)
	}

	err = s.storage.SaveUserMessage(ctx, req.SessionId, req.Input.Message[0].Content)
	if err != nil {
		return nil, fmt.Errorf("error when save user message at session : %w", err)
	}

	modelmessageDetail := ModelMessageDetail{
		Content: resp.Content,
	}

	if modelmessageDetail.Content == strings.Trim(modelmessageDetail.Content, modelmessageDetail.Content) {
		modelmessageDetail.Content = "No response"
	}

	err = s.storage.SaveModelMessage(ctx, req.SessionId, modelmessageDetail)
	if err != nil {
		return nil, fmt.Errorf("error when save model message at session : %w", err)
	}

	return &GetMessageResponse{Message: modelmessageDetail.Content}, nil
}

func (s *MessageService) callChatbot(req ChatbotProcessModelRequest) (*GetMessageModelResponse, error) {
	client := &http.Client{}

	reqHttp, err := s.setHttpRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error when setting http request : %w", err)
	}

	resp, err := client.Do(reqHttp)
	if err != nil {
		return nil, fmt.Errorf("error whell calling API Model : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from model: %w", err)
	}

	var response GetMessageModelResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	fmt.Println()
	slog.Info("resp is", "response", response)
	fmt.Println()

	return &response, nil
}

func (s *MessageService) setHttpRequest(req ChatbotProcessModelRequest) (*http.Request, error) {
	data := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    req.Input.Message[0].Role,
				"content": req.Input.Message[0].Content,
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %w", err)
	}
	buffer := bytes.NewBuffer(jsonData)

	modelURL := s.cfg.APIkey.ModelURL
	if req.ModelType == "COT" {
		modelURL = s.cfg.APIkey.ModelCOTURL
	}

	reqHttp, err := http.NewRequest("POST", modelURL, buffer)
	if err != nil {
		return nil, fmt.Errorf("error when creating API request: %w", err)
	}
	reqHttp.Header.Set("Content-Type", "application/json")
	reqHttp.Header.Set("Authorization", "Bearer "+s.cfg.APIkey.ModelAPIkey)

	return reqHttp, nil
}
