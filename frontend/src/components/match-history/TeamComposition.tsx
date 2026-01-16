import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { MatchPlayer, ROLE_ABBREVIATIONS, Role } from '@/types'

interface Props {
  side: 'blue' | 'red'
  picks: string[]
  bans: string[]
  players?: MatchPlayer[]
  isTeamDraft: boolean
}

// Role order for display
const ROLE_ORDER: Role[] = ['top', 'jungle', 'mid', 'adc', 'support']

export default function TeamComposition({ side, picks, bans, players, isTeamDraft }: Props) {
  const champions = useSelector((state: RootState) => state.champions.champions)

  const getChampion = (championId: string) => {
    return champions[championId]
  }

  const getSortedPlayers = () => {
    if (!players) return []
    return [...players].sort((a, b) => {
      const aIdx = ROLE_ORDER.indexOf(a.assignedRole)
      const bIdx = ROLE_ORDER.indexOf(b.assignedRole)
      return aIdx - bIdx
    })
  }

  const sideColor = side === 'blue' ? 'lol-blue' : 'red-500'
  const sideBgColor = side === 'blue' ? 'blue-900' : 'red-900'

  return (
    <div className={`bg-${sideBgColor}/20 rounded-lg p-4 border border-${sideColor}/30`}>
      {/* Header */}
      <h3 className={`text-lg font-bold text-${sideColor} mb-4 capitalize`}>
        {side} Side
      </h3>

      {/* Picks */}
      <div className="mb-4">
        <h4 className="text-xs text-gray-400 uppercase mb-2">Picks</h4>
        <div className="space-y-2">
          {isTeamDraft && players ? (
            // Team draft: show player + champion per role
            getSortedPlayers().map((player, idx) => {
              const champion = picks[idx] ? getChampion(picks[idx]) : null
              const isNone = picks[idx] === 'None'

              return (
                <div key={player.userId} className="flex items-center gap-3">
                  {/* Champion Image */}
                  <div className="flex-shrink-0">
                    {champion && !isNone ? (
                      <img
                        src={champion.imageUrl}
                        alt={champion.name}
                        className="w-12 h-12 rounded"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded bg-gray-700 flex items-center justify-center text-gray-500">
                        ?
                      </div>
                    )}
                  </div>

                  {/* Player Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400 font-mono">
                        {ROLE_ABBREVIATIONS[player.assignedRole]}
                      </span>
                      <span className="text-sm text-white truncate">
                        {player.displayName}
                        {player.isCaptain && (
                          <span className="ml-1 text-xs text-yellow-500">C</span>
                        )}
                      </span>
                    </div>
                    <div className="text-xs text-gray-400">
                      {champion && !isNone ? champion.name : 'No pick'}
                    </div>
                  </div>
                </div>
              )
            })
          ) : (
            // 1v1 draft: just show champions
            picks.map((pickId, idx) => {
              const champion = getChampion(pickId)
              const isNone = pickId === 'None'

              return (
                <div key={idx} className="flex items-center gap-3">
                  <div className="flex-shrink-0">
                    {champion && !isNone ? (
                      <img
                        src={champion.imageUrl}
                        alt={champion.name}
                        className="w-12 h-12 rounded"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded bg-gray-700 flex items-center justify-center text-gray-500">
                        ?
                      </div>
                    )}
                  </div>
                  <div className="text-sm text-white">
                    {champion && !isNone ? champion.name : 'No pick'}
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Bans */}
      <div>
        <h4 className="text-xs text-gray-400 uppercase mb-2">Bans</h4>
        <div className="flex flex-wrap gap-2">
          {bans.map((banId, idx) => {
            const champion = getChampion(banId)
            const isNone = banId === 'None'

            return (
              <div key={idx} className="relative">
                {champion && !isNone ? (
                  <div className="relative">
                    <img
                      src={champion.imageUrl}
                      alt={champion.name}
                      className="w-10 h-10 rounded grayscale opacity-60"
                    />
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="w-full h-0.5 bg-red-500 rotate-45" />
                    </div>
                  </div>
                ) : (
                  <div className="w-10 h-10 rounded bg-gray-700 flex items-center justify-center text-xs text-gray-500">
                    {isNone ? '-' : '?'}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
