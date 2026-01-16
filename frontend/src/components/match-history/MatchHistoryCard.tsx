import { Link } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { MatchHistoryItem } from '@/types'

interface Props {
  match: MatchHistoryItem
}

export default function MatchHistoryCard({ match }: Props) {
  const champions = useSelector((state: RootState) => state.champions.champions)

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getChampionImage = (championId: string) => {
    const champion = champions[championId]
    return champion?.imageUrl || ''
  }

  const getSideColor = (side: string) => {
    if (side === 'blue') return 'border-lol-blue'
    if (side === 'red') return 'border-red-500'
    return 'border-gray-600'
  }

  const getSideBadge = (side: string) => {
    if (side === 'blue') return <span className="text-xs bg-lol-blue text-black px-2 py-0.5 rounded">Blue Side</span>
    if (side === 'red') return <span className="text-xs bg-red-500 text-white px-2 py-0.5 rounded">Red Side</span>
    return <span className="text-xs bg-gray-600 text-white px-2 py-0.5 rounded">Spectator</span>
  }

  return (
    <Link
      to={`/match/${match.id}`}
      className={`block bg-gray-800 rounded-lg p-4 border-l-4 ${getSideColor(match.yourSide)} hover:bg-gray-700 transition`}
    >
      <div className="flex justify-between items-start mb-3">
        <div>
          <div className="flex items-center gap-2 mb-1">
            {getSideBadge(match.yourSide)}
            {match.isTeamDraft && (
              <span className="text-xs bg-purple-600 text-white px-2 py-0.5 rounded">
                Team Draft
              </span>
            )}
            <span className="text-xs bg-gray-600 text-gray-300 px-2 py-0.5 rounded capitalize">
              {match.draftMode.replace('_', ' ')}
            </span>
          </div>
          <p className="text-sm text-gray-400">
            {formatDate(match.completedAt)}
          </p>
        </div>
        <p className="text-xs text-gray-500 font-mono">
          #{match.shortCode}
        </p>
      </div>

      {/* Team Compositions */}
      <div className="grid grid-cols-2 gap-4">
        {/* Blue Team */}
        <div>
          <p className="text-xs text-lol-blue mb-1 font-semibold">Blue Side</p>
          <div className="flex gap-1">
            {match.bluePicks.map((champId, idx) => (
              <div key={idx} className="relative">
                {getChampionImage(champId) ? (
                  <img
                    src={getChampionImage(champId)}
                    alt={champId}
                    className="w-10 h-10 rounded object-cover border border-lol-blue/50"
                  />
                ) : (
                  <div className="w-10 h-10 rounded bg-gray-700 border border-lol-blue/50 flex items-center justify-center text-xs text-gray-500">
                    ?
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Red Team */}
        <div>
          <p className="text-xs text-red-400 mb-1 font-semibold">Red Side</p>
          <div className="flex gap-1">
            {match.redPicks.map((champId, idx) => (
              <div key={idx} className="relative">
                {getChampionImage(champId) ? (
                  <img
                    src={getChampionImage(champId)}
                    alt={champId}
                    className="w-10 h-10 rounded object-cover border border-red-500/50"
                  />
                ) : (
                  <div className="w-10 h-10 rounded bg-gray-700 border border-red-500/50 flex items-center justify-center text-xs text-gray-500">
                    ?
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Team Players (for team drafts) */}
      {match.isTeamDraft && (match.blueTeam || match.redTeam) && (
        <div className="mt-3 pt-3 border-t border-gray-700 grid grid-cols-2 gap-4">
          <div className="text-xs text-gray-400">
            {match.blueTeam?.map(p => p.displayName).join(', ')}
          </div>
          <div className="text-xs text-gray-400">
            {match.redTeam?.map(p => p.displayName).join(', ')}
          </div>
        </div>
      )}
    </Link>
  )
}
