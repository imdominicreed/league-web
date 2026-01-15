import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { createLobby } from '@/store/slices/lobbySlice'
import { VotingMode, VOTING_MODE_LABELS, VOTING_MODE_DESCRIPTIONS } from '@/types'

export default function CreateLobby() {
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()
  const { loading, error } = useSelector((state: RootState) => state.lobby)

  const [draftMode, setDraftMode] = useState<'pro_play' | 'fearless'>('pro_play')
  const [timerDuration, setTimerDuration] = useState(30)
  const [votingEnabled, setVotingEnabled] = useState(false)
  const [votingMode, setVotingMode] = useState<VotingMode>('majority')

  const handleCreate = async () => {
    const result = await dispatch(createLobby({
      draftMode,
      timerDurationSeconds: timerDuration,
      votingEnabled,
      votingMode: votingEnabled ? votingMode : undefined,
    }))
    if (createLobby.fulfilled.match(result)) {
      navigate(`/lobby/${result.payload.id}`)
    }
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-8">
      <h1 className="text-4xl font-bold text-lol-gold mb-8">Create 10-Man Lobby</h1>

      <div className="bg-gray-800 rounded-lg p-8 w-full max-w-md">
        {error && (
          <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-2 rounded mb-4">
            {error}
          </div>
        )}

        <div className="mb-6">
          <label className="block text-gray-300 mb-2">Draft Mode</label>
          <select
            value={draftMode}
            onChange={(e) => setDraftMode(e.target.value as 'pro_play' | 'fearless')}
            className="w-full bg-gray-700 border border-gray-600 rounded px-4 py-2 text-white"
          >
            <option value="pro_play">Pro Play</option>
            <option value="fearless">Fearless</option>
          </select>
        </div>

        <div className="mb-6">
          <label className="block text-gray-300 mb-2">Timer Duration (seconds)</label>
          <input
            type="number"
            value={timerDuration}
            onChange={(e) => setTimerDuration(Number(e.target.value))}
            min={10}
            max={120}
            className="w-full bg-gray-700 border border-gray-600 rounded px-4 py-2 text-white"
          />
        </div>

        {/* Voting Section */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-4">
            <input
              type="checkbox"
              id="votingEnabled"
              checked={votingEnabled}
              onChange={(e) => setVotingEnabled(e.target.checked)}
              className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-lol-blue focus:ring-lol-blue"
            />
            <label htmlFor="votingEnabled" className="text-gray-300">
              Enable Team Voting
            </label>
          </div>

          {votingEnabled && (
            <div className="ml-7">
              <label className="block text-gray-300 mb-2 text-sm">Voting Mode</label>
              <select
                value={votingMode}
                onChange={(e) => setVotingMode(e.target.value as VotingMode)}
                className="w-full bg-gray-700 border border-gray-600 rounded px-4 py-2 text-white text-sm"
              >
                {(['majority', 'unanimous', 'captain_override'] as VotingMode[]).map(mode => (
                  <option key={mode} value={mode}>
                    {VOTING_MODE_LABELS[mode]}
                  </option>
                ))}
              </select>
              <p className="text-gray-500 text-xs mt-2">
                {VOTING_MODE_DESCRIPTIONS[votingMode]}
              </p>
            </div>
          )}
        </div>

        <button
          onClick={handleCreate}
          disabled={loading}
          className="w-full bg-lol-blue text-black font-semibold py-3 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
        >
          {loading ? 'Creating...' : 'Create Lobby'}
        </button>

        <Link to="/" className="block text-center text-gray-400 hover:text-white mt-4">
          &larr; Back to Home
        </Link>
      </div>
    </div>
  )
}
