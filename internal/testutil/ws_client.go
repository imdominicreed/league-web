package testutil

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/websocket"
	gorillaWS "github.com/gorilla/websocket"
)

// WSClient is a test WebSocket client
type WSClient struct {
	t        *testing.T
	conn     *gorillaWS.Conn
	messages chan *websocket.Message
	errors   chan error
	done     chan struct{}
	mu       sync.Mutex
}

// NewWSClient creates a new WebSocket test client
func NewWSClient(t *testing.T, url string) *WSClient {
	t.Helper()

	dialer := gorillaWS.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket: %v", err)
	}

	client := &WSClient{
		t:        t,
		conn:     conn,
		messages: make(chan *websocket.Message, 100),
		errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}

	go client.readPump()

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

// readPump reads messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer close(c.messages)
	for {
		select {
		case <-c.done:
			return
		default:
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				select {
				case <-c.done:
					return
				case c.errors <- err:
				}
				return
			}

			var msg websocket.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				c.errors <- err
				continue
			}

			select {
			case c.messages <- &msg:
			case <-c.done:
				return
			}
		}
	}
}

// Close closes the WebSocket connection gracefully
func (c *WSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return
	default:
		close(c.done)
		// Send close frame and close connection without artificial delay
		c.conn.WriteMessage(gorillaWS.CloseMessage, gorillaWS.FormatCloseMessage(gorillaWS.CloseNormalClosure, ""))
		c.conn.Close()
	}
}

// sendCommand sends a v2 COMMAND message to the server
func (c *WSClient) sendCommand(action websocket.CommandAction, payload interface{}) {
	c.t.Helper()

	cmd := websocket.Command{
		Action:  action,
		Payload: nil,
	}

	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			c.t.Fatalf("failed to marshal command payload: %v", err)
		}
		cmd.Payload = payloadBytes
	}

	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		c.t.Fatalf("failed to marshal command: %v", err)
	}

	msg := &websocket.Msg{
		Type:      websocket.MsgTypeCommand,
		Payload:   cmdBytes,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.t.Fatalf("failed to marshal message: %v", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(gorillaWS.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		c.t.Fatalf("failed to send command: %v", err)
	}
}

// JoinRoom sends a v2 join_room COMMAND
func (c *WSClient) JoinRoom(roomID, side string) {
	c.sendCommand(websocket.CmdJoinRoom, websocket.CmdJoinRoomPayload{
		RoomID: roomID,
		Side:   side,
	})
}

// Ready sends a v2 set_ready COMMAND
func (c *WSClient) Ready(ready bool) {
	c.sendCommand(websocket.CmdSetReady, websocket.CmdSetReadyPayload{
		Ready: ready,
	})
}

// StartDraft sends a v2 start_draft COMMAND
func (c *WSClient) StartDraft() {
	c.sendCommand(websocket.CmdStartDraft, nil)
}

// SelectChampion sends a v2 select_champion COMMAND
func (c *WSClient) SelectChampion(championID string) {
	c.sendCommand(websocket.CmdSelectChampion, websocket.CmdSelectChampionPayload{
		ChampionID: championID,
	})
}

// LockIn sends a v2 lock_in COMMAND
func (c *WSClient) LockIn() {
	c.sendCommand(websocket.CmdLockIn, nil)
}

// HoverChampion sends a v2 hover_champion COMMAND
func (c *WSClient) HoverChampion(championID *string) {
	c.sendCommand(websocket.CmdHoverChampion, websocket.CmdHoverChampionPayload{
		ChampionID: championID,
	})
}

// SyncState sends a v2 sync_state QUERY
func (c *WSClient) SyncState() {
	c.sendQuery(websocket.QuerySyncState)
}

// sendQuery sends a v2 QUERY message to the server
func (c *WSClient) sendQuery(queryType websocket.QueryType) {
	c.t.Helper()

	query := websocket.Query{
		Query: queryType,
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		c.t.Fatalf("failed to marshal query: %v", err)
	}

	msg := &websocket.Msg{
		Type:      websocket.MsgTypeQuery,
		Payload:   queryBytes,
		Timestamp: time.Now().UnixMilli(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		c.t.Fatalf("failed to marshal message: %v", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(gorillaWS.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		c.t.Fatalf("failed to send query: %v", err)
	}
}

// ExpectMessage waits for a message of the specified type
func (c *WSClient) ExpectMessage(msgType websocket.MessageType, timeout time.Duration) *websocket.Message {
	c.t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.messages:
			if msg == nil {
				c.t.Fatalf("connection closed while waiting for %s", msgType)
			}
			if msg.Type == msgType {
				return msg
			}
			// Skip other message types (like TIMER_TICK)
		case err := <-c.errors:
			c.t.Fatalf("error while waiting for %s: %v", msgType, err)
		case <-deadline:
			c.t.Fatalf("timeout waiting for message type %s", msgType)
		}
	}
}

// ExpectStateSync waits for and decodes a STATE_SYNC message
func (c *WSClient) ExpectStateSync(timeout time.Duration) *websocket.StateSyncPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypeStateSync, timeout)

	var payload websocket.StateSyncPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode state sync payload: %v", err)
	}

	return &payload
}

