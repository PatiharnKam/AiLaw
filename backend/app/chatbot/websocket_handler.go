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

type Client struct {
	conn    *websocket.Conn
	userID  string
	send    chan []byte
	done    chan struct{}
	handler *Handler
	mu      sync.Mutex
	closed  bool
}

func (h *Handler) WebSocketHandler(c *gin.Context) {
	logger := slog.Default()

	userID := c.GetString("userId")
	if userID == "" {
		logger.Error("WebSocket: unauthorized - no user ID from middleware")
		c.JSON(http.StatusUnauthorized, app.Response{
			Code:    app.UnauthorizedErrorCode,
			Message: app.UnauthorizedErrorMessage,
		})
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			allowedOrigins := h.cfg.AllowedOrigin
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
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
		done:    make(chan struct{}),
		handler: h,
	}

	go client.writePump()
	go client.readPump()

	logger.Info("WebSocket connected", "userId", userID)
}

func (c *Client) readPump() {
	logger := slog.Default()
	defer func() {
		c.mu.Lock()
		c.closed = true
		close(c.done)
		c.mu.Unlock()

		close(c.send)
		c.conn.Close()
		logger.Info("WebSocket disconnected", "userId", c.userID)
	}()

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Monitor done channel
	go func() {
		select {
		case <-c.done:
			cancel()
		case <-ctx.Done():
		}
	}()

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
			wsResp.Error = &WSError{
				Code:    "model_error",
				Message: event.Error,
			}
		default:
			return
		}
		c.sendResponse(wsResp)
	}

	resp, err := c.handler.service.ChatbotProcessWithStream(ctx, req, streamCallback)
	if err != nil {
		logger.Error("Chat process error", "error", err.Error())
		c.sendError(resp.Code, resp.Message)
		return
	}

	respData := resp.Data.(StreamingMessageResponse)

	c.sendResponse(WSResponse{
		Type:           "done",
		SessionID:      msg.SessionID,
		ModelMessageID: respData.ModelMessageID,
		Content:        respData.Message,
	})
}

func (c *Client) sendResponse(resp WSResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

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
		Type: "error",
		Error: &WSError{
			Code:    code,
			Message: message,
		},
	})
}
