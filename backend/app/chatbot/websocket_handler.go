package chatbot

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/PatiharnKam/AiLaw/app"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Configure allowed origins for production
		return true
	},
}

type Client struct {
	conn    *websocket.Conn
	userID  string
	send    chan []byte
	handler *Handler
	mu      sync.Mutex
}

// WebSocket Message Types
type WSMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId,omitempty"`
	Content   string `json:"content,omitempty"`
	ModelType string `json:"modelType,omitempty"` // "default" or "COT"
}

type WSResponse struct {
	Type           string     `json:"type"`
	Content        string     `json:"content,omitempty"`
	SessionID      string     `json:"sessionId,omitempty"`
	ModelMessageID string     `json:"modelMessageId,omitempty"`
	Error          string     `json:"error,omitempty"`
	Usage          *UsageInfo `json:"usage,omitempty"`

	// COT specific fields
	Steps       []string `json:"steps,omitempty"`
	Rationale   string   `json:"rationale,omitempty"`
	CurrentStep int      `json:"currentStep,omitempty"`
	TotalSteps  int      `json:"totalSteps,omitempty"`
	StepDesc    string   `json:"stepDescription,omitempty"`
	Status      string   `json:"status,omitempty"`
}

type UsageInfo struct {
	InputTokens  int   `json:"inputTokens"`
	OutputTokens int   `json:"outputTokens"`
	TotalTokens  int   `json:"totalTokens"`
	Remaining    int64 `json:"remaining"`
}

func (h *Handler) WebSocketHandler(c *gin.Context) {
	logger := slog.Default()

	// ดึง userId จาก context ที่ middleware set ไว้
	userID := c.GetString("userId")
	if userID == "" {
		logger.Error("WebSocket: unauthorized - no user ID from middleware")
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.UnauthorizedErrorCode,
			Message: app.UnauthorizedErrorMessage,
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket: failed to upgrade connection", "error", err.Error())
		return
	}

	client := &Client{
		conn:    conn,
		userID:  userID,
		send:    make(chan []byte, 256),
		handler: h,
	}

	go client.writePump()
	go client.readPump()

	logger.Info("WebSocket connected", "userId", userID)
}

func (c *Client) readPump() {
	logger := slog.Default()
	defer func() {
		c.conn.Close()
		close(c.send)
		logger.Info("WebSocket disconnected", "userId", c.userID)
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error", "error", err.Error())
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.sendError("invalid_message", "Invalid message format")
			continue
		}

		switch wsMsg.Type {
		case "chat":
			go c.handleChatMessage(wsMsg)
		case "ping":
			c.sendResponse(WSResponse{Type: "pong"})
		default:
			c.sendError("unknown_type", "Unknown message type: "+wsMsg.Type)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleChatMessage(msg WSMessage) {
	logger := slog.Default()
	ctx := context.Background()

	if msg.SessionID == "" || msg.Content == "" {
		c.sendError("invalid_request", "sessionId and content are required")
		return
	}

	// Send acknowledgment
	c.sendResponse(WSResponse{
		Type:      "ack",
		SessionID: msg.SessionID,
	})

	// Build request
	req := ChatbotProcessRequest{
		UserId:    c.userID,
		SessionId: msg.SessionID,
		ModelType: msg.ModelType,
		Input: Input{
			Messages: Messages{
				Role:    "user",
				Content: msg.Content,
			},
		},
	}

	// Stream callback
	streamCallback := func(event StreamEvent) {
		var wsResp WSResponse
		wsResp.SessionID = msg.SessionID

		switch event.Type {
		case "guard_passed":
			wsResp.Type = "guard_passed"
		case "status":
			wsResp.Type = "status"
			wsResp.Status = event.Message
		case "plan":
			wsResp.Type = "plan"
			wsResp.Steps = event.Steps
			wsResp.Rationale = event.Rationale
		case "cot_step":
			wsResp.Type = "cot_step"
			wsResp.CurrentStep = event.Step
			wsResp.TotalSteps = event.Total
			wsResp.StepDesc = event.Description
		case "content":
			wsResp.Type = "chunk"
			wsResp.Content = event.Text
		case "error":
			wsResp.Type = "error"
			wsResp.Error = event.Error
		default:
			return
		}
		c.sendResponse(wsResp)
	}

	var resp *StreamingMessageResponse
	var err error

	resp, err = c.handler.service.ChatbotProcessWithStream(ctx, req, streamCallback)
	if err != nil {
		logger.Error("Chat process error", "error", err.Error())
		c.sendError("process_error", err.Error())
		return
	}

	// Send completion
	c.sendResponse(WSResponse{
		Type:           "done",
		SessionID:      msg.SessionID,
		ModelMessageID: resp.ModelMessageID,
		Content:        resp.Message,
		Usage: &UsageInfo{
			InputTokens:  resp.InputTokens,
			OutputTokens: resp.OutputTokens,
			TotalTokens:  resp.TotalTokens,
			Remaining:    resp.Remaining,
		},
	})
}

func (c *Client) sendResponse(resp WSResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("Failed to marshal response", "error", err.Error())
		return
	}

	select {
	case c.send <- data:
	default:
		slog.Warn("Client send buffer full", "userId", c.userID)
	}
}

func (c *Client) sendError(code, message string) {
	c.sendResponse(WSResponse{
		Type:  "error",
		Error: message,
	})
}
