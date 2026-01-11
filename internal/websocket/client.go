package websocket

import (
	"encoding/json"
	"log"
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
	switch msg.Type {
	case MessageTypeJoinRoom:
		var payload JoinRoomPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid join room payload")
			return
		}
		c.hub.joinRoom <- &JoinRoomRequest{
			Client: c,
			RoomID: payload.RoomID,
			Side:   payload.Side,
		}

	case MessageTypeSelectChampion:
		var payload SelectChampionPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid select champion payload")
			return
		}
		if c.room != nil {
			c.room.selectChampion <- &SelectChampionRequest{
				Client:     c,
				ChampionID: payload.ChampionID,
			}
		}

	case MessageTypeLockIn:
		if c.room != nil {
			c.room.lockIn <- c
		}

	case MessageTypeHoverChampion:
		var payload HoverChampionPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid hover champion payload")
			return
		}
		if c.room != nil {
			c.room.hoverChampion <- &HoverChampionRequest{
				Client:     c,
				ChampionID: payload.ChampionID,
			}
		}

	case MessageTypeReady:
		var payload ReadyPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid ready payload")
			return
		}
		if c.room != nil {
			c.room.ready <- &ReadyRequest{
				Client: c,
				Ready:  payload.Ready,
			}
		}

	case MessageTypeStartDraft:
		if c.room != nil {
			c.room.startDraft <- c
		}

	case MessageTypeSyncState:
		if c.room != nil {
			c.room.syncState <- c
		}

	case MessageTypePauseDraft:
		if c.room != nil {
			c.room.pauseDraft <- c
		}

	case MessageTypeResumeDraft:
		if c.room != nil {
			c.room.resumeDraft <- c
		}

	case MessageTypeProposeEdit:
		var payload ProposeEditPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid propose edit payload")
			return
		}
		if c.room != nil {
			c.room.proposeEdit <- &ProposeEditRequest{
				Client:  c,
				Payload: payload,
			}
		}

	case MessageTypeConfirmEdit:
		if c.room != nil {
			c.room.confirmEdit <- c
		}

	case MessageTypeRejectEdit:
		if c.room != nil {
			c.room.rejectEdit <- c
		}

	case MessageTypeReadyToResume:
		var payload ReadyToResumePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			c.sendError("INVALID_PAYLOAD", "Invalid ready to resume payload")
			return
		}
		if c.room != nil {
			c.room.readyToResume <- &ReadyToResumeRequest{
				Client: c,
				Ready:  payload.Ready,
			}
		}
	}
}

func (c *Client) sendError(code, message string) {
	msg, _ := NewMessage(MessageTypeError, ErrorPayload{
		Code:    code,
		Message: message,
	})
	data, _ := json.Marshal(msg)
	c.send <- data
}

func (c *Client) Send(msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal message: %v", err)
		return
	}
	c.send <- data
}
