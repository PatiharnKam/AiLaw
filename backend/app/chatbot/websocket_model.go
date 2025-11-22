package service

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

	"github.com/PatiharnKam/AiLaw/config"
	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/websocket"
)

type WebSocketService struct {
	cfg    *config.Config
	client *genai.Client
	model  *genai.GenerativeModel
}

type WebSocketMessage struct {
	Type    string `json:"type"` // "user", "assistant", "system", "ack", "error", "token", "complete"
	Content string `json:"content"`
	IsLast  bool   `json:"isLast,omitempty"`
}

// RunPod API structures
type RunPodRequest struct {
	Input Input `json:"input"`
}

type RunPodOutputItem struct {
	Layer string `json:"layer,omitempty"`
	State string `json:"state,omitempty"`
	Type  string `json:"type,omitempty"`
	Data  string `json:"data,omitempty"`
}

type RunPodResponse struct {
	DelayTime     int                `json:"delayTime"`
	ExecutionTime int                `json:"executionTime"`
	ID            string             `json:"id"`
	Output        []RunPodOutputItem `json:"output"`
	Status        string             `json:"status"`
	WorkerID      string             `json:"workerId"`
}

func NewWebSocketService(cfg *config.Config, client *genai.Client, model *genai.GenerativeModel) *WebSocketService {
	return &WebSocketService{
		cfg:    cfg,
		client: client,
		model:  model,
	}
}

func (s *WebSocketService) ProcessMessageWithStreaming(ctx context.Context, userMessage string, conn *websocket.Conn) error {
	// สร้าง request body สำหรับ RunPod API
	reqBody := RunPodRequest{}
	reqBody.Input.Message = []Messages{
		{
			Role: "user",
			Content: userMessage,
		},
		
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Error marshaling request:", err.Error())
		return fmt.Errorf("failed to create request: %w", err)
	}
	bufferJsonData := bytes.NewBuffer(jsonData)

	// สร้าง HTTP request
	req, err := http.NewRequest("POST", s.cfg.APIkey.ModelURL, bufferJsonData)
	if err != nil {
		slog.Error("Error creating request:", err.Error())
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIkey.ModelAPIkey)
	
	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request:", err.Error())
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("API error:", "status", resp.StatusCode, "body", string(bodyBytes))
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// อ่าน response แบบ streaming (ถ้า RunPod รองรับ SSE)
	// ถ้าไม่รองรับ streaming ให้ใช้แบบ batch ด้านล่าง
	return s.processRunPodResponse(resp.Body, conn)
}

// แบบ batch - รอ response ครบแล้วค่อย stream ไป frontend
func (s *WebSocketService) processRunPodResponse(body io.ReadCloser, conn *websocket.Conn) error {
	responseText := ""
	hasError := false

	defer func() {
		if !hasError {
			completeMsg := WebSocketMessage{
				Type:    "complete",
				Content: responseText,
				IsLast:  true,
			}

			if err := conn.WriteJSON(completeMsg); err != nil {
				slog.Error("Error sending completion:", err.Error())
			} else {
				slog.Info("Streaming response completed", "total_length", len(responseText))
			}
		}
	}()

	// อ่าน response
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		hasError = true
		slog.Error("Error reading response:", err.Error())
		conn.WriteJSON(WebSocketMessage{
			Type:    "error",
			Content: "Failed to read API response",
			IsLast:  true,
		})
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var runpodResp RunPodResponse
	if err := json.Unmarshal(bodyBytes, &runpodResp); err != nil {
		hasError = true
		slog.Error("Error parsing response:", err.Error())
		conn.WriteJSON(WebSocketMessage{
			Type:    "error",
			Content: "Failed to parse API response",
			IsLast:  true,
		})
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// ดึงข้อมูลจาก output array
	for _, item := range runpodResp.Output {
		// ตรวจสอบว่าเป็น layer "detail" หรือ "Detail" และมี data
		if (strings.EqualFold(item.Layer, "detail") || strings.EqualFold(item.Layer, "Detail")) && 
		   item.Type == "delta" && 
		   item.Data != "" {
			
			responseText += item.Data

			// ส่ง token ไป frontend
			tokenMsg := WebSocketMessage{
				Type:    "token",
				Content: item.Data,
				IsLast:  false,
			}

			if err := conn.WriteJSON(tokenMsg); err != nil {
				hasError = true
				slog.Error("Error sending token:", err.Error())
				return err
			}

			// เพิ่ม delay เล็กน้อยเพื่อให้เห็น streaming effect
			time.Sleep(50 * time.Millisecond)
		}

		// ตรวจสอบว่า done แล้วหรือยัง
		if strings.EqualFold(item.Layer, "detail") && item.State == "done" {
			slog.Info("Detail layer completed")
			break
		}
	}

	if responseText == "" {
		hasError = true
		slog.Error("No data found in response")
		conn.WriteJSON(WebSocketMessage{
			Type:    "error",
			Content: "No response data received from API",
			IsLast:  true,
		})
		return fmt.Errorf("no data in response")
	}

	return nil
}

// แบบ true streaming - ถ้า RunPod รองรับ Server-Sent Events (SSE)
func (s *WebSocketService) processRunPodResponseSSE(body io.ReadCloser, conn *websocket.Conn) error {
	responseText := ""
	hasError := false

	defer func() {
		if !hasError {
			completeMsg := WebSocketMessage{
				Type:    "complete",
				Content: responseText,
				IsLast:  true,
			}

			if err := conn.WriteJSON(completeMsg); err != nil {
				slog.Error("Error sending completion:", err.Error())
			} else {
				slog.Info("Streaming response completed", "total_length", len(responseText))
			}
		}
	}()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse SSE format: data: {...}
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")
			
			var item RunPodOutputItem
			if err := json.Unmarshal([]byte(jsonStr), &item); err != nil {
				slog.Error("Error parsing SSE data:", err.Error())
				continue
			}

			// ดึง data จาก detail layer
			if (strings.EqualFold(item.Layer, "detail") || strings.EqualFold(item.Layer, "Detail")) && 
			   item.Type == "delta" && 
			   item.Data != "" {
				
				responseText += item.Data

				tokenMsg := WebSocketMessage{
					Type:    "token",
					Content: item.Data,
					IsLast:  false,
				}

				if err := conn.WriteJSON(tokenMsg); err != nil {
					hasError = true
					slog.Error("Error sending token:", err.Error())
					return err
				}
			}

			// ตรวจสอบว่า done แล้วหรือยัง
			if strings.EqualFold(item.Layer, "detail") && item.State == "done" {
				slog.Info("Detail layer completed")
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		hasError = true
		slog.Error("Error reading stream:", err.Error())
		conn.WriteJSON(WebSocketMessage{
			Type:    "error",
			Content: "Error reading stream",
			IsLast:  true,
		})
		return fmt.Errorf("stream error: %w", err)
	}

	return nil
}