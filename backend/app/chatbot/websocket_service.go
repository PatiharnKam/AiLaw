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
)

// chatbotProcessStreamInternal - internal streaming implementation
func (s *MessageService) ChatbotProcessWithStream(ctx context.Context, req ChatbotProcessRequest, onChunk StreamCallback) (*StreamingMessageResponse, error) {
	logger := slog.Default()

	// 1. Check prompt length
	userPromptTokens, err := s.quotaService.CheckPromptLength(req.Input.Messages.Content)
	if err != nil {
		return nil, fmt.Errorf("prompt length exceeded: %w", err)
	}

	// 2. Check quota
	quotaStatus, err := s.quotaService.CheckQuota(ctx, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("failed to check quota: %w", err)
	}

	if quotaStatus.IsExceeded {
		return nil, fmt.Errorf("daily quota exceeded")
	}

	// 3. Update last message timestamp
	err = s.storage.UpdateLastMessageAt(ctx, req.UserId, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error updating last message: %w", err)
	}

	// 4. Save user message
	err = s.storage.SaveUserMessage(ctx, req.SessionId, req.Input.Messages.Content, userPromptTokens)
	if err != nil {
		return nil, fmt.Errorf("error saving user message: %w", err)
	}

	// 5. Call FastAPI streaming endpoint
	var fullContent string
	var inputTokens, outputTokens int

	modelURL := s.cfg.Model.ModelStreamURL
	if req.ModelType == "COT" {
		modelURL = s.cfg.Model.ModelCOTStreamURL
	}

	err = s.callFastAPIStream(ctx, req, modelURL, func(event StreamEvent) {
		switch event.Type {
		case "content":
			fullContent += event.Text
			if onChunk != nil {
				onChunk(event)
			}
		case "status", "guard_passed", "plan", "cot_step":
			if onChunk != nil {
				onChunk(event)
			}
		case "done":
			inputTokens = event.InputTokens
			outputTokens = event.OutputTokens
			if event.FullContent != "" {
				fullContent = event.FullContent
			}
		case "error":
			logger.Error("stream error", "error", event.Error)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	totalTokens := inputTokens + outputTokens

	fmt.Println()
	fmt.Println()
	fmt.Println("totalTokens :",totalTokens)
	fmt.Println("inputTokens :",inputTokens)
	fmt.Println("outputTokens :",outputTokens)
	fmt.Println()
	fmt.Println()

	// 6. Save model message
	modelMessageDetail := ModelMessageDetail{
		Content:          fullContent,
		PromptTokens:     &inputTokens,
		CompletionTokens: &totalTokens,
	}

	messageID, err := s.storage.SaveModelMessage(ctx, req.SessionId, modelMessageDetail)
	if err != nil {
		return nil, fmt.Errorf("error saving model message: %w", err)
	}

	// 7. Consume tokens
	err = s.quotaService.ConsumeTokens(ctx, req.UserId, int64(totalTokens))
	if err != nil {
		logger.Warn("failed to consume tokens", "error", err)
	}

	// 8. Get remaining quota
	newQuotaStatus, _ := s.quotaService.CheckQuota(ctx, req.UserId)

	return &StreamingMessageResponse{
		Message:        fullContent,
		ModelMessageID: messageID,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		TotalTokens:    totalTokens,
		Remaining:      newQuotaStatus.Remaining,
	}, nil
}

// callFastAPIStream calls FastAPI SSE endpoint
func (s *MessageService) callFastAPIStream(
	ctx context.Context,
	req ChatbotProcessRequest,
	modelURL string,
	onEvent func(StreamEvent),
) error {
	// Build request body
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
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", modelURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.Model.ModelAPIkey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error calling API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
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

			if event.Type == "done" || event.Type == "error" {
				break
			}
		}
	}

	return nil
}
