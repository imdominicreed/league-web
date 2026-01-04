import { useEffect, useRef, useCallback, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { RootState } from '@/store'
import { syncState, championSelected, phaseChanged, updateTimer, championHovered, draftCompleted } from '@/store/slices/draftSlice'
import { syncRoom, playerUpdate, setConnectionStatus, setError } from '@/store/slices/roomSlice'
import { WSMessage, StateSyncPayload } from '@/types'

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws`

export function useWebSocket(roomId: string, side: string) {
  const dispatch = useDispatch()
  const { accessToken } = useSelector((state: RootState) => state.auth)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  const connect = useCallback(() => {
    if (!accessToken || !roomId || !side) return

    dispatch(setConnectionStatus('connecting'))

    const ws = new WebSocket(`${WS_URL}?token=${accessToken}`)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      dispatch(setConnectionStatus('connected'))

      // Join room with assigned side
      ws.send(JSON.stringify({
        type: 'JOIN_ROOM',
        payload: { roomId, side },
        timestamp: Date.now(),
      }))
    }

    ws.onmessage = (event) => {
      const message: WSMessage = JSON.parse(event.data)
      handleMessage(message)
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
  }, [accessToken, roomId, side, dispatch])

  const handleMessage = (message: WSMessage) => {
    switch (message.type) {
      case 'STATE_SYNC': {
        const payload = message.payload as StateSyncPayload
        dispatch(syncState({
          ...payload.draft,
          yourSide: payload.yourSide,
          fearlessBans: payload.fearlessBans,
        }))
        dispatch(syncRoom({
          room: {
            id: payload.room.id,
            shortCode: payload.room.shortCode,
            draftMode: payload.room.draftMode as 'pro_play' | 'fearless',
            timerDurationSeconds: payload.room.timerDuration / 1000,
            status: payload.room.status as 'waiting' | 'in_progress' | 'completed',
          },
          players: payload.players,
          spectatorCount: payload.spectatorCount,
        }))
        break
      }
      case 'PLAYER_UPDATE':
        dispatch(playerUpdate(message.payload as { side: string; player: { userId: string; displayName: string; ready: boolean } | null; action: string }))
        break
      case 'DRAFT_STARTED':
      case 'PHASE_CHANGED':
        dispatch(phaseChanged(message.payload as { currentPhase: number; currentTeam: string; actionType: string; timerRemainingMs: number }))
        break
      case 'CHAMPION_SELECTED':
        dispatch(championSelected(message.payload as { phase: number; team: string; actionType: string; championId: string }))
        break
      case 'CHAMPION_HOVERED':
        dispatch(championHovered(message.payload as { side: string; championId: string | null }))
        break
      case 'TIMER_TICK':
        dispatch(updateTimer(message.payload as { remainingMs: number }))
        break
      case 'DRAFT_COMPLETED':
        dispatch(draftCompleted(message.payload as { blueBans: string[]; redBans: string[]; bluePicks: string[]; redPicks: string[] }))
        break
      case 'ERROR':
        console.error('Server error message:', (message.payload as { message: string }).message)
        dispatch(setError((message.payload as { message: string }).message))
        break
    }
  }

  const send = useCallback((type: string, payload: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type,
        payload,
        timestamp: Date.now(),
      }))
    }
  }, [])

  const selectChampion = useCallback((championId: string) => {
    send('SELECT_CHAMPION', { championId })
  }, [send])

  const lockIn = useCallback(() => {
    send('LOCK_IN', {})
  }, [send])

  const hoverChampion = useCallback((championId: string | null) => {
    send('HOVER_CHAMPION', { championId })
  }, [send])

  const setReady = useCallback((ready: boolean) => {
    send('READY', { ready })
  }, [send])

  const startDraft = useCallback(() => {
    send('START_DRAFT', {})
  }, [send])

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
    send,
  }
}
