package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// LobbyClient wraps a WebSocket connection for a lobby participant
type LobbyClient struct {
	hub     *LobbyHub
	conn    *websocket.Conn
	send    chan []byte
	userID  uuid.UUID
	lobbyID uuid.UUID

	mu     sync.RWMutex
	closed bool
}

// NewLobbyClient creates a new lobby client
func NewLobbyClient(hub *LobbyHub, conn *websocket.Conn, userID uuid.UUID) *LobbyClient {
	return &LobbyClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *LobbyClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
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
				log.Printf("LobbyClient websocket error: %v", err)
			}
			break
		}

		var msg LobbyMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("LobbyClient failed to unmarshal message: %v", err)
			continue
		}

		c.handleMessage(&msg)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *LobbyClient) WritePump() {
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

// handleMessage processes incoming client messages
func (c *LobbyClient) handleMessage(msg *LobbyMessage) {
	switch msg.Type {
	case LobbyMsgJoinLobby:
		c.handleJoinLobby(msg)
	default:
		log.Printf("LobbyClient unknown message type: %s", msg.Type)
		c.sendError("UNKNOWN_MESSAGE", "Unknown message type")
	}
}

// handleJoinLobby processes join lobby requests
func (c *LobbyClient) handleJoinLobby(msg *LobbyMessage) {
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		c.sendError("INVALID_PAYLOAD", "Invalid payload")
		return
	}

	var payload JoinLobbyPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		c.sendError("INVALID_PAYLOAD", "Invalid join lobby payload")
		return
	}

	lobbyID, err := uuid.Parse(payload.LobbyID)
	if err != nil {
		c.sendError("INVALID_LOBBY_ID", "Invalid lobby ID format")
		return
	}

	c.hub.joinLobby <- &JoinLobbyRequest{
		Client:  c,
		LobbyID: lobbyID,
	}
}

// Send sends a message to the client
func (c *LobbyClient) Send(msg *LobbyMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("LobbyClient failed to marshal message: %v", err)
		return
	}
	c.trySend(data)
}

// trySend safely sends data to the client
func (c *LobbyClient) trySend(data []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return
	}

	select {
	case c.send <- data:
	default:
		// Buffer full, drop the message
	}
}

// sendError sends an error message to the client
func (c *LobbyClient) sendError(code, message string) {
	msg := NewLobbyMessage(LobbyMsgError, LobbyErrorPayload{
		Code:    code,
		Message: message,
	})
	c.Send(msg)
}

// Close marks the client as closed and closes its send channel
func (c *LobbyClient) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()

	close(c.send)
}

// UserID returns the client's user ID
func (c *LobbyClient) UserID() uuid.UUID {
	return c.userID
}

// LobbyID returns the client's current lobby ID
func (c *LobbyClient) LobbyID() uuid.UUID {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lobbyID
}

// SetLobbyID sets the client's current lobby ID
func (c *LobbyClient) SetLobbyID(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lobbyID = id
}
