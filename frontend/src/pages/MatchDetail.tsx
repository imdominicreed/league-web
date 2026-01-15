import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchChampions } from '@/store/slices/championsSlice'
import { matchHistoryApi } from '@/api/matchHistory'
import { MatchDetail as MatchDetailType } from '@/types'

export default function MatchDetail() {
  const { roomId } = useParams<{ roomId: string }>()
  const dispatch = useDispatch<AppDispatch>()
  const { championsList } = useSelector((state: RootState) => state.champions)

  const [match, setMatch] = useState<MatchDetailType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (championsList.length === 0) {
      dispatch(fetchChampions())
    }
  }, [dispatch, championsList.length])

  useEffect(() => {
    if (roomId) {
      loadMatch()
    }
  }, [roomId])

  const loadMatch = async () => {
    if (!roomId) return
    try {
      setLoading(true)
      setError(null)
      const data = await matchHistoryApi.getDetail(roomId)
      setMatch(data)
    } catch (err) {
      setError('Failed to load match details')
      console.error('Error loading match details:', err)
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-lol-dark flex items-center justify-center">
        <div className="text-gray-400">Loading...</div>
      </div>
    )
  }

  if (error || !match) {
    return (
      <div className="min-h-screen bg-lol-dark flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-400 mb-4">{error || 'Match not found'}</p>
          <Link
            to="/match-history"
            className="text-lol-blue hover:underline"
          >
            Back to Match History
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-lol-dark p-6">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-lol-gold">Match Detail</h1>
            <p className="text-gray-400 mt-1">#{match.shortCode}</p>
          </div>
          <Link
            to="/match-history"
            className="text-gray-400 hover:text-white transition"
          >
            Back to History
          </Link>
        </div>

        {/* Match detail content will be implemented in Phase 5 */}
        <div className="bg-gray-800 rounded-lg p-6">
          <p className="text-gray-400">Match detail view coming soon...</p>
          <pre className="mt-4 text-xs text-gray-500 overflow-auto">
            {JSON.stringify(match, null, 2)}
          </pre>
        </div>
      </div>
    </div>
  )
}
