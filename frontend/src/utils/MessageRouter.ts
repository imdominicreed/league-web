/**
 * MessageRouter - Type-safe WebSocket message routing
 *
 * Replaces the large switch statement in useWebSocket.ts with a handler registry pattern.
 * Supports both the old protocol (for backwards compatibility) and the new v2 protocol.
 */

import type { WSMessage } from '@/types'
import type {
  MsgType,
  Msg,
  Event,
  EventType,
  StatePayload,
  Timer,
  Err,
  EventHandler,
  EvtDraftStartedPayload,
  EvtDraftCompletedPayload,
  EvtPhaseChangedPayload,
  EvtChampionSelectedPayload,
  EvtChampionHoveredPayload,
  EvtPlayerJoinedPayload,
  EvtPlayerLeftPayload,
  EvtPlayerReadyChangedPayload,
  EvtDraftPausedPayload,
  EvtDraftResumedPayload,
  EvtResumeReadyChangedPayload,
  EvtResumeCountdownPayload,
  EvtEditProposedPayload,
  EvtEditAppliedPayload,
  EvtEditRejectedPayload,
} from '@/types/websocket'

/**
 * Handler types for the old protocol message types
 */
export interface OldProtocolHandlers {
  STATE_SYNC?: (payload: unknown) => void
  PLAYER_UPDATE?: (payload: unknown) => void
  DRAFT_STARTED?: (payload: unknown) => void
  PHASE_CHANGED?: (payload: unknown) => void
  CHAMPION_SELECTED?: (payload: unknown) => void
  CHAMPION_HOVERED?: (payload: unknown) => void
  TIMER_TICK?: (payload: unknown) => void
  DRAFT_COMPLETED?: (payload: unknown) => void
  DRAFT_PAUSED?: (payload: unknown) => void
  DRAFT_RESUMED?: (payload: unknown) => void
  EDIT_PROPOSED?: (payload: unknown) => void
  EDIT_APPLIED?: (payload: unknown) => void
  EDIT_REJECTED?: (payload: unknown) => void
  RESUME_READY_UPDATE?: (payload: unknown) => void
  RESUME_COUNTDOWN?: (payload: unknown) => void
  ERROR?: (payload: unknown) => void
}

/**
 * Handler types for the new v2 protocol
 */
export interface V2EventHandlers {
  draft_started?: EventHandler<EvtDraftStartedPayload>
  draft_completed?: EventHandler<EvtDraftCompletedPayload>
  phase_changed?: EventHandler<EvtPhaseChangedPayload>
  champion_selected?: EventHandler<EvtChampionSelectedPayload>
  champion_hovered?: EventHandler<EvtChampionHoveredPayload>
  player_joined?: EventHandler<EvtPlayerJoinedPayload>
  player_left?: EventHandler<EvtPlayerLeftPayload>
  player_ready_changed?: EventHandler<EvtPlayerReadyChangedPayload>
  draft_paused?: EventHandler<EvtDraftPausedPayload>
  draft_resumed?: EventHandler<EvtDraftResumedPayload>
  resume_ready_changed?: EventHandler<EvtResumeReadyChangedPayload>
  resume_countdown?: EventHandler<EvtResumeCountdownPayload>
  edit_proposed?: EventHandler<EvtEditProposedPayload>
  edit_applied?: EventHandler<EvtEditAppliedPayload>
  edit_rejected?: EventHandler<EvtEditRejectedPayload>
}

export interface V2Handlers extends V2EventHandlers {
  onState?: (state: StatePayload) => void
  onTimer?: (timer: Timer) => void
  onError?: (error: Err) => void
}

/**
 * MessageRouter class for handling WebSocket messages
 *
 * Supports both old and new protocols with type-safe handler registration.
 */
export class MessageRouter {
  private oldHandlers: Map<string, (payload: unknown) => void> = new Map()
  private eventHandlers: Map<EventType, EventHandler<unknown>> = new Map()
  private stateHandler: ((state: StatePayload) => void) | null = null
  private timerHandler: ((timer: Timer) => void) | null = null
  private errorHandler: ((error: Err) => void) | null = null

  /**
   * Register handlers for old protocol message types
   */
  registerOldHandlers(handlers: OldProtocolHandlers): this {
    for (const [type, handler] of Object.entries(handlers)) {
      if (handler) {
        this.oldHandlers.set(type, handler)
      }
    }
    return this
  }

