import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { createLobby } from '@/store/slices/lobbySlice'

export default function CreateLobby() {
  const navigate = useNavigate()
  const dispatch = useDispatch<AppDispatch>()
  const { loading, error } = useSelector((state: RootState) => state.lobby)

  const [draftMode, setDraftMode] = useState<'pro_play' | 'fearless'>('pro_play')
  const [timerDuration, setTimerDuration] = useState(30)

  const handleCreate = async () => {
    const result = await dispatch(createLobby({ draftMode, timerDurationSeconds: timerDuration }))
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
