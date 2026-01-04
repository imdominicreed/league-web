import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { roomsApi } from '@/api/rooms'

export default function JoinDraft() {
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()
  const { code: urlCode } = useParams()

  useEffect(() => {
    if (urlCode) {
      setCode(urlCode.toUpperCase())
    }
  }, [urlCode])

  const handleJoin = async () => {
    if (!code.trim()) {
      setError('Please enter a room code')
      return
    }

    setLoading(true)
    setError(null)
    try {
      const room = await roomsApi.get(code.toUpperCase())
      navigate(`/draft/${room.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Room not found')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <div className="w-full max-w-md">
        <h1 className="text-3xl font-bold text-center text-lol-gold mb-8">Join Draft Room</h1>

        <div className="space-y-6">
          {error && (
            <div className="bg-red-500/20 border border-red-500 text-red-300 px-4 py-2 rounded">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="code" className="block text-sm font-medium text-gray-300 mb-2">
              Room Code
            </label>
            <input
              type="text"
              id="code"
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              placeholder="Enter room code..."
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white text-center text-2xl tracking-widest uppercase focus:outline-none focus:border-lol-blue"
              maxLength={6}
            />
          </div>

          <button
            onClick={handleJoin}
            disabled={loading}
            className="w-full bg-lol-blue text-black font-semibold py-3 px-6 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
          >
            {loading ? 'Joining...' : 'Join Room'}
          </button>
        </div>
      </div>
    </div>
  )
}
