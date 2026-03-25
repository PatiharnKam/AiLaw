package chatbot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/google/uuid"
)

func (s *MessageService) ChatbotProcessWithStream(ctx context.Context, req ChatbotProcessRequest, onChunk StreamCallback) (app.Response, error) {
	logger := slog.Default()

	userPromptTokens, err := s.quotaService.CheckPromptLength(req.Input.Messages.Content)
	if err != nil {
		return app.Response{
			Code:    app.UserPromptLengthExceededErrorCode,
			Message: app.UserPromptLengthExceededErrorMessage,
		}, fmt.Errorf("prompt length exceeded: %w", err)
	}

	quotaStatus, err := s.quotaService.CheckQuota(ctx, req.UserId)
	if err != nil {
		return app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		}, fmt.Errorf("failed to check quota: %w", err)
	}

	if quotaStatus.IsExceeded {
		return app.Response{
			Code:    app.QuotaExceededErrorCode,
			Message: app.QuotaExceededErrorMessage,
		}, fmt.Errorf("daily quota exceeded")
	}

	err = s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		}, fmt.Errorf("error updating last message: %w", err)
	}

	modelURL := s.cfg.Model.ModelStreamURL
	if req.ModelType == "COT" {
		modelURL = s.cfg.Model.ModelCOTStreamURL
	}

	modelMessageDetail := ModelMessageDetail{
		ModelType: req.ModelType,
	}

	var modelErr error
	var wasCancelled bool
	responseTime, err := s.callFastAPIStream(ctx, req, modelURL, func(event StreamEvent) {
		switch event.Type {
		case "content":
			modelMessageDetail.Content += event.Text
			if onChunk != nil {
				onChunk(event)
			}
		case "status", "guard_passed", "plan", "cot_step":
			if onChunk != nil {
				onChunk(event)
			}
		case "done":
			modelMessageDetail.TotalInputTokens = event.TotalInputTokens
			modelMessageDetail.TotalOutputTokens = event.TotalOutputTokens
			modelMessageDetail.FinalOutputTokens = event.FinalOutputTokens
			modelMessageDetail.TotalUsedTokens = event.TotalUsedTokens
			if event.FullContent != "" {
				modelMessageDetail.Content = event.FullContent
			}
		case "cancelled":
			wasCancelled = true
			logger.Info("stream cancelled by FastAPI", "sessionId", req.SessionId)
		case "error":
			logger.Error("stream error", "error", event.Error)
			modelErr = fmt.Errorf("model error: %s", event.Error)
		}
	})
	modelMessageDetail.ResponseTime = responseTime

	if wasCancelled || ctx.Err() == context.Canceled {
		return app.Response{}, fmt.Errorf("cancelled")
	}

	if err != nil || modelErr != nil {
		return app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		}, fmt.Errorf("streaming error: %w", err)
	}

	modelMessageId := uuid.NewString()
	err = s.storage.SaveUserMessage(ctx, req.UserId, req.SessionId, modelMessageId, req.Input.Messages.Content, userPromptTokens)
	if err != nil {
		return app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		}, fmt.Errorf("error saving user message: %w", err)
	}

	err = s.storage.SaveModelMessage(ctx, req.UserId, req.SessionId, modelMessageId, modelMessageDetail)
	if err != nil {
		return app.Response{
			Code:    app.InternalServerErrorCode,
			Message: app.InternalServerErrorMessage,
		}, fmt.Errorf("error saving model message: %w", err)
	}

	err = s.quotaService.ConsumeTokens(ctx, req.UserId, int64(modelMessageDetail.TotalUsedTokens))
	if err != nil {
		logger.Warn("failed to consume tokens", "error", err)
	}

	return app.Response{
		Code:    app.SUCCESS_CODE,
		Message: app.SUCCESS_MSG,
		Data: StreamingMessageResponse{
			Message:        modelMessageDetail.Content,
			ModelMessageID: modelMessageId,
		},
	}, nil
}

// callFastAPIStream calls FastAPI SSE endpoint
func (s *MessageService) callFastAPIStream(
	ctx context.Context,
	req ChatbotProcessRequest,
	modelURL string,
	onEvent func(StreamEvent),
) (*float64, error) {
	// Build request body
	data := ChatbotRequest{
		Messages: []Messages{
			{
				Role:    req.Input.Messages.Role,
				Content: req.Input.Messages.Content,
			},
		},
		SessionID: req.SessionId,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %w", err)
	}

	callStart := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, "POST", modelURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.Model.ModelAPIkey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error calling API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	reader := bufio.NewReader(resp.Body)
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse SSE format: "data: {...}"
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				break
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				slog.Warn("failed to parse SSE event", "data", data, "error", err)
				continue
			}

			onEvent(event)

			if event.Type == "done" || event.Type == "error" || event.Type == "cancelled" {
				break
			}
		}
	}

	responseTime := time.Since(callStart).Seconds()
	return &responseTime, nil
}

// CancelModelRequest sends a cancel signal to the FastAPI service
func (s *MessageService) CancelModelRequest(sessionID string) {
	logger := slog.Default()

	cancelURL := s.cfg.Model.ModelCancelURL

	body, _ := json.Marshal(map[string]string{
		"session_id": sessionID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cancelURL, bytes.NewBuffer(body))
	if err != nil {
		logger.Error("Failed to create cancel request", "error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.Model.ModelAPIkey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logger.Error("Failed to send cancel request to model", "error", err)
		return
	}
	defer resp.Body.Close()
}
