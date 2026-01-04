import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchLobby, setReady, generateTeams, selectMatchOption, startDraft } from '@/store/slices/lobbySlice'
import { LobbyPlayerGrid } from '@/components/lobby/LobbyPlayerGrid'
import { MatchOptionCard } from '@/components/lobby/MatchOptionCard'

export default function LobbyRoom() {
  const { lobbyId } = useParams<{ lobbyId: string }>()
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()

  const { lobby, matchOptions, loading, error, generatingTeams, selectingOption, startingDraft, createdRoom } = useSelector(
    (state: RootState) => state.lobby
  )
  const { user } = useSelector((state: RootState) => state.auth)

  const [selectedOption, setSelectedOption] = useState<number | null>(null)
  const [pollInterval, setPollInterval] = useState<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    if (lobbyId) {
      dispatch(fetchLobby(lobbyId))
      const interval = setInterval(() => dispatch(fetchLobby(lobbyId)), 3000)
      setPollInterval(interval)
      return () => clearInterval(interval)
    }
  }, [lobbyId, dispatch])

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

  const handleReady = (ready: boolean) => {
    if (lobbyId) dispatch(setReady({ idOrCode: lobbyId, ready }))
  }

  const handleGenerateTeams = () => {
    if (lobby) dispatch(generateTeams(lobby.id))
  }

  const handleSelectOption = () => {
    if (lobby && selectedOption) {
      dispatch(selectMatchOption({ lobbyId: lobby.id, optionNumber: selectedOption }))
    }
  }

  const handleStartDraft = () => {
    if (lobby) {
      dispatch(startDraft(lobby.id))
    }
  }

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
      <div className="max-w-6xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-lol-gold">10-Man Lobby</h1>
            <p className="text-gray-400">Code: <span className="text-white font-mono">{lobby.shortCode}</span></p>
          </div>
          <Link to="/" className="text-gray-400 hover:text-white">&larr; Leave</Link>
        </div>

        {error && <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">{error}</div>}

        {lobby.status === 'waiting_for_players' && (
          <>
            <LobbyPlayerGrid players={lobby.players} currentUserId={user?.id} onReady={handleReady} />
            {isCreator && allReady && (
              <div className="mt-6 text-center">
                <button
                  onClick={handleGenerateTeams}
                  disabled={generatingTeams}
                  className="bg-lol-gold text-black font-semibold py-3 px-8 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
                >
                  {generatingTeams ? 'Generating Teams...' : 'Generate Team Options'}
                </button>
              </div>
            )}
          </>
        )}

        {(lobby.status === 'matchmaking' || lobby.status === 'team_selected') && matchOptions && (
          <div className="space-y-6">
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
                  {startingDraft ? 'Starting Draft...' : 'Start Draft'}
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
