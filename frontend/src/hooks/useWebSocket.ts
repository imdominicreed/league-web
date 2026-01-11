import { useEffect, useRef, useCallback, useState, useMemo } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { RootState } from '@/store'
import {
  syncState,
  championSelected,
  phaseChanged,
  updateTimer,
  championHovered,
  draftCompleted,
  draftPaused,
  draftResumed,
  editProposed,
  editApplied,
  editRejected,
  resumeReadyUpdate,
  resumeCountdownUpdate,
} from '@/store/slices/draftSlice'
import { syncRoom, playerUpdate, updateRoomStatus, setConnectionStatus, setError } from '@/store/slices/roomSlice'
import { WSMessage, StateSyncPayload } from '@/types'
import { MessageRouter } from '@/utils/MessageRouter'
import type { CommandAction } from '@/types/websocket'

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws`

export function useWebSocket(roomId: string, side: string) {
  const dispatch = useDispatch()
  const { accessToken } = useSelector((state: RootState) => state.auth)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  // Create message router with handlers for server responses (still old format)
  const router = useMemo(() => {
    return new MessageRouter().registerOldHandlers({
      STATE_SYNC: (payload) => {
        const p = payload as StateSyncPayload
        dispatch(syncState({
          ...p.draft,
          yourSide: p.yourSide,
          fearlessBans: p.fearlessBans,
          teamPlayers: p.teamPlayers,
          isTeamDraft: p.isTeamDraft,
          isPaused: p.draft.isPaused ?? false,
          pausedBy: p.draft.pausedBy ?? null,
          pausedBySide: p.draft.pausedBySide ?? null,
          pendingEdit: p.draft.pendingEdit ?? null,
          blueResumeReady: p.draft.blueResumeReady ?? false,
          redResumeReady: p.draft.redResumeReady ?? false,
          resumeCountdown: p.draft.resumeCountdown ?? 0,
        }))
        dispatch(syncRoom({
          room: {
            id: p.room.id,
            shortCode: p.room.shortCode,
            draftMode: p.room.draftMode as 'pro_play' | 'fearless',
            timerDurationSeconds: p.room.timerDuration / 1000,
            status: p.room.status as 'waiting' | 'in_progress' | 'completed',
          },
          players: p.players,
          spectatorCount: p.spectatorCount,
          isCaptain: p.isCaptain,
          isTeamDraft: p.isTeamDraft,
        }))
      },
      PLAYER_UPDATE: (payload) => {
        dispatch(playerUpdate(payload as { side: string; player: { userId: string; displayName: string; ready: boolean } | null; action: string }))
      },
      DRAFT_STARTED: (payload) => {
        dispatch(updateRoomStatus('in_progress'))
        dispatch(phaseChanged(payload as { currentPhase: number; currentTeam: string; actionType: string; timerRemainingMs: number }))
      },
      PHASE_CHANGED: (payload) => {
        dispatch(phaseChanged(payload as { currentPhase: number; currentTeam: string; actionType: string; timerRemainingMs: number }))
      },
      CHAMPION_SELECTED: (payload) => {
        dispatch(championSelected(payload as { phase: number; team: string; actionType: string; championId: string }))
      },
      CHAMPION_HOVERED: (payload) => {
        dispatch(championHovered(payload as { side: string; championId: string | null }))
      },
      TIMER_TICK: (payload) => {
        dispatch(updateTimer(payload as { remainingMs: number }))
      },
      DRAFT_COMPLETED: (payload) => {
        dispatch(updateRoomStatus('completed'))
        dispatch(draftCompleted(payload as { blueBans: string[]; redBans: string[]; bluePicks: string[]; redPicks: string[] }))
      },
      ERROR: (payload) => {
        const p = payload as { message: string }
        console.error('Server error:', p.message)
        dispatch(setError(p.message))
      },
      DRAFT_PAUSED: (payload) => {
        dispatch(draftPaused(payload as { pausedBy: string; pausedBySide: 'blue' | 'red'; timerFrozenAt: number }))
      },
      DRAFT_RESUMED: (payload) => {
        dispatch(draftResumed(payload as { timerRemainingMs: number }))
      },
      EDIT_PROPOSED: (payload) => {
        dispatch(editProposed(payload as {
          proposedBy: string; proposedSide: 'blue' | 'red'; slotType: 'ban' | 'pick'
          team: 'blue' | 'red'; slotIndex: number; oldChampionId: string; newChampionId: string; expiresAt: number
        }))
      },
      EDIT_APPLIED: (payload) => {
        dispatch(editApplied(payload as {
          slotType: 'ban' | 'pick'; team: 'blue' | 'red'; slotIndex: number; newChampionId: string
          blueBans: string[]; redBans: string[]; bluePicks: string[]; redPicks: string[]
        }))
      },
      EDIT_REJECTED: () => {
        dispatch(editRejected())
      },
      RESUME_READY_UPDATE: (payload) => {
        dispatch(resumeReadyUpdate(payload as { blueReady: boolean; redReady: boolean }))
      },
      RESUME_COUNTDOWN: (payload) => {
        dispatch(resumeCountdownUpdate(payload as { secondsRemaining: number; cancelledBy?: string }))
      },
    })
  }, [dispatch])

  // Send a v2 COMMAND message
  const sendCommand = useCallback((action: CommandAction, payload?: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: 'COMMAND',
        payload: { action, ...(payload ? { payload } : {}) },
        timestamp: Date.now(),
      }))
    }
  }, [])

  const connect = useCallback(() => {
    if (!accessToken || !roomId || !side) return

    dispatch(setConnectionStatus('connecting'))

    const ws = new WebSocket(`${WS_URL}?token=${accessToken}`)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      dispatch(setConnectionStatus('connected'))

      // Join room with v2 COMMAND
      sendCommand('join_room', { roomId, side })
    }

    ws.onmessage = (event) => {
      const message: WSMessage = JSON.parse(event.data)
      router.handle(message)
    }

    ws.onclose = () => {
      setIsConnected(false)
      dispatch(setConnectionStatus('disconnected'))

      // Reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }

    ws.onerror = (event) => {
      console.error('WebSocket connection error:', event)
      dispatch(setError('WebSocket connection error'))
    }
  }, [accessToken, roomId, side, dispatch, router, sendCommand])

  // Action handlers using v2 COMMAND format
  const selectChampion = useCallback((championId: string) => {
    sendCommand('select_champion', { championId })
  }, [sendCommand])

  const lockIn = useCallback(() => {
    sendCommand('lock_in')
  }, [sendCommand])

  const hoverChampion = useCallback((championId: string | null) => {
    sendCommand('hover_champion', { championId })
  }, [sendCommand])

  const setReady = useCallback((ready: boolean) => {
    sendCommand('set_ready', { ready })
  }, [sendCommand])

  const startDraft = useCallback(() => {
    sendCommand('start_draft')
  }, [sendCommand])

  const pauseDraft = useCallback(() => {
    sendCommand('pause_draft')
  }, [sendCommand])

  const resumeDraft = useCallback(() => {
    sendCommand('resume_ready', { ready: true })
  }, [sendCommand])

  const proposeEdit = useCallback((slotType: 'ban' | 'pick', team: 'blue' | 'red', slotIndex: number, championId: string) => {
    sendCommand('propose_edit', { slotType, team, slotIndex, championId })
  }, [sendCommand])

  const confirmEdit = useCallback(() => {
    sendCommand('respond_edit', { accept: true })
  }, [sendCommand])

  const rejectEdit = useCallback(() => {
    sendCommand('respond_edit', { accept: false })
  }, [sendCommand])

  const readyToResume = useCallback((ready: boolean) => {
    sendCommand('resume_ready', { ready })
  }, [sendCommand])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      wsRef.current?.close()
    }
  }, [connect])

  return {
    isConnected,
    selectChampion,
    lockIn,
    hoverChampion,
    setReady,
    startDraft,
    pauseDraft,
    resumeDraft,
    proposeEdit,
    confirmEdit,
    rejectEdit,
    readyToResume,
  }
}
