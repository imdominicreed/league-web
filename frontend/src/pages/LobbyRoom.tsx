import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import {
  fetchLobby,
  fetchMatchOptions,
  setReady,
  generateTeams,
  selectMatchOption,
  startDraft,
  takeCaptain,
  promoteCaptain,
  kickPlayer,
  proposeSwap,
  proposeMatchmake,
  proposeStartDraft,
  fetchPendingAction,
  approvePendingAction,
  cancelPendingAction,
  fetchTeamStats,
} from '@/store/slices/lobbySlice'
import { TeamColumn } from '@/components/lobby/TeamColumn'
import { PendingActionBanner } from '@/components/lobby/PendingActionBanner'
import { TeamStatsPanel } from '@/components/lobby/TeamStatsPanel'
import { CaptainControls } from '@/components/lobby/CaptainControls'
import { MatchOptionCard } from '@/components/lobby/MatchOptionCard'

export default function LobbyRoom() {
  const { lobbyId } = useParams<{ lobbyId: string }>()
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()

  const {
    lobby,
    matchOptions,
    pendingAction,
    teamStats,
    loading,
    error,
    generatingTeams,
    selectingOption,
    startingDraft,
    createdRoom,
    takingCaptain,
    promotingCaptain,
    kickingPlayer,
    proposingAction,
    approvingAction,
    cancellingAction,
    fetchingTeamStats,
  } = useSelector((state: RootState) => state.lobby)
  const { user } = useSelector((state: RootState) => state.auth)

  const [selectedOption, setSelectedOption] = useState<number | null>(null)
  const [pollInterval, setPollInterval] = useState<ReturnType<typeof setInterval> | null>(null)

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
    if (lobby) dispatch(proposeSwap({ lobbyId: lobby.id, player1Id, player2Id, swapType }))
  }, [lobby, dispatch])

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

  const handleGenerateTeams = useCallback(() => {
    if (lobby) dispatch(generateTeams(lobby.id))
  }, [lobby, dispatch])

  const handleSelectOption = useCallback(() => {
    if (lobby && selectedOption) {
      dispatch(selectMatchOption({ lobbyId: lobby.id, optionNumber: selectedOption }))
    }
  }, [lobby, selectedOption, dispatch])

  const handleStartDraft = useCallback(() => {
    if (lobby) dispatch(startDraft(lobby.id))
  }, [lobby, dispatch])

  const isCreator = lobby?.createdBy === user?.id
  const allReady = lobby?.players.length === 10 && lobby.players.every(p => p.isReady)

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
              Code: <span className="text-white font-mono">{lobby.shortCode}</span>
              <span className="mx-2">|</span>
              Status: <span className="text-lol-blue capitalize">{lobby.status.replace(/_/g, ' ')}</span>
            </p>
          </div>
          <Link to="/" className="text-gray-400 hover:text-white">&larr; Leave</Link>
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

        {/* Main Content - Two Column Layout */}
        {(lobby.status === 'waiting_for_players' || lobby.status === 'team_selected') && (
          <div className="grid grid-cols-1 lg:grid-cols-[1fr_auto_1fr] gap-6 mb-6">
            {/* Blue Team */}
            <TeamColumn
              side="blue"
              players={lobby.players}
              currentUserId={user?.id}
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
            />
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
            hasPendingAction={!!pendingAction && pendingAction.status === 'pending'}
            onTakeCaptain={handleTakeCaptain}
            onPromoteCaptain={handlePromoteCaptain}
            onKickPlayer={handleKickPlayer}
            onProposeSwap={handleProposeSwap}
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

        {/* Legacy Generate Teams for Creator (fallback) */}
        {lobby.status === 'waiting_for_players' && isCreator && allReady && (
          <div className="mt-6 text-center">
            <button
              onClick={handleGenerateTeams}
              disabled={generatingTeams}
              className="bg-lol-gold text-black font-semibold py-3 px-8 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
            >
              {generatingTeams ? 'Generating Teams...' : 'Generate Team Options (Creator Only)'}
            </button>
          </div>
        )}

        {/* Match Options Selection */}
        {(lobby.status === 'matchmaking' || lobby.status === 'team_selected') && matchOptions && (
          <div className="space-y-6 mt-8">
            <h2 className="text-xl font-semibold text-white">Select Team Composition</h2>
            <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
              {matchOptions.map(opt => (
                <MatchOptionCard
                  key={opt.optionNumber}
                  option={opt}
                  isSelected={selectedOption === opt.optionNumber || lobby.selectedMatchOption === opt.optionNumber}
                  onSelect={isCreator && lobby.status === 'matchmaking' ? () => setSelectedOption(opt.optionNumber) : undefined}
                  disabled={!isCreator || lobby.status === 'team_selected'}
                />
              ))}
            </div>
            {isCreator && lobby.status === 'matchmaking' && selectedOption && (
              <div className="text-center">
                <button
                  onClick={handleSelectOption}
                  disabled={selectingOption}
                  className="bg-lol-gold text-black font-semibold py-3 px-8 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
                >
                  {selectingOption ? 'Confirming...' : 'Confirm Selection'}
                </button>
              </div>
            )}
            {isCreator && lobby.status === 'team_selected' && (
              <div className="text-center">
                <button
                  onClick={handleStartDraft}
                  disabled={startingDraft}
                  className="bg-green-600 text-white font-semibold py-3 px-8 rounded-lg hover:bg-green-500 transition disabled:opacity-50"
                >
                  {startingDraft ? 'Starting Draft...' : 'Start Draft (Creator Only)'}
                </button>
              </div>
            )}
          </div>
        )}

        {/* Player Count */}
        <div className="text-center text-gray-400 mt-6">
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
