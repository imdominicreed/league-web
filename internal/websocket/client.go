package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	room   *Room
	userID uuid.UUID
	side   string // "blue", "red", "spectator"
	ready  bool

	mu     sync.RWMutex
	closed bool
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			continue
		}

		c.handleMessage(&msg)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	// Route to v2 command handler
	handler := NewCommandHandler(c)
	msgTypeStr := string(msg.Type)

	switch msgTypeStr {
	case string(MsgTypeCommand):
		v2Msg := &Msg{
			Type:      MsgTypeCommand,
			Payload:   msg.Payload,
			Timestamp: msg.Timestamp,
			Seq:       msg.Seq,
		}
		handler.HandleCommand(v2Msg)

	case string(MsgTypeQuery):
		v2Msg := &Msg{
			Type:      MsgTypeQuery,
			Payload:   msg.Payload,
			Timestamp: msg.Timestamp,
			Seq:       msg.Seq,
		}
		handler.HandleQuery(v2Msg)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
		c.sendError("UNKNOWN_MESSAGE", "Unknown message type")
	}
}

func (c *Client) sendError(code, message string) {
	msg, _ := NewMessage(MessageTypeError, ErrorPayload{
		Code:    code,
		Message: message,
	})
	data, _ := json.Marshal(msg)
	c.trySend(data)
}

func (c *Client) Send(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}
	c.trySend(data)
}

// trySend safely sends data to the client, handling closed channels gracefully.
// The RLock is held through the entire send to prevent races with Close().
func (c *Client) trySend(data []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return
	}

	// Non-blocking send - if buffer is full, drop the message
	select {
	case c.send <- data:
	default:
		// Buffer full, drop the message
	}
}

// Close marks the client as closed and closes its send channel.
// Safe to call multiple times.
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	close(c.send)
}