// ExpectPlayerUpdate waits for and decodes a PLAYER_UPDATE message
func (c *WSClient) ExpectPlayerUpdate(timeout time.Duration) *websocket.PlayerUpdatePayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypePlayerUpdate, timeout)

	var payload websocket.PlayerUpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode player update payload: %v", err)
	}

	return &payload
}

// ExpectDraftStarted waits for and decodes a DRAFT_STARTED message
func (c *WSClient) ExpectDraftStarted(timeout time.Duration) *websocket.DraftStartedPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypeDraftStarted, timeout)

	var payload websocket.DraftStartedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode draft started payload: %v", err)
	}

	return &payload
}

// ExpectPhaseChanged waits for and decodes a PHASE_CHANGED message
func (c *WSClient) ExpectPhaseChanged(timeout time.Duration) *websocket.PhaseChangedPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypePhaseChanged, timeout)

	var payload websocket.PhaseChangedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode phase changed payload: %v", err)
	}

	return &payload
}

// ExpectChampionSelected waits for and decodes a CHAMPION_SELECTED message
func (c *WSClient) ExpectChampionSelected(timeout time.Duration) *websocket.ChampionSelectedPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypeChampionSelected, timeout)

	var payload websocket.ChampionSelectedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode champion selected payload: %v", err)
	}

	return &payload
}

// ExpectDraftCompleted waits for and decodes a DRAFT_COMPLETED message
func (c *WSClient) ExpectDraftCompleted(timeout time.Duration) *websocket.DraftCompletedPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypeDraftCompleted, timeout)

	var payload websocket.DraftCompletedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode draft completed payload: %v", err)
	}

	return &payload
}

// ExpectError waits for and decodes an ERROR message
func (c *WSClient) ExpectError(timeout time.Duration) *websocket.ErrorPayload {
	c.t.Helper()

	msg := c.ExpectMessage(websocket.MessageTypeError, timeout)

	var payload websocket.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.t.Fatalf("failed to decode error payload: %v", err)
	}

	return &payload
}

// ExpectErrorWithCode waits for an error with a specific code
func (c *WSClient) ExpectErrorWithCode(code string, timeout time.Duration) *websocket.ErrorPayload {
	c.t.Helper()

	payload := c.ExpectError(timeout)
	if payload.Code != code {
		c.t.Fatalf("expected error code %s, got %s: %s", code, payload.Code, payload.Message)
	}

	return payload
}

// ExpectNoMessage verifies no messages are received within timeout
func (c *WSClient) ExpectNoMessage(timeout time.Duration) {
	c.t.Helper()

	select {
	case msg := <-c.messages:
		if msg != nil && msg.Type != websocket.MessageTypeTimerTick {
			c.t.Fatalf("unexpected message received: %s", msg.Type)
		}
	case <-time.After(timeout):
		// Expected - no message received
	}
}

// DrainMessages drains all pending messages from the channel with a timeout.
// It waits for messages to settle, then drains everything currently buffered.
func (c *WSClient) DrainMessages() {
	c.DrainMessagesWithTimeout(100 * time.Millisecond)
}

// DrainMessagesWithTimeout drains messages, waiting up to timeout for the channel to settle.
// This replaces the old sleep+drain pattern with a proper implementation.
func (c *WSClient) DrainMessagesWithTimeout(timeout time.Duration) {
	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.messages:
			if msg == nil {
				return
			}
			// Reset deadline when we receive a message - more might be coming
			deadline = time.After(50 * time.Millisecond)
		case <-deadline:
			// No messages for timeout duration, channel is settled
			return
		case <-c.done:
			return
		}
	}
}

