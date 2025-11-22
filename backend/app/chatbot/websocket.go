package service

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"strings"
// 	"time"

// 	"github.com/PatiharnKam/AiLaw/config"
// 	"github.com/google/generative-ai-go/genai"
// 	"github.com/gorilla/websocket"
// )

// type WebSocketService struct {
// 	cfg    *config.Config
// 	client *genai.Client
// 	model  *genai.GenerativeModel
// }

// type WebSocketMessage struct {
// 	Type    string `json:"type"` // "user", "assistant", "system", "ack", "error", "token", "complete"
// 	Content string `json:"content"`
// 	IsLast  bool   `json:"isLast,omitempty"`
// }

// func NewWebSocketService(cfg *config.Config, client *genai.Client, model *genai.GenerativeModel) *WebSocketService {
// 	return &WebSocketService{
// 		cfg:    cfg,
// 		client: client,
// 		model:  model,
// 	}
// }

// func (s *WebSocketService) ProcessMessageWithStreaming(ctx context.Context, userMessage string, conn *websocket.Conn) error {
// 	prompt := genai.Text(userMessage)

// 	// เรียก Gemini API แบบ streaming
// 	iter := s.model.GenerateContentStream(ctx, prompt)

// 	responseText := ""
// 	hasError := false

// 	// ใช้ defer เพื่อจัดการ cleanup
// 	defer func() {
// 		if !hasError {
// 			// ส่งสัญญาณว่าเสร็จสิ้นแล้ว
// 			completeMsg := WebSocketMessage{
// 				Type:    "complete",
// 				Content: responseText,
// 				IsLast:  true,
// 			}

// 			if err := conn.WriteJSON(completeMsg); err != nil {
// 				slog.Error("Error sending completion:", err.Error())
// 			} else {
// 				slog.Info("Streaming response completed", "total_length", len(responseText))
// 			}
// 		}
// 	}()

// 	for {
// 		resp, err := iter.Next()
// 		if err != nil {
// 			// ใช้ strings.Contains เพื่อจับ error message ได้หลากหลายกว่า
// 			errStr := err.Error()
// 			if strings.Contains(errStr, "no more items") ||
// 				strings.Contains(errStr, "iterator is done") ||
// 				strings.Contains(errStr, "EOF") {
// 				slog.Info("Streaming completed normally:", errStr)
// 				break // จบ loop ปกติ
// 			}

// 			// Error จริงๆ
// 			hasError = true
// 			slog.Error("Streaming error:", errStr)

// 			// ส่ง error message ไป frontend
// 			errorMsg := WebSocketMessage{
// 				Type:    "error",
// 				Content: "Sorry, I encountered an error while generating the response.",
// 				IsLast:  true,
// 			}
// 			conn.WriteJSON(errorMsg)

// 			return fmt.Errorf("failed to get streaming response: %w", err)
// 		}

// 		// ประมวลผล response แต่ละ chunk
// 		if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
// 			for _, part := range resp.Candidates[0].Content.Parts {
// 				chunkText := fmt.Sprintf("%v", part)
// 				responseText += chunkText

// 				// ส่ง token ทีละ chunk
// 				tokenMsg := WebSocketMessage{
// 					Type:    "token",
// 					Content: chunkText,
// 					IsLast:  false,
// 				}

// 				if err := conn.WriteJSON(tokenMsg); err != nil {
// 					hasError = true
// 					slog.Error("Error sending token:", err.Error())
// 					return err
// 				}

// 				// เพิ่ม delay เล็กน้อยเพื่อให้ frontend render ทัน
// 				time.Sleep(10 * time.Millisecond)
// 			}
// 		}
// 	}

// 	return nil
// }
