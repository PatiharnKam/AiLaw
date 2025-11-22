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
	model   *genai.GenerativeModel
}

func NewService(cfg *config.Config, storage Storage, client *genai.Client, model *genai.GenerativeModel) *MessageService {
	return &MessageService{
		cfg:     cfg,
		storage: storage,
		client:  client,
		model:   model,
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
	fmt.Println("start update last message")
	err := s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error when update last message at session : %w", err)
	}

	prompt := genai.Text(req.UserMessage)

	// เรียก Gemini API
	resp, err := s.model.GenerateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// ดึง response text
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	fmt.Println("start save user message")
	err = s.storage.SaveUserMessage(ctx, req.SessionId, req.UserMessage)
	if err != nil {
		return nil, fmt.Errorf("error when save user message at session : %w", err)
	}

	// แปลง response เป็น string
	responseText := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		responseText += fmt.Sprintf("%v", part)
	}

	modelmessageDetail := ModelMessageDetail{
		Content: responseText,
	}

	fmt.Println("start save model message")
	err = s.storage.SaveModelMessage(ctx, req.SessionId, modelmessageDetail)
	if err != nil {
		return nil, fmt.Errorf("error when save model message at session : %w", err)
	}

	return &GetMessageResponse{Message: responseText}, nil
}
func (s *MessageService) ChatbotModelProcess(ctx context.Context, req ChatbotProcessModelRequest) (*GetMessageResponse, error) {
	err := s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error when update last message at session : %w", err)
	}

	resp, err := s.callChatbot(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed when call chatbot : %w", err)
	}

	err = s.storage.SaveUserMessage(ctx, req.SessionId, req.Input.Message[0].Content)
	if err != nil {
		return nil, fmt.Errorf("error when save user message at session : %w", err)
	}

	modelmessageDetail := ModelMessageDetail{
		Content: resp.Output.Content,
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

func (s *MessageService) callChatbot(ctx context.Context, req ChatbotProcessModelRequest) (*GetMessageModelResponse, error) {
	client := &http.Client{}

	data := map[string]interface{}{
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    req.Input.Message[0].Role,
					"content": req.Input.Message[0].Content, // Assuming your GetMessageModelRequest has Message field
				},
			},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %w", err)
	}
	buffer := bytes.NewBuffer(jsonData)

	fmt.Println()
	fmt.Println()
	slog.Info("buffer is : ", "buffer", buffer)
	fmt.Println()
	fmt.Println()

	reqHttp, err := http.NewRequest("POST", s.cfg.APIkey.ModelURL, buffer)
	if err != nil {
		return nil, fmt.Errorf("error when creating API request: %w", err)
	}
	reqHttp.Header.Set("Content-Type", "application/json")
	reqHttp.Header.Set("Authorization", "Bearer "+s.cfg.APIkey.ModelAPIkey)

	resp, err := client.Do(reqHttp)
	if err != nil {
		return nil, fmt.Errorf("error whell calling API Model : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from model: %w", err)
	}

	var response GetMessageModelResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	fmt.Println()
	fmt.Println()
	slog.Info("resp is", "response", response)
	fmt.Println()
	fmt.Println()

	return &response, nil
}
