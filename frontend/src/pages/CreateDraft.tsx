import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { roomsApi } from '@/api/rooms'

export default function CreateDraft() {
  const [draftMode, setDraftMode] = useState<'pro_play' | 'fearless'>('pro_play')
  const [timerDuration, setTimerDuration] = useState(30)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  const handleCreate = async () => {
    setLoading(true)
    setError(null)
    try {
      const room = await roomsApi.create({ draftMode, timerDuration })
      navigate(`/draft/${room.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create room')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <div className="w-full max-w-md">
        <h1 className="text-3xl font-bold text-center text-lol-gold mb-8">Create Draft Room</h1>

        <div className="space-y-6">
          {error && (
            <div className="bg-red-500/20 border border-red-500 text-red-300 px-4 py-2 rounded">
              {error}
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Draft Mode
            </label>
            <div className="grid grid-cols-2 gap-4">
              <button
                type="button"
                onClick={() => setDraftMode('pro_play')}
                className={`py-3 px-4 rounded-lg border-2 transition ${
                  draftMode === 'pro_play'
                    ? 'border-lol-blue bg-lol-blue/20 text-lol-blue'
                    : 'border-gray-700 text-gray-400 hover:border-gray-500'
                }`}
              >
                Pro Play
              </button>
              <button
                type="button"
                onClick={() => setDraftMode('fearless')}
                className={`py-3 px-4 rounded-lg border-2 transition ${
                  draftMode === 'fearless'
                    ? 'border-lol-blue bg-lol-blue/20 text-lol-blue'
                    : 'border-gray-700 text-gray-400 hover:border-gray-500'
                }`}
              >
                Fearless
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Timer Duration: {timerDuration}s
            </label>
            <input
              type="range"
              min="15"
              max="60"
              step="5"
              value={timerDuration}
              onChange={(e) => setTimerDuration(Number(e.target.value))}
              className="w-full"
            />
            <div className="flex justify-between text-xs text-gray-500 mt-1">
              <span>15s</span>
              <span>60s</span>
            </div>
          </div>

          <button
            onClick={handleCreate}
            disabled={loading}
            className="w-full bg-lol-blue text-black font-semibold py-3 px-6 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
          >
            {loading ? 'Creating...' : 'Create Room'}
          </button>
        </div>
      </div>
    </div>
  )
}
