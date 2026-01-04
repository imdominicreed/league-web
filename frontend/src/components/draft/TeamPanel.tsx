import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { Player } from '@/types'
import ChampionCard from './ChampionCard'

interface Props {
  side: 'blue' | 'red'
  player: Player | null
  bans: string[]
  picks: string[]
  isActive: boolean
  hoveredChampion: string | null
}

export default function TeamPanel({ side, player, bans, picks, isActive, hoveredChampion }: Props) {
  const { champions } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)

  const sideColor = side === 'blue' ? 'blue-side' : 'red-side'
  const borderClass = isActive ? `border-${sideColor}` : 'border-gray-800'

  // For current pick/ban, show hovered champion
  const currentActionChampion = isActive && hoveredChampion ? hoveredChampion : null

  return (
    <div className={`w-64 bg-gray-900 border-r ${side === 'red' ? 'border-l' : ''} ${borderClass} flex flex-col`}>
      {/* Player Name */}
      <div className={`p-4 border-b border-gray-800 bg-${sideColor}/10`}>
        <div className={`text-${sideColor} font-semibold text-lg`}>
          {side === 'blue' ? 'Blue Side' : 'Red Side'}
        </div>
        <div className="text-gray-400 text-sm">
          {player?.displayName || 'Waiting...'}
        </div>
      </div>

      {/* Bans */}
      <div className="p-4 border-b border-gray-800">
        <div className="text-xs text-gray-500 uppercase tracking-wider mb-2">Bans</div>
        <div className="flex gap-1">
          {[0, 1, 2, 3, 4].map((i) => {
            const championId = bans[i]
            const champion = championId ? champions[championId] : null
            const isCurrentBan = draft.actionType === 'ban' && isActive && i === bans.length

            return (
              <div
                key={i}
                className={`w-10 h-10 rounded bg-gray-800 overflow-hidden ${
                  isCurrentBan ? 'ring-2 ring-yellow-500 animate-pulse' : ''
                }`}
              >
                {champion ? (
                  <img
                    src={champion.imageUrl}
                    alt={champion.name}
                    className="w-full h-full object-cover opacity-50 grayscale"
                  />
                ) : isCurrentBan && currentActionChampion && champions[currentActionChampion] ? (
                  <img
                    src={champions[currentActionChampion].imageUrl}
                    alt="Selecting..."
                    className="w-full h-full object-cover opacity-50"
                  />
                ) : null}
              </div>
            )
          })}
        </div>
      </div>

      {/* Picks */}
      <div className="flex-1 p-4">
        <div className="text-xs text-gray-500 uppercase tracking-wider mb-2">Picks</div>
        <div className="space-y-2">
          {[0, 1, 2, 3, 4].map((i) => {
            const championId = picks[i]
            const champion = championId ? champions[championId] : null
            const isCurrentPick = draft.actionType === 'pick' && isActive && i === picks.length

            return (
              <div
                key={i}
                className={`h-16 rounded bg-gray-800 overflow-hidden flex items-center ${
                  isCurrentPick ? 'ring-2 ring-yellow-500 animate-pulse' : ''
                }`}
              >
                {champion ? (
                  <ChampionCard champion={champion} size="small" />
                ) : isCurrentPick && currentActionChampion && champions[currentActionChampion] ? (
                  <ChampionCard champion={champions[currentActionChampion]} size="small" selecting />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-gray-600 text-sm">
                    Pick {i + 1}
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