  /**
   * Register a handler for an old protocol message type
   */
  onOldMessage(type: string, handler: (payload: unknown) => void): this {
    this.oldHandlers.set(type, handler)
    return this
  }

  /**
   * Register handlers for v2 protocol events
   */
  registerEventHandlers(handlers: V2EventHandlers): this {
    for (const [event, handler] of Object.entries(handlers)) {
      if (handler) {
        this.eventHandlers.set(event as EventType, handler)
      }
    }
    return this
  }

  /**
   * Register a handler for a v2 protocol event
   */
  onEvent<T>(event: EventType, handler: EventHandler<T>): this {
    this.eventHandlers.set(event, handler as EventHandler<unknown>)
    return this
  }

  /**
   * Register a handler for v2 STATE messages
   */
  onState(handler: (state: StatePayload) => void): this {
    this.stateHandler = handler
    return this
  }

  /**
   * Register a handler for v2 TIMER messages
   */
  onTimer(handler: (timer: Timer) => void): this {
    this.timerHandler = handler
    return this
  }

  /**
   * Register a handler for v2 ERR messages
   */
  onError(handler: (error: Err) => void): this {
    this.errorHandler = handler
    return this
  }

  /**
   * Register all v2 handlers at once
   */
  registerV2Handlers(handlers: V2Handlers): this {
    const { onState, onTimer, onError, ...eventHandlers } = handlers
    if (onState) this.stateHandler = onState
    if (onTimer) this.timerHandler = onTimer
    if (onError) this.errorHandler = onError
    this.registerEventHandlers(eventHandlers)
    return this
  }

  /**
   * Route an incoming message to the appropriate handler
   *
   * Automatically detects old vs new protocol based on message structure
   */
  handle(message: WSMessage | Msg): void {
    // Detect protocol version based on message type
    if (this.isV2Message(message)) {
      this.handleV2Message(message as Msg)
    } else {
      this.handleOldMessage(message as WSMessage)
    }
  }

  /**
   * Check if a message is using the v2 protocol
   */
  private isV2Message(message: WSMessage | Msg): boolean {
    const v2Types: MsgType[] = ['COMMAND', 'QUERY', 'EVENT', 'STATE', 'TIMER', 'ERR']
    return v2Types.includes(message.type as MsgType)
  }

  /**
   * Handle old protocol messages
   */
  private handleOldMessage(message: WSMessage): void {
    const handler = this.oldHandlers.get(message.type)
    if (handler) {
      handler(message.payload)
    } else {
      console.warn(`No handler registered for message type: ${message.type}`)
    }
  }

  /**
   * Handle v2 protocol messages
   */
  private handleV2Message(message: Msg): void {
    switch (message.type) {
      case 'EVENT': {
        const event = message.payload as Event
        const handler = this.eventHandlers.get(event.event)
        if (handler) {
          handler(event.payload)
        } else {
          console.warn(`No handler registered for event: ${event.event}`)
        }
        break
      }
      case 'STATE': {
        if (this.stateHandler) {
          this.stateHandler(message.payload as StatePayload)
        }
        break
      }
      case 'TIMER': {
        if (this.timerHandler) {
          this.timerHandler(message.payload as Timer)
        }
        break
      }
      case 'ERR': {
        if (this.errorHandler) {
          this.errorHandler(message.payload as Err)
        }
        break
      }
      default:
        console.warn(`Unknown v2 message type: ${message.type}`)
    }
  }

  /**
   * Clear all handlers
   */
  clear(): void {
    this.oldHandlers.clear()
    this.eventHandlers.clear()
    this.stateHandler = null
    this.timerHandler = null
    this.errorHandler = null
  }
}

/**
 * Create a pre-configured MessageRouter for the current (old) protocol
 *
 * This is a helper to easily migrate from the switch statement.
 */
export function createOldProtocolRouter(handlers: OldProtocolHandlers): MessageRouter {
  return new MessageRouter().registerOldHandlers(handlers)
}

/**
 * Create a pre-configured MessageRouter for the v2 protocol
 */
export function createV2Router(handlers: V2Handlers): MessageRouter {
  return new MessageRouter().registerV2Handlers(handlers)
}
