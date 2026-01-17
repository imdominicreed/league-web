import { useEffect, useRef, useCallback, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { RootState } from '@/store'
import {
  setLobby,
  setMatchOptions,
  setPendingAction,
  setTeamStats,
  setVotingStatus,
  updatePlayer,
  removePlayer,
  updatePlayerReady,
  updateStatus,
  updateSelectedOption,
  updateVoteCounts,
  updatePendingActionApproval,
  updateRoomId,
  updateCaptain,
} from '@/store/slices/lobbySlice'
import {
  LobbyWSMessage,
  LobbyStateSyncPayload,
  PlayerJoinedPayload,
  PlayerLeftPayload,
  PlayerReadyChangedPayload,
  StatusChangedPayload,
  MatchOptionsGeneratedPayload,
  TeamSelectedPayload,
  VoteCastPayload,
  ActionProposedPayload,
  ActionApprovedPayload,
  ActionExecutedPayload,
  ActionCancelledPayload,
  DraftStartingPayload,
  CaptainChangedPayload,
  PlayerKickedPayload,
  TeamStatsUpdatedPayload,
  VotingStatusUpdatedPayload,
  LobbyErrorPayload,
  toLobbyPlayer,
  toMatchOption,
  toTeamStats,
  toPendingAction,
  toVotingStatus,
} from '@/types/lobbyWebSocket'
import { Lobby, LobbyStatus, VotingMode } from '@/types'

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/lobby-ws`

export function useLobbyWebSocket(lobbyId: string | undefined) {
  const dispatch = useDispatch()
  const { accessToken, user } = useSelector((state: RootState) => state.auth)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleMessageRef = useRef<((msg: LobbyWSMessage) => void) | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Handler for state sync
  const handleStateSync = useCallback((payload: LobbyStateSyncPayload) => {
    console.log('[LobbyWS] Received state_sync, status:', payload.lobby.status, 'matchOptions:', payload.matchOptions?.length ?? 0)
    const lobby: Lobby = {
      id: payload.lobby.id,
      shortCode: payload.lobby.shortCode,
      createdBy: payload.lobby.createdBy,
      status: payload.lobby.status as LobbyStatus,
      selectedMatchOption: payload.lobby.selectedMatchOption,
      draftMode: payload.lobby.draftMode as 'pro_play' | 'fearless',
      timerDurationSeconds: payload.lobby.timerDurationSeconds,
      roomId: payload.lobby.roomId,
      votingEnabled: payload.lobby.votingEnabled,
      votingMode: payload.lobby.votingMode as VotingMode,
      votingDeadline: payload.lobby.votingDeadline,
      players: payload.players.map(toLobbyPlayer),
    }
    dispatch(setLobby(lobby))

    if (payload.matchOptions) {
      dispatch(setMatchOptions(payload.matchOptions.map(toMatchOption)))
    }

    if (payload.teamStats) {
      dispatch(setTeamStats(toTeamStats(payload.teamStats)))
    }

    if (payload.votingStatus) {
      dispatch(setVotingStatus(toVotingStatus(payload.votingStatus)))
    }

    dispatch(setPendingAction(toPendingAction(payload.pendingAction)))
  }, [dispatch])

  // Handler for player joined
  const handlePlayerJoined = useCallback((payload: PlayerJoinedPayload) => {
    dispatch(updatePlayer(toLobbyPlayer(payload.player)))
  }, [dispatch])

  // Handler for player left
  const handlePlayerLeft = useCallback((payload: PlayerLeftPayload) => {
    dispatch(removePlayer(payload.userId))
  }, [dispatch])

  // Handler for player ready changed
  const handlePlayerReadyChanged = useCallback((payload: PlayerReadyChangedPayload) => {
    dispatch(updatePlayerReady({ userId: payload.userId, isReady: payload.isReady }))
  }, [dispatch])

  // Handler for status changed
  const handleStatusChanged = useCallback((payload: StatusChangedPayload) => {
    dispatch(updateStatus(payload.newStatus))
  }, [dispatch])

  // Handler for match options generated
  const handleMatchOptionsGenerated = useCallback((payload: MatchOptionsGeneratedPayload) => {
    console.log('[LobbyWS] Received match_options_generated:', payload.options.length, 'options')
    dispatch(setMatchOptions(payload.options.map(toMatchOption)))
  }, [dispatch])

  // Handler for team selected
  const handleTeamSelected = useCallback((payload: TeamSelectedPayload) => {
    // Update players with their team assignments
    payload.assignments.forEach(player => {
      dispatch(updatePlayer(toLobbyPlayer(player)))
    })

    // Update team stats if provided
    if (payload.teamStats) {
      dispatch(setTeamStats(toTeamStats(payload.teamStats)))
    }

    // Update selected match option
    dispatch(updateSelectedOption(payload.optionNumber))
  }, [dispatch])

  // Handler for vote cast
  const handleVoteCast = useCallback((payload: VoteCastPayload) => {
    // Convert voters to the expected format
    const voters: Record<number, { userId: string; displayName: string }[]> = {}
    if (payload.voters) {
      for (const [optNum, voterList] of Object.entries(payload.voters)) {
        voters[parseInt(optNum)] = voterList.map(v => ({
          userId: v.userId,
          displayName: v.displayName,
        }))
      }
    }
    dispatch(updateVoteCounts({
      voteCounts: payload.voteCounts,
      votesCast: payload.votesCast,
      voters,
      // Pass vote info for updating current user's votes
      votingUserId: payload.userId,
      optionNumber: payload.optionNumber,
      voteAdded: payload.voteAdded,
      currentUserId: user?.id,
    }))
  }, [dispatch, user?.id])

  // Handler for action proposed
  const handleActionProposed = useCallback((payload: ActionProposedPayload) => {
    dispatch(setPendingAction(toPendingAction(payload.action)))
  }, [dispatch])

  // Handler for action approved
  const handleActionApproved = useCallback((payload: ActionApprovedPayload) => {
    dispatch(updatePendingActionApproval({
      approvedByBlue: payload.approvedByBlue,
      approvedByRed: payload.approvedByRed,
    }))
  }, [dispatch])

  // Handler for action executed
  const handleActionExecuted = useCallback((_payload: ActionExecutedPayload) => {
    dispatch(setPendingAction(null))
  }, [dispatch])

  // Handler for action cancelled
  const handleActionCancelled = useCallback((_payload: ActionCancelledPayload) => {
    dispatch(setPendingAction(null))
  }, [dispatch])

  // Handler for draft starting
  const handleDraftStarting = useCallback((payload: DraftStartingPayload) => {
    dispatch(updateRoomId(payload.roomId))
    dispatch(updateStatus('drafting'))
  }, [dispatch])

  // Handler for captain changed
  const handleCaptainChanged = useCallback((payload: CaptainChangedPayload) => {
    dispatch(updateCaptain({
      team: payload.team,
      newCaptainId: payload.newCaptainId,
      oldCaptainId: payload.oldCaptainId,
    }))
  }, [dispatch])

  // Handler for player kicked
  const handlePlayerKicked = useCallback((payload: PlayerKickedPayload) => {
    dispatch(removePlayer(payload.userId))
  }, [dispatch])

  // Handler for team stats updated
  const handleTeamStatsUpdated = useCallback((payload: TeamStatsUpdatedPayload) => {
    dispatch(setTeamStats(toTeamStats(payload.stats)))
  }, [dispatch])

  // Handler for voting status updated
  const handleVotingStatusUpdated = useCallback((payload: VotingStatusUpdatedPayload) => {
    dispatch(setVotingStatus(toVotingStatus(payload.status)))
  }, [dispatch])

  // Handler for errors
  const handleError = useCallback((payload: LobbyErrorPayload) => {
    console.error('Lobby WebSocket error:', payload.code, payload.message)
    setError(payload.message)
  }, [])

  // Message router
  const handleMessage = useCallback((msg: LobbyWSMessage) => {
    switch (msg.type) {
      case 'lobby_state_sync':
        handleStateSync(msg.payload as LobbyStateSyncPayload)
        break
      case 'player_joined':
        handlePlayerJoined(msg.payload as PlayerJoinedPayload)
        break
      case 'player_left':
        handlePlayerLeft(msg.payload as PlayerLeftPayload)
        break
      case 'player_ready_changed':
        handlePlayerReadyChanged(msg.payload as PlayerReadyChangedPayload)
        break
      case 'status_changed':
        handleStatusChanged(msg.payload as StatusChangedPayload)
        break
      case 'match_options_generated':
        handleMatchOptionsGenerated(msg.payload as MatchOptionsGeneratedPayload)
        break
      case 'team_selected':
        handleTeamSelected(msg.payload as TeamSelectedPayload)
        break
      case 'vote_cast':
        handleVoteCast(msg.payload as VoteCastPayload)
        break
      case 'action_proposed':
        handleActionProposed(msg.payload as ActionProposedPayload)
        break
      case 'action_approved':
        handleActionApproved(msg.payload as ActionApprovedPayload)
        break
      case 'action_executed':
        handleActionExecuted(msg.payload as ActionExecutedPayload)
        break
      case 'action_cancelled':
        handleActionCancelled(msg.payload as ActionCancelledPayload)
        break
      case 'draft_starting':
        handleDraftStarting(msg.payload as DraftStartingPayload)
        break
      case 'captain_changed':
        handleCaptainChanged(msg.payload as CaptainChangedPayload)
        break
      case 'player_kicked':
        handlePlayerKicked(msg.payload as PlayerKickedPayload)
        break
      case 'team_stats_updated':
        handleTeamStatsUpdated(msg.payload as TeamStatsUpdatedPayload)
        break
      case 'voting_status_updated':
        handleVotingStatusUpdated(msg.payload as VotingStatusUpdatedPayload)
        break
      case 'error':
        handleError(msg.payload as LobbyErrorPayload)
        break
      default:
        console.warn('Unknown lobby WebSocket message type:', msg.type)
    }
  }, [
    handleStateSync,
    handlePlayerJoined,
    handlePlayerLeft,
    handlePlayerReadyChanged,
    handleStatusChanged,
    handleMatchOptionsGenerated,
    handleTeamSelected,
    handleVoteCast,
    handleActionProposed,
    handleActionApproved,
    handleActionExecuted,
    handleActionCancelled,
    handleDraftStarting,
    handleCaptainChanged,
    handlePlayerKicked,
    handleTeamStatsUpdated,
    handleVotingStatusUpdated,
    handleError,
  ])

  // Keep handleMessage ref updated to avoid stale closures
  useEffect(() => {
    handleMessageRef.current = handleMessage
  }, [handleMessage])

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (!accessToken || !lobbyId) return

    // Validate lobbyId is a valid UUID before connecting
    const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
    if (!uuidRegex.test(lobbyId)) {
      console.error('[LobbyWS] Invalid lobby ID format:', lobbyId)
      return
    }

    const ws = new WebSocket(`${WS_URL}?token=${accessToken}`)
    wsRef.current = ws

    ws.onopen = () => {
      console.log('[LobbyWS] Connected, joining lobby:', lobbyId)
      setIsConnected(true)
      setError(null)

      // Join the lobby
      ws.send(JSON.stringify({
        type: 'join_lobby',
        payload: { lobbyId },
        timestamp: Date.now(),
      }))
    }

    ws.onmessage = (event) => {
      try {
        const message: LobbyWSMessage = JSON.parse(event.data)
        // Use ref to always call the latest handler (avoids stale closure)
        handleMessageRef.current?.(message)
      } catch (err) {
        console.error('Failed to parse lobby WebSocket message:', err)
      }
    }

    ws.onclose = () => {
      setIsConnected(false)

      // Reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }

    ws.onerror = (event) => {
      console.error('Lobby WebSocket connection error:', event)
      setError('WebSocket connection error')
    }
  }, [accessToken, lobbyId])

  // Effect to manage connection lifecycle
  useEffect(() => {
    if (lobbyId && accessToken) {
      connect()
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      wsRef.current?.close()
    }
  }, [connect, lobbyId, accessToken])

  return {
    isConnected,
    error,
  }
}
