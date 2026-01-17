import { MatchOption, MatchAssignment, ROLE_DISPLAY_NAMES, ALGORITHM_LABELS, AlgorithmType, VoterInfo } from '@/types'

interface MatchOptionCardProps {
  option: MatchOption
  isSelected: boolean
  onSelect?: () => void
  disabled?: boolean
  voteCount?: number
  totalVotes?: number
  isVotingEnabled?: boolean
  userVotes?: number[] // options the user has voted for
  voters?: VoterInfo[]
}

function getAlgorithmBadgeColor(type: AlgorithmType): string {
  switch (type) {
    case 'mmr_balanced':
      return 'bg-blue-600/20 text-blue-400'
    case 'role_comfort':
      return 'bg-purple-600/20 text-purple-400'
    case 'lane_balanced':
      return 'bg-green-600/20 text-green-400'
    case 'comfort_first':
      return 'bg-teal-600/20 text-teal-400'
    default:
      return 'bg-gray-600/20 text-gray-400'
  }
}

export function MatchOptionCard({
  option,
  isSelected,
  onSelect,
  disabled,
  voteCount,
  totalVotes,
  isVotingEnabled,
  userVotes,
  voters,
}: MatchOptionCardProps) {
  const hasUserVoted = userVotes?.includes(option.optionNumber) ?? false
  const votePercentage = totalVotes && totalVotes > 0 ? Math.round((voteCount || 0) / totalVotes * 100) : 0
  const blueTeam = option.assignments.filter(a => a.team === 'blue')
  const redTeam = option.assignments.filter(a => a.team === 'red')

  // Sort by role order
  const roleOrder = ['top', 'jungle', 'mid', 'adc', 'support']
  const sortByRole = (a: MatchAssignment, b: MatchAssignment) =>
    roleOrder.indexOf(a.assignedRole) - roleOrder.indexOf(b.assignedRole)

  blueTeam.sort(sortByRole)
  redTeam.sort(sortByRole)

  const renderTeam = (team: MatchAssignment[], side: 'blue' | 'red') => {
    const avgComfort = side === 'blue' ? (option.avgBlueComfort ?? 0) : (option.avgRedComfort ?? 0)
    const avgMmr = side === 'blue' ? option.blueTeamAvgMmr : option.redTeamAvgMmr
    return (
      <div className="space-y-1">
        <div className="flex items-center justify-between mb-2">
          <h4
            className={`font-semibold text-sm ${
              side === 'blue' ? 'text-blue-400' : 'text-red-400'
            }`}
          >
            {side === 'blue' ? 'Blue' : 'Red'}
          </h4>
          <span className="text-xs text-gray-500">{avgMmr} avg</span>
        </div>
        {team.map(player => (
          <div
            key={player.userId}
            className="bg-gray-700/50 rounded px-2 py-1.5"
          >
            <div className="flex items-center gap-2 text-sm">
              <span className="text-gray-400 text-xs font-medium w-8 shrink-0">
                {ROLE_DISPLAY_NAMES[player.assignedRole].slice(0, 3).toUpperCase()}
              </span>
              <span className="text-white truncate flex-1 min-w-0">
                {player.displayName || 'Unknown'}
              </span>
              <span className="text-gray-500 text-xs shrink-0 tabular-nums">{player.roleMmr}</span>
              <span className="text-yellow-400 text-xs shrink-0" title={`Comfort: ${player.comfortRating}/5`}>
                {player.comfortRating}★
              </span>
            </div>
          </div>
        ))}
        <div className="text-xs text-gray-500 pt-1">
          Comfort: {avgComfort.toFixed(1)}/5
        </div>
      </div>
    )
  }

  const borderClass = isSelected
    ? 'border-lol-gold shadow-lg shadow-lol-gold/20'
    : hasUserVoted && isVotingEnabled
    ? 'border-purple-500 shadow-lg shadow-purple-500/20'
    : 'border-gray-700 hover:border-gray-600'

  return (
    <div
      data-testid={`match-option-${option.optionNumber}`}
      className={`bg-gray-800 rounded-lg p-4 border-2 transition-all ${borderClass} ${onSelect && !disabled ? 'cursor-pointer' : ''}`}
      onClick={() => onSelect && !disabled && onSelect()}
    >
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2">
          <span className="text-lg font-bold text-white">Option {option.optionNumber}</span>
          {hasUserVoted && isVotingEnabled && (
            <span className="bg-purple-600/20 text-purple-400 px-2 py-0.5 rounded text-xs font-medium">
              Your Vote
            </span>
          )}
        </div>
        <span
          className={`px-2 py-0.5 rounded text-xs font-medium ${
            option.balanceScore >= 80
              ? 'bg-green-600/20 text-green-400'
              : option.balanceScore >= 60
              ? 'bg-yellow-600/20 text-yellow-400'
              : 'bg-red-600/20 text-red-400'
          }`}
        >
          {option.balanceScore.toFixed(0)}%
        </span>
      </div>

      <div className="flex items-center gap-2 mb-3 flex-wrap">
        {option.algorithmType && (
          <span className={`px-2 py-0.5 rounded text-xs font-medium ${getAlgorithmBadgeColor(option.algorithmType)}`}>
            {ALGORITHM_LABELS[option.algorithmType] || option.algorithmType}
          </span>
        )}
        {option.usedMmrThreshold > 0 && (
          <span className="bg-gray-600/20 text-gray-400 px-2 py-0.5 rounded text-xs font-medium">
            Within {option.usedMmrThreshold} ELO
          </span>
        )}
        <span className="text-xs text-gray-400">
          Δ{option.mmrDifference} MMR
        </span>
        {option.maxLaneDiff !== undefined && option.maxLaneDiff > 0 && (
          <span className="text-xs text-gray-500">
            Max lane gap: {option.maxLaneDiff}
          </span>
        )}
        {isVotingEnabled && voteCount !== undefined && (
          <span className="text-xs text-purple-400 font-medium">
            {voteCount} vote{voteCount !== 1 ? 's' : ''} ({votePercentage}%)
          </span>
        )}
      </div>

      {/* Voters list */}
      {isVotingEnabled && voters && voters.length > 0 && (
        <div data-testid={`voters-option-${option.optionNumber}`} className="mb-3 flex flex-wrap gap-1">
          {voters.map((voter) => (
            <span
              key={voter.userId}
              className="bg-purple-600/20 text-purple-300 px-2 py-0.5 rounded text-xs"
              data-testid={`voter-${voter.userId}`}
            >
              {voter.displayName}
            </span>
          ))}
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
              : hasUserVoted && isVotingEnabled
              ? 'bg-purple-600 hover:bg-purple-500 text-white'
              : 'bg-gray-700 hover:bg-gray-600 text-white'
          }`}
        >
          {isSelected
            ? 'Selected'
            : isVotingEnabled
            ? hasUserVoted
              ? 'Voted (Click to Remove)'
              : 'Vote for This'
            : 'Select This Option'}
        </button>
      )}
    </div>
  )
}
