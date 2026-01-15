import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { LobbyPlayer } from '@/types'
import {
  fetchLobby,
  fetchMatchOptions,
  setReady,
  startDraft,
  takeCaptain,
  promoteCaptain,
  kickPlayer,
  proposeSwap,
  proposeMatchmake,
  proposeSelectOption,
  proposeStartDraft,
  fetchPendingAction,
  approvePendingAction,
  cancelPendingAction,
  fetchTeamStats,
  castVote,
  fetchVotingStatus,
  endVoting,
} from '@/store/slices/lobbySlice'
import { TeamColumn } from '@/components/lobby/TeamColumn'
import { PendingActionBanner } from '@/components/lobby/PendingActionBanner'
import { TeamStatsPanel } from '@/components/lobby/TeamStatsPanel'
import { CaptainControls } from '@/components/lobby/CaptainControls'
import { MatchOptionCard } from '@/components/lobby/MatchOptionCard'
import { VotingBanner } from '@/components/lobby/VotingBanner'

export default function LobbyRoom() {
  const { lobbyId } = useParams<{ lobbyId: string }>()
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()

  const {
    lobby,
    matchOptions,
    pendingAction,
    teamStats,
    votingStatus,
    loading,
    error,
    startingDraft,
    createdRoom,
    takingCaptain,
    promotingCaptain,
    kickingPlayer,
    proposingAction,
    approvingAction,
    cancellingAction,
    fetchingTeamStats,
    castingVote,
    endingVoting,
  } = useSelector((state: RootState) => state.lobby)
  const { user } = useSelector((state: RootState) => state.auth)

  const [pollInterval, setPollInterval] = useState<ReturnType<typeof setInterval> | null>(null)

  // Swap mode state
  const [swapMode, setSwapMode] = useState(false)
  const [selectedForSwap, setSelectedForSwap] = useState<string | null>(null)

  // Polling for lobby state
  useEffect(() => {
    if (lobbyId) {
      dispatch(fetchLobby(lobbyId))
      dispatch(fetchPendingAction(lobbyId))
      const interval = setInterval(() => {
        dispatch(fetchLobby(lobbyId))
        dispatch(fetchPendingAction(lobbyId))
      }, 3000)
      setPollInterval(interval)
      return () => clearInterval(interval)
    }
  }, [lobbyId, dispatch])

  // Fetch match options when lobby is in matchmaking/team_selected status
  useEffect(() => {
    if (lobby && (lobby.status === 'matchmaking' || lobby.status === 'team_selected') && !matchOptions) {
      dispatch(fetchMatchOptions(lobby.id))
    }
  }, [lobby, matchOptions, dispatch])

  // Fetch team stats when teams are assigned
  useEffect(() => {
    if (lobby?.status === 'team_selected' && !teamStats) {
      dispatch(fetchTeamStats(lobby.id))
    }
  }, [lobby, teamStats, dispatch])

  // Fetch voting status when voting is enabled
  useEffect(() => {
    if (lobby?.votingEnabled && lobby.status === 'matchmaking') {
      dispatch(fetchVotingStatus(lobby.id))
      // Poll voting status more frequently when voting is active
      const votingPollInterval = setInterval(() => {
        dispatch(fetchVotingStatus(lobby.id))
      }, 2000)
      return () => clearInterval(votingPollInterval)
    }
  }, [lobby?.id, lobby?.votingEnabled, lobby?.status, dispatch])

  // Navigate to draft when it starts
  useEffect(() => {
    if (lobby?.status === 'drafting' && lobby.roomId) {
      if (pollInterval) clearInterval(pollInterval)
      navigate(`/draft/${lobby.roomId}`)
    }
  }, [lobby, navigate, pollInterval])

  useEffect(() => {
    if (createdRoom) {
      if (pollInterval) clearInterval(pollInterval)
      navigate(`/draft/${createdRoom.id}`)
    }
  }, [createdRoom, navigate, pollInterval])

  // Get current user's player info
  const currentPlayer = lobby?.players.find(p => p.userId === user?.id)
  const currentUserSide = currentPlayer?.team || null
  const isCaptain = currentPlayer?.isCaptain || false
  const isReady = currentPlayer?.isReady || false
  const hasTeams = lobby?.status === 'team_selected'
  const isMatchmaking = lobby?.status === 'matchmaking'

  // Handlers
  const handleReady = useCallback((ready: boolean) => {
    if (lobbyId) dispatch(setReady({ idOrCode: lobbyId, ready }))
  }, [lobbyId, dispatch])

  const handleTakeCaptain = useCallback(() => {
    if (lobby) dispatch(takeCaptain(lobby.id))
  }, [lobby, dispatch])

  const handlePromoteCaptain = useCallback((userId: string) => {
    if (lobby) dispatch(promoteCaptain({ lobbyId: lobby.id, userId }))
  }, [lobby, dispatch])

  const handleKickPlayer = useCallback((userId: string) => {
    if (lobby) dispatch(kickPlayer({ lobbyId: lobby.id, userId }))
  }, [lobby, dispatch])

  const handleProposeSwap = useCallback((player1Id: string, player2Id: string, swapType: 'players' | 'roles') => {
    if (lobby) {
      dispatch(proposeSwap({ lobbyId: lobby.id, player1Id, player2Id, swapType }))
      // Reset swap mode after proposing
      setSwapMode(false)
      setSelectedForSwap(null)
    }
  }, [lobby, dispatch])

  // Handle player click in swap mode
  const handlePlayerClick = useCallback((player: LobbyPlayer) => {
    if (!swapMode) return

    if (!selectedForSwap) {
      // First selection
      setSelectedForSwap(player.id)
    } else if (selectedForSwap === player.id) {
      // Clicked same player, deselect
      setSelectedForSwap(null)
    } else {
      // Second selection - auto-detect swap type and propose
      const firstPlayer = lobby?.players.find(p => p.id === selectedForSwap)
      const secondPlayer = player

      if (!firstPlayer) {
        setSelectedForSwap(null)
        return
      }

      // Auto-detect swap type based on teams
      const swapType = firstPlayer.team === secondPlayer.team ? 'roles' : 'players'
      handleProposeSwap(firstPlayer.userId, secondPlayer.userId, swapType)
    }
  }, [swapMode, selectedForSwap, lobby?.players, handleProposeSwap])

  // Cancel swap mode on Escape
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && swapMode) {
        setSwapMode(false)
        setSelectedForSwap(null)
      }
    }
    window.addEventListener('keydown', handleEscape)
    return () => window.removeEventListener('keydown', handleEscape)
  }, [swapMode])

  // Toggle swap mode handler
  const handleToggleSwapMode = useCallback(() => {
    setSwapMode(prev => !prev)
    setSelectedForSwap(null)
  }, [])

  const handleProposeMatchmake = useCallback(() => {
    if (lobby) dispatch(proposeMatchmake(lobby.id))
  }, [lobby, dispatch])

  const handleProposeStartDraft = useCallback(() => {
    if (lobby) dispatch(proposeStartDraft(lobby.id))
  }, [lobby, dispatch])

  const handleApprovePendingAction = useCallback(() => {
    if (lobby && pendingAction) {
      dispatch(approvePendingAction({ lobbyId: lobby.id, actionId: pendingAction.id }))
    }
  }, [lobby, pendingAction, dispatch])

  const handleCancelPendingAction = useCallback(() => {
    if (lobby && pendingAction) {
      dispatch(cancelPendingAction({ lobbyId: lobby.id, actionId: pendingAction.id }))
    }
  }, [lobby, pendingAction, dispatch])

  const handleProposeSelectOption = useCallback((optionNumber: number) => {
    if (lobby) {
      dispatch(proposeSelectOption({ lobbyId: lobby.id, optionNumber }))
    }
  }, [lobby, dispatch])

  const handleStartDraft = useCallback(() => {
    if (lobby) dispatch(startDraft(lobby.id))
  }, [lobby, dispatch])

  const handleCastVote = useCallback((optionNumber: number) => {
    if (lobby) dispatch(castVote({ lobbyId: lobby.id, optionNumber }))
  }, [lobby, dispatch])

  const handleEndVoting = useCallback((forceOption?: number) => {
    if (lobby) dispatch(endVoting({ lobbyId: lobby.id, forceOption }))
  }, [lobby, dispatch])

  if (loading && !lobby) {
    return <div className="min-h-screen flex items-center justify-center text-gray-400">Loading...</div>
  }

  if (!lobby) {
    return <div className="min-h-screen flex items-center justify-center text-red-400">Lobby not found</div>
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-lol-gold">10-Man Lobby</h1>
            <p className="text-gray-400">
              Code: <span className="text-white font-mono" data-testid="lobby-code-display">{lobby.shortCode}</span>
              <span className="mx-2">|</span>
              Status: <span className="text-lol-blue capitalize" data-testid="lobby-status-display">{lobby.status.replace(/_/g, ' ')}</span>
            </p>
          </div>
          <Link to="/" className="text-gray-400 hover:text-white" data-testid="lobby-link-leave">&larr; Leave</Link>
        </div>

        {/* Error Display */}
        {error && <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">{error}</div>}

        {/* Pending Action Banner */}
        {pendingAction && pendingAction.status === 'pending' && (
          <PendingActionBanner
            action={pendingAction}
            players={lobby.players}
            currentUserId={user?.id}
            currentUserSide={currentUserSide}
            isCaptain={isCaptain}
            onApprove={handleApprovePendingAction}
            onCancel={handleCancelPendingAction}
            approving={approvingAction}
            cancelling={cancellingAction}
          />
        )}

        {/* Voting Banner */}
        {lobby.votingEnabled && votingStatus && lobby.status === 'matchmaking' && (
          <VotingBanner
            votingStatus={votingStatus}
            isCaptain={isCaptain}
            canForceOption={lobby.votingMode === 'captain_override'}
            winningOptionNum={votingStatus.winningOption}
            onEndVoting={handleEndVoting}
            endingVoting={endingVoting}
          />
        )}

        {/* Main Content - Two Column Layout */}
        {(lobby.status === 'waiting_for_players' || lobby.status === 'matchmaking' || lobby.status === 'team_selected') && (
          <div className="grid grid-cols-1 lg:grid-cols-[1fr_auto_1fr] gap-6 mb-6">
            {/* Blue Team */}
            <TeamColumn
              side="blue"
              players={lobby.players}
              currentUserId={user?.id}
              swapMode={swapMode}
              selectedPlayerId={selectedForSwap}
              pendingAction={pendingAction}
              onPlayerClick={swapMode ? handlePlayerClick : undefined}
            />

            {/* Center Panel - Stats */}
            <div className="w-64">
              {teamStats ? (
                <TeamStatsPanel stats={teamStats} loading={fetchingTeamStats} />
              ) : (
                <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700 text-center text-gray-500">
                  Team stats available after matchmaking
                </div>
              )}
            </div>

            {/* Red Team */}
            <TeamColumn
              side="red"
              players={lobby.players}
              currentUserId={user?.id}
              swapMode={swapMode}
              selectedPlayerId={selectedForSwap}
              pendingAction={pendingAction}
              onPlayerClick={swapMode ? handlePlayerClick : undefined}
            />
          </div>
        )}

        {/* Swap Mode Banner */}
        {swapMode && (
          <div className="mb-6 bg-lol-gold/20 border border-lol-gold rounded-lg p-4 flex items-center justify-between">
            <div>
              <p className="text-lol-gold font-semibold">Swap Mode</p>
              <p className="text-gray-300 text-sm">
                {!selectedForSwap
                  ? 'Click on any player to select them'
                  : 'Now click another player to swap (same team = swap roles, different team = swap teams)'
                }
              </p>
            </div>
            <button
              onClick={() => {
                setSwapMode(false)
                setSelectedForSwap(null)
              }}
              className="text-gray-400 hover:text-white px-4 py-2 rounded-lg bg-gray-700 hover:bg-gray-600 transition"
            >
              Cancel (Esc)
            </button>
          </div>
        )}

        {/* Captain Controls */}
        {currentPlayer && currentUserSide && currentUserSide !== 'spectator' && (
          <CaptainControls
            players={lobby.players}
            currentUserId={user?.id || ''}
            currentUserSide={currentUserSide}
            isCaptain={isCaptain}
            hasTeams={hasTeams}
            isMatchmaking={isMatchmaking}
            hasPendingAction={!!pendingAction && pendingAction.status === 'pending'}
            swapMode={swapMode}
            onTakeCaptain={handleTakeCaptain}
            onPromoteCaptain={handlePromoteCaptain}
            onKickPlayer={handleKickPlayer}
            onToggleSwapMode={handleToggleSwapMode}
            onProposeMatchmake={handleProposeMatchmake}
            onProposeStartDraft={handleProposeStartDraft}
            onSetReady={handleReady}
            isReady={isReady}
            takingCaptain={takingCaptain}
            promotingCaptain={promotingCaptain}
            kickingPlayer={kickingPlayer}
            proposingAction={proposingAction}
          />
        )}

        {/* Match Options Selection */}
        {(lobby.status === 'matchmaking' || lobby.status === 'team_selected') && matchOptions && (
          <div className="space-y-6 mt-8">
            <h2 className="text-xl font-semibold text-white">
              {lobby.status === 'matchmaking'
                ? lobby.votingEnabled
                  ? 'Vote for Team Composition'
                  : 'Select Team Composition'
                : 'Selected Team Composition'}
            </h2>
            <p className="text-gray-400 text-sm">
              {lobby.status === 'matchmaking'
                ? lobby.votingEnabled
                  ? 'Click on an option to cast your vote.'
                  : isCaptain
                    ? 'Click on an option to propose it. The other captain must approve.'
                    : 'Waiting for a captain to propose a team composition...'
                : ''}
            </p>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {matchOptions.map(opt => (
                <MatchOptionCard
                  key={opt.optionNumber}
                  option={opt}
                  isSelected={lobby.selectedMatchOption === opt.optionNumber}
                  onSelect={
                    lobby.status === 'matchmaking' && !castingVote
                      ? lobby.votingEnabled
                        ? () => handleCastVote(opt.optionNumber)
                        : isCaptain && !proposingAction
                          ? () => handleProposeSelectOption(opt.optionNumber)
                          : undefined
                      : undefined
                  }
                  disabled={lobby.status === 'team_selected' || castingVote || (!lobby.votingEnabled && (!isCaptain || proposingAction))}
                  voteCount={votingStatus?.voteCounts?.[opt.optionNumber] || 0}
                  totalVotes={votingStatus?.votesCast || 0}
                  isVotingEnabled={lobby.votingEnabled && lobby.status === 'matchmaking'}
                  userVote={votingStatus?.userVote}
                  voters={votingStatus?.voters?.[opt.optionNumber]}
                />
              ))}
            </div>
            {/* Start Draft for Captain */}
            {isCaptain && lobby.status === 'team_selected' && (
              <div className="text-center">
                <button
                  onClick={handleStartDraft}
                  disabled={startingDraft}
                  className="bg-green-600 text-white font-semibold py-3 px-8 rounded-lg hover:bg-green-500 transition disabled:opacity-50"
                  data-testid="lobby-button-start-draft"
                >
                  {startingDraft ? 'Starting Draft...' : 'Start Draft'}
                </button>
              </div>
            )}
          </div>
        )}

        {/* Player Count */}
        <div className="text-center text-gray-400 mt-6" data-testid="lobby-player-count">
          {lobby.players.length}/10 players
          {lobby.players.length > 0 && (
            <span className="ml-2">
              ({lobby.players.filter(p => p.isReady).length} ready)
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
