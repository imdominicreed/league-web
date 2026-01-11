package websocket

import (
	"encoding/json"
	"log"
)

// CommandHandler routes v2 COMMAND messages to the appropriate room handlers.
// It provides a bridge between the new protocol and the existing room channels.
type CommandHandler struct {
	client *Client
}

// NewCommandHandler creates a new command handler for a client.
func NewCommandHandler(client *Client) *CommandHandler {
	return &CommandHandler{client: client}
}

// HandleCommand processes a v2 COMMAND message.
func (ch *CommandHandler) HandleCommand(msg *Msg) {
	var cmd Command
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		ch.client.sendError("INVALID_COMMAND", "Invalid command format")
		return
	}

	switch cmd.Action {
	case CmdJoinRoom:
		ch.handleJoinRoom(cmd.Payload)
	case CmdSelectChampion:
		ch.handleSelectChampion(cmd.Payload)
	case CmdLockIn:
		ch.handleLockIn()
	case CmdHoverChampion:
		ch.handleHoverChampion(cmd.Payload)
	case CmdSetReady:
		ch.handleSetReady(cmd.Payload)
	case CmdStartDraft:
		ch.handleStartDraft()
	case CmdPauseDraft:
		ch.handlePauseDraft()
	case CmdResumeReady:
		ch.handleResumeReady(cmd.Payload)
	case CmdProposeEdit:
		ch.handleProposeEdit(cmd.Payload)
	case CmdRespondEdit:
		ch.handleRespondEdit(cmd.Payload)
	default:
		log.Printf("Unknown command action: %s", cmd.Action)
		ch.client.sendError("UNKNOWN_COMMAND", "Unknown command action")
	}
}

// HandleQuery processes a v2 QUERY message.
func (ch *CommandHandler) HandleQuery(msg *Msg) {
	var query Query
	if err := json.Unmarshal(msg.Payload, &query); err != nil {
		ch.client.sendError("INVALID_QUERY", "Invalid query format")
		return
	}

	switch query.Query {
	case QuerySyncState:
		if ch.client.room != nil {
			ch.client.room.syncState <- ch.client
		}
	default:
		ch.client.sendError("UNKNOWN_QUERY", "Unknown query type")
	}
}

func (ch *CommandHandler) handleJoinRoom(payload json.RawMessage) {
	var p CmdJoinRoomPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid join room payload")
		return
	}
	ch.client.hub.joinRoom <- &JoinRoomRequest{
		Client: ch.client,
		RoomID: p.RoomID,
		Side:   p.Side,
	}
}

func (ch *CommandHandler) handleSelectChampion(payload json.RawMessage) {
	var p CmdSelectChampionPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid select champion payload")
		return
	}
	if ch.client.room != nil {
		ch.client.room.selectChampion <- &SelectChampionRequest{
			Client:     ch.client,
			ChampionID: p.ChampionID,
		}
	}
}

func (ch *CommandHandler) handleLockIn() {
	if ch.client.room != nil {
		ch.client.room.lockIn <- ch.client
	}
}

func (ch *CommandHandler) handleHoverChampion(payload json.RawMessage) {
	var p CmdHoverChampionPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid hover champion payload")
		return
	}
	if ch.client.room != nil {
		ch.client.room.hoverChampion <- &HoverChampionRequest{
			Client:     ch.client,
			ChampionID: p.ChampionID,
		}
	}
}

func (ch *CommandHandler) handleSetReady(payload json.RawMessage) {
	var p CmdSetReadyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid set ready payload")
		return
	}
	if ch.client.room != nil {
		ch.client.room.ready <- &ReadyRequest{
			Client: ch.client,
			Ready:  p.Ready,
		}
	}
}

func (ch *CommandHandler) handleStartDraft() {
	if ch.client.room != nil {
		ch.client.room.startDraft <- ch.client
	}
}

func (ch *CommandHandler) handlePauseDraft() {
	if ch.client.room != nil {
		ch.client.room.pauseDraft <- ch.client
	}
}

func (ch *CommandHandler) handleResumeReady(payload json.RawMessage) {
	var p CmdResumeReadyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid resume ready payload")
		return
	}
	if ch.client.room != nil {
		ch.client.room.readyToResume <- &ReadyToResumeRequest{
			Client: ch.client,
			Ready:  p.Ready,
		}
	}
}

func (ch *CommandHandler) handleProposeEdit(payload json.RawMessage) {
	var p CmdProposeEditPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid propose edit payload")
		return
	}
	if ch.client.room != nil {
		ch.client.room.proposeEdit <- &ProposeEditRequest{
			Client: ch.client,
			Payload: ProposeEditPayload{
				SlotType:   p.SlotType,
				Team:       p.Team,
				SlotIndex:  p.SlotIndex,
				ChampionID: p.ChampionID,
			},
		}
	}
}

func (ch *CommandHandler) handleRespondEdit(payload json.RawMessage) {
	var p CmdRespondEditPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		ch.client.sendError("INVALID_PAYLOAD", "Invalid respond edit payload")
		return
	}
	if ch.client.room != nil {
		if p.Accept {
			ch.client.room.confirmEdit <- ch.client
		} else {
			ch.client.room.rejectEdit <- ch.client
		}
	}
}