// ExpectAnyMessage waits for any message to arrive and returns it
func (c *WSClient) ExpectAnyMessage(timeout time.Duration) *websocket.Message {
	c.t.Helper()

	select {
	case msg := <-c.messages:
		if msg == nil {
			c.t.Fatal("connection closed while waiting for message")
		}
		return msg
	case err := <-c.errors:
		c.t.Fatalf("error while waiting for message: %v", err)
	case <-time.After(timeout):
		c.t.Fatal("timeout waiting for any message")
	}
	return nil
}

// WaitForMessageCount waits until at least count messages have been received.
// It drains those messages and returns them.
func (c *WSClient) WaitForMessageCount(count int, timeout time.Duration) []*websocket.Message {
	c.t.Helper()

	messages := make([]*websocket.Message, 0, count)
	deadline := time.After(timeout)

	for len(messages) < count {
		select {
		case msg := <-c.messages:
			if msg == nil {
				c.t.Fatalf("connection closed after receiving %d/%d messages", len(messages), count)
			}
			// Skip TIMER_TICK messages as they're noise
			if msg.Type != websocket.MessageTypeTimerTick {
				messages = append(messages, msg)
			}
		case err := <-c.errors:
			c.t.Fatalf("error after receiving %d/%d messages: %v", len(messages), count, err)
		case <-deadline:
			c.t.Fatalf("timeout waiting for messages: got %d/%d", len(messages), count)
		}
	}

	return messages
}

// ExpectPlayerUpdateForSide waits for a PLAYER_UPDATE message for a specific side
func (c *WSClient) ExpectPlayerUpdateForSide(side string, timeout time.Duration) *websocket.PlayerUpdatePayload {
	c.t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.messages:
			if msg == nil {
				c.t.Fatalf("connection closed while waiting for PLAYER_UPDATE for %s", side)
			}
			if msg.Type == websocket.MessageTypePlayerUpdate {
				var payload websocket.PlayerUpdatePayload
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					c.t.Fatalf("failed to decode player update payload: %v", err)
				}
				if payload.Side == side {
					return &payload
				}
			}
			// Skip other message types
		case err := <-c.errors:
			c.t.Fatalf("error while waiting for PLAYER_UPDATE for %s: %v", side, err)
		case <-deadline:
			c.t.Fatalf("timeout waiting for PLAYER_UPDATE for side %s", side)
		}
	}
}

// ExpectMessagesOfTypes waits for messages of the specified types in any order.
// Returns a map of message type to the received message.
func (c *WSClient) ExpectMessagesOfTypes(types []websocket.MessageType, timeout time.Duration) map[websocket.MessageType]*websocket.Message {
	c.t.Helper()

	needed := make(map[websocket.MessageType]bool)
	for _, t := range types {
		needed[t] = true
	}

	result := make(map[websocket.MessageType]*websocket.Message)
	deadline := time.After(timeout)

	for len(result) < len(types) {
		select {
		case msg := <-c.messages:
			if msg == nil {
				c.t.Fatalf("connection closed while waiting for messages, got %d/%d", len(result), len(types))
			}
			if needed[msg.Type] && result[msg.Type] == nil {
				result[msg.Type] = msg
			}
			// Skip messages we're not looking for or already have
		case err := <-c.errors:
			c.t.Fatalf("error while waiting for messages: %v", err)
		case <-deadline:
			missing := []websocket.MessageType{}
			for _, t := range types {
				if result[t] == nil {
					missing = append(missing, t)
				}
			}
			c.t.Fatalf("timeout waiting for messages, missing: %v", missing)
		}
	}

	return result
}

// SkipUntilMessageType skips messages until finding one of the specified type
func (c *WSClient) SkipUntilMessageType(msgType websocket.MessageType, timeout time.Duration) *websocket.Message {
	c.t.Helper()

	deadline := time.After(timeout)
	for {
		select {
		case msg := <-c.messages:
			if msg == nil {
				c.t.Fatalf("connection closed while waiting for %s", msgType)
			}
			if msg.Type == msgType {
				return msg
			}
			// Skip other messages
		case err := <-c.errors:
			c.t.Fatalf("error while waiting for %s: %v", msgType, err)
		case <-deadline:
			c.t.Fatalf("timeout waiting for message type %s", msgType)
		}
	}
}

// WaitForConnection is a no-op since connection is established in NewWSClient.
// Kept for API compatibility.
func (c *WSClient) WaitForConnection(timeout time.Duration) {
	// Connection is already established in NewWSClient
	// No sleep needed - the dial succeeded or would have failed
}
