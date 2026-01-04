import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { Player, Champion } from '@/types'

interface Props {
  side: 'blue' | 'red'
  player: Player | null
  picks: string[]
  isActive: boolean
  hoveredChampion: string | null
}

// Position labels for picks
const POSITIONS = ['TOP', 'JGL', 'MID', 'BOT', 'SUP']

// Get splash art URL for a champion
function getSplashUrl(champion: Champion): string {
  return `https://ddragon.leagueoflegends.com/cdn/img/champion/loading/${champion.id}_0.jpg`
}

export default function TeamPanel({ side, player, picks, isActive, hoveredChampion }: Props) {
  const { champions } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)

  const teamColor = side === 'blue' ? 'blue-team' : 'red-team'
  const borderColor = side === 'blue' ? 'border-blue-team' : 'border-red-team'

  // For current pick, show hovered champion
  const currentActionChampion = isActive && hoveredChampion ? hoveredChampion : null

  return (
    <div className={`w-36 bg-lol-dark-blue flex flex-col border-l border-r border-lol-border ${
      isActive ? borderColor : ''
    }`}>
      {/* Team Header */}
      <div className={`px-3 py-2 border-b border-lol-border bg-${teamColor}/10`}>
        <div className={`font-beaufort text-${teamColor} text-sm uppercase tracking-wider`}>
          {side === 'blue' ? 'Blue' : 'Red'} Side
        </div>
        <div className="text-lol-gold-light text-xs truncate">
          {player?.displayName || 'Waiting...'}
        </div>
      </div>

      {/* Picks - Large Splash Art Slots */}
      <div className="flex-1 flex flex-col">
        {[0, 1, 2, 3, 4].map((i) => {
          const championId = picks[i]
          const champion = championId ? champions[championId] : null
          const isCurrentPick = draft.actionType === 'pick' && isActive && i === picks.length
          const showHovered = isCurrentPick && currentActionChampion && champions[currentActionChampion]

          return (
            <div
              key={i}
              className={`flex-1 relative overflow-hidden border-b border-lol-border last:border-b-0 ${
                isCurrentPick ? 'ring-2 ring-inset ring-lol-gold animate-pulse' : ''
              }`}
            >
              {champion ? (
                // Picked champion - show splash art
                <>
                  <img
                    src={getSplashUrl(champion)}
                    alt={champion.name}
                    className="absolute inset-0 w-full h-full object-cover object-top"
                  />
                  {/* Gradient overlay for text readability */}
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent" />
                  {/* Champion name */}
                  <div className="absolute bottom-0 left-0 right-0 p-2">
                    <div className="text-white text-xs font-semibold truncate">
                      {champion.name}
                    </div>
                  </div>
                </>
              ) : showHovered ? (
                // Currently hovering - show preview
                <>
                  <img
                    src={getSplashUrl(champions[currentActionChampion])}
                    alt="Selecting..."
                    className="absolute inset-0 w-full h-full object-cover object-top opacity-60"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent" />
                  <div className="absolute bottom-0 left-0 right-0 p-2">
                    <div className="text-lol-gold text-xs font-semibold truncate">
                      {champions[currentActionChampion].name}
                    </div>
                  </div>
                </>
              ) : (
                // Empty slot
                <div className="absolute inset-0 flex items-center justify-center bg-lol-gray/30">
                  <span className={`font-beaufort text-sm ${
                    isCurrentPick ? 'text-lol-gold' : 'text-gray-600'
                  }`}>
                    {POSITIONS[i]}
                  </span>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
