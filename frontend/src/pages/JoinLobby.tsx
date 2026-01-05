import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { lobbyApi } from '@/api/lobby'

export default function JoinLobby() {
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
      setError('Please enter a lobby code')
      return
    }

    setLoading(true)
    setError(null)
    try {
      const lobby = await lobbyApi.get(code.toUpperCase())
      await lobbyApi.join(lobby.id)
      navigate(`/lobby/${lobby.id}`)
    } catch (err) {
      if (err instanceof Error) {
        if (err.message.includes('not found')) {
          setError('Lobby not found')
        } else if (err.message.includes('full')) {
          setError('Lobby is full')
        } else if (err.message.includes('state')) {
          setError('Cannot join lobby - draft may have already started')
        } else {
          setError(err.message)
        }
      } else {
        setError('Failed to join lobby')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <div className="w-full max-w-md">
        <h1 className="text-3xl font-bold text-center text-lol-gold mb-8">Join 10-Man Lobby</h1>

        <div className="space-y-6">
          {error && (
            <div className="bg-red-500/20 border border-red-500 text-red-300 px-4 py-2 rounded">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="code" className="block text-sm font-medium text-gray-300 mb-2">
              Lobby Code
            </label>
            <input
              type="text"
              id="code"
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              placeholder="Enter lobby code..."
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white text-center text-2xl tracking-widest uppercase focus:outline-none focus:border-purple-500"
              maxLength={8}
            />
          </div>

          <button
            onClick={handleJoin}
            disabled={loading}
            className="w-full bg-purple-600 text-white font-semibold py-3 px-6 rounded-lg hover:bg-purple-700 transition disabled:opacity-50"
          >
            {loading ? 'Joining...' : 'Join Lobby'}
          </button>
        </div>
      </div>
    </div>
  )
}
