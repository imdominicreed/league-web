import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchChampions } from '@/store/slices/championsSlice'
import { matchHistoryApi } from '@/api/matchHistory'
import { MatchDetail as MatchDetailType } from '@/types'
import DraftTimeline from '@/components/match-history/DraftTimeline'
import TeamComposition from '@/components/match-history/TeamComposition'

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

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getSideBadge = (side: string) => {
    if (side === 'blue') return <span className="text-xs bg-lol-blue text-black px-2 py-0.5 rounded">Blue Side</span>
    if (side === 'red') return <span className="text-xs bg-red-500 text-white px-2 py-0.5 rounded">Red Side</span>
    return <span className="text-xs bg-gray-600 text-white px-2 py-0.5 rounded">Spectator</span>
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
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-bold text-lol-gold">Match Detail</h1>
              {getSideBadge(match.yourSide)}
              {match.isTeamDraft && (
                <span className="text-xs bg-purple-600 text-white px-2 py-0.5 rounded">
                  Team Draft
                </span>
              )}
            </div>
            <p className="text-gray-400">
              <span className="font-mono text-sm">#{match.shortCode}</span>
              <span className="mx-2">•</span>
              <span className="capitalize">{match.draftMode.replace('_', ' ')}</span>
              <span className="mx-2">•</span>
              {match.timerDurationSeconds}s timer
            </p>
          </div>
          <Link
            to="/match-history"
            className="text-gray-400 hover:text-white transition"
          >
            Back to History
          </Link>
        </div>

        {/* Timestamps */}
        {match.completedAt && (
          <div className="mb-6 text-sm text-gray-400">
            Completed {formatDate(match.completedAt)}
          </div>
        )}

        {/* Team Compositions */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
          <TeamComposition
            side="blue"
            picks={match.bluePicks}
            bans={match.blueBans}
            players={match.blueTeam}
            isTeamDraft={match.isTeamDraft}
          />
          <TeamComposition
            side="red"
            picks={match.redPicks}
            bans={match.redBans}
            players={match.redTeam}
            isTeamDraft={match.isTeamDraft}
          />
        </div>

        {/* Draft Timeline */}
        <DraftTimeline actions={match.actions} />
      </div>
    </div>
  )
}
