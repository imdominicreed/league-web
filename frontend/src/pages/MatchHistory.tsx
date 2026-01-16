import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchChampions } from '@/store/slices/championsSlice'
import { matchHistoryApi } from '@/api/matchHistory'
import { MatchHistoryItem } from '@/types'
import MatchHistoryCard from '@/components/match-history/MatchHistoryCard'

export default function MatchHistory() {
  const dispatch = useDispatch<AppDispatch>()
  const { championsList } = useSelector((state: RootState) => state.champions)

  const [matches, setMatches] = useState<MatchHistoryItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [hasMore, setHasMore] = useState(true)

  const LIMIT = 20

  useEffect(() => {
    // Load champions if not loaded
    if (championsList.length === 0) {
      dispatch(fetchChampions())
    }
  }, [dispatch, championsList.length])

  useEffect(() => {
    loadMatches()
  }, [])

  const loadMatches = async (offset = 0) => {
    try {
      setLoading(true)
      setError(null)
      const data = await matchHistoryApi.list({ limit: LIMIT, offset })
      if (offset === 0) {
        setMatches(data)
      } else {
        setMatches(prev => [...prev, ...data])
      }
      setHasMore(data.length === LIMIT)
    } catch (err) {
      setError('Failed to load match history')
      console.error('Error loading match history:', err)
    } finally {
      setLoading(false)
    }
  }

  const loadMore = () => {
    loadMatches(matches.length)
  }

  return (
    <div className="min-h-screen bg-lol-dark p-6">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-3xl font-bold text-lol-gold">Match History</h1>
            <p className="text-gray-400 mt-1">Your completed drafts</p>
          </div>
          <Link
            to="/"
            className="text-gray-400 hover:text-white transition"
          >
            Back to Home
          </Link>
        </div>

        {/* Content */}
        {loading && matches.length === 0 ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-gray-400">Loading...</div>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-red-400 mb-4">{error}</p>
            <button
              onClick={() => loadMatches()}
              className="bg-lol-blue text-black px-4 py-2 rounded hover:bg-opacity-80 transition"
            >
              Retry
            </button>
          </div>
        ) : matches.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-400 mb-4">No completed matches yet</p>
            <Link
              to="/create"
              className="inline-block bg-lol-blue text-black px-6 py-2 rounded font-semibold hover:bg-opacity-80 transition"
            >
              Start a Draft
            </Link>
          </div>
        ) : (
          <>
            <div className="space-y-3">
              {matches.map(match => (
                <MatchHistoryCard key={match.id} match={match} />
              ))}
            </div>

            {/* Load More */}
            {hasMore && (
              <div className="mt-6 text-center">
                <button
                  onClick={loadMore}
                  disabled={loading}
                  className="bg-gray-700 text-white px-6 py-2 rounded hover:bg-gray-600 transition disabled:opacity-50"
                >
                  {loading ? 'Loading...' : 'Load More'}
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
