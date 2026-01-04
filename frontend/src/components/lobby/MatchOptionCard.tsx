import { MatchOption, MatchAssignment, ROLE_DISPLAY_NAMES } from '@/types'

interface MatchOptionCardProps {
  option: MatchOption
  isSelected: boolean
  onSelect?: () => void
  disabled?: boolean
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

  const renderTeam = (team: MatchAssignment[], side: 'blue' | 'red') => (
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
      <div className="text-xs text-gray-400 mt-1">
        Avg MMR: {side === 'blue' ? option.blueTeamAvgMmr : option.redTeamAvgMmr}
      </div>
    </div>
  )

  return (
    <div
      className={`bg-gray-800 rounded-lg p-4 border-2 transition-all ${
        isSelected
          ? 'border-lol-gold shadow-lg shadow-lol-gold/20'
          : 'border-gray-700 hover:border-gray-600'
      } ${onSelect && !disabled ? 'cursor-pointer' : ''}`}
      onClick={() => onSelect && !disabled && onSelect()}
    >
      <div className="flex items-center justify-between mb-4">
        <span className="text-lg font-bold text-white">Option {option.optionNumber}</span>
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
