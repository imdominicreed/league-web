import { MatchOption, MatchAssignment, ROLE_DISPLAY_NAMES, ALGORITHM_LABELS, AlgorithmType } from '@/types'

interface MatchOptionCardProps {
  option: MatchOption
  isSelected: boolean
  onSelect?: () => void
  disabled?: boolean
}

function getAlgorithmBadgeColor(type: AlgorithmType): string {
  switch (type) {
    case 'mmr_balanced':
      return 'bg-blue-600/20 text-blue-400'
    case 'role_comfort':
      return 'bg-purple-600/20 text-purple-400'
    case 'lane_balanced':
      return 'bg-green-600/20 text-green-400'
    default:
      return 'bg-gray-600/20 text-gray-400'
  }
}

export function MatchOptionCard({ option, isSelected, onSelect, disabled }: MatchOptionCardProps) {
  const blueTeam = option.assignments.filter(a => a.team === 'blue')
  const redTeam = option.assignments.filter(a => a.team === 'red')

  // Sort by role order
  const roleOrder = ['top', 'jungle', 'mid', 'adc', 'support']
  const sortByRole = (a: MatchAssignment, b: MatchAssignment) =>
    roleOrder.indexOf(a.assignedRole) - roleOrder.indexOf(b.assignedRole)

  blueTeam.sort(sortByRole)
  redTeam.sort(sortByRole)

  // Algorithm-specific highlight
  const getAlgorithmHighlight = () => {
    if (!option.algorithmType) return null
    switch (option.algorithmType) {
      case 'mmr_balanced':
        return `MMR Diff: ${option.mmrDifference}`
      case 'role_comfort': {
        const avgComfort = ((option.avgBlueComfort ?? 0) + (option.avgRedComfort ?? 0)) / 2
        return `Avg Comfort: ${avgComfort.toFixed(1)}/5`
      }
      case 'lane_balanced':
        return `Max Lane Gap: ${option.maxLaneDiff ?? 0} MMR`
      default:
        return null
    }
  }

  const renderTeam = (team: MatchAssignment[], side: 'blue' | 'red') => {
    const avgComfort = side === 'blue' ? (option.avgBlueComfort ?? 0) : (option.avgRedComfort ?? 0)
    return (
      <div className="space-y-2">
        <h4
          className={`font-semibold text-sm ${
            side === 'blue' ? 'text-blue-400' : 'text-red-400'
          }`}
        >
          {side === 'blue' ? 'Blue Team' : 'Red Team'}
        </h4>
        {team.map(player => (
          <div
            key={player.userId}
            className="flex items-center justify-between text-sm bg-gray-700/50 rounded px-2 py-1"
          >
            <div className="flex items-center gap-2">
              <span className="text-gray-400 w-16 text-xs">
                {ROLE_DISPLAY_NAMES[player.assignedRole]}
              </span>
              <span className="text-white truncate max-w-[100px]">
                {player.displayName || 'Unknown'}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-gray-500 text-xs">{player.roleMmr}</span>
              <span className="text-yellow-400 text-xs">
                {'★'.repeat(player.comfortRating)}
                {'☆'.repeat(5 - player.comfortRating)}
              </span>
            </div>
          </div>
        ))}
        <div className="text-xs text-gray-400 mt-1 flex gap-3">
          <span>Avg MMR: {side === 'blue' ? option.blueTeamAvgMmr : option.redTeamAvgMmr}</span>
          <span>Comfort: {avgComfort.toFixed(1)}/5</span>
        </div>
      </div>
    )
  }

  return (
    <div
      data-testid={`match-option-${option.optionNumber}`}
      className={`bg-gray-800 rounded-lg p-4 border-2 transition-all ${
        isSelected
          ? 'border-lol-gold shadow-lg shadow-lol-gold/20'
          : 'border-gray-700 hover:border-gray-600'
      } ${onSelect && !disabled ? 'cursor-pointer' : ''}`}
      onClick={() => onSelect && !disabled && onSelect()}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold text-white">Option {option.optionNumber}</span>
          {option.algorithmType && (
            <span className={`px-2 py-0.5 rounded text-xs font-medium ${getAlgorithmBadgeColor(option.algorithmType)}`}>
              {ALGORITHM_LABELS[option.algorithmType] || option.algorithmType}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`px-2 py-1 rounded text-xs font-medium ${
              option.balanceScore >= 80
                ? 'bg-green-600/20 text-green-400'
                : option.balanceScore >= 60
                ? 'bg-yellow-600/20 text-yellow-400'
                : 'bg-red-600/20 text-red-400'
            }`}
          >
            Balance: {option.balanceScore.toFixed(1)}
          </span>
          <span className="text-xs text-gray-400">
            Δ{option.mmrDifference} MMR
          </span>
        </div>
      </div>

      {getAlgorithmHighlight() && (
        <div className="text-xs text-gray-400 mb-3">
          {getAlgorithmHighlight()}
        </div>
      )}

      <div className="grid grid-cols-2 gap-4">
        {renderTeam(blueTeam, 'blue')}
        {renderTeam(redTeam, 'red')}
      </div>

      {onSelect && !disabled && (
        <button
          className={`w-full mt-4 py-2 rounded font-semibold transition-colors ${
            isSelected
              ? 'bg-lol-gold text-black'
              : 'bg-gray-700 hover:bg-gray-600 text-white'
          }`}
        >
          {isSelected ? 'Selected' : 'Select This Option'}
        </button>
      )}
    </div>
  )
}
