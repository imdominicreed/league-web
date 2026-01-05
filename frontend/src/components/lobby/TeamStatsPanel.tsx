import { TeamStats, ROLE_DISPLAY_NAMES, Role } from '@/types'

interface TeamStatsPanelProps {
  stats: TeamStats
  loading?: boolean
}

export function TeamStatsPanel({ stats, loading }: TeamStatsPanelProps) {
  if (loading) {
    return (
      <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
        <div className="text-center text-gray-400">Loading stats...</div>
      </div>
    )
  }

  const mmrDiffAbs = Math.abs(stats.mmrDifference)
  const mmrAdvantage = stats.mmrDifference > 0 ? 'Blue' : stats.mmrDifference < 0 ? 'Red' : 'Even'

  const getBalanceColor = (diff: number) => {
    const absDiff = Math.abs(diff)
    if (absDiff < 50) return 'text-green-400'
    if (absDiff < 150) return 'text-yellow-400'
    return 'text-red-400'
  }

  const getLaneColor = (diff: number) => {
    const absDiff = Math.abs(diff)
    if (absDiff < 100) return 'text-green-400'
    if (absDiff < 250) return 'text-yellow-400'
    return 'text-red-400'
  }

  return (
    <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
      <h4 className="text-sm font-semibold text-gray-400 mb-4 uppercase tracking-wide">
        Team Comparison
      </h4>

      <div className="space-y-4">
        {/* MMR Balance */}
        <div>
          <div className="flex justify-between items-center mb-1">
            <span className="text-gray-400 text-sm">Team MMR</span>
            <span className={`text-sm font-medium ${getBalanceColor(stats.mmrDifference)}`}>
              {mmrDiffAbs === 0 ? 'Balanced' : `${mmrAdvantage} +${mmrDiffAbs}`}
            </span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="text-lol-blue">{stats.blueTeamAvgMmr}</span>
            <div className="flex-1 h-2 bg-gray-700 rounded-full relative overflow-hidden">
              <div
                className="absolute top-0 left-0 h-full bg-lol-blue rounded-l-full"
                style={{ width: `${(stats.blueTeamAvgMmr / (stats.blueTeamAvgMmr + stats.redTeamAvgMmr)) * 100}%` }}
              />
              <div
                className="absolute top-0 right-0 h-full bg-lol-red rounded-r-full"
                style={{ width: `${(stats.redTeamAvgMmr / (stats.blueTeamAvgMmr + stats.redTeamAvgMmr)) * 100}%` }}
              />
            </div>
            <span className="text-lol-red">{stats.redTeamAvgMmr}</span>
          </div>
        </div>

        {/* Comfort Rating */}
        <div>
          <div className="flex justify-between items-center mb-1">
            <span className="text-gray-400 text-sm">Avg Comfort</span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="text-lol-blue">{stats.avgBlueComfort.toFixed(1)}</span>
            <div className="flex-1 h-2 bg-gray-700 rounded-full relative overflow-hidden">
              <div
                className="absolute top-0 left-0 h-full bg-lol-blue rounded-l-full"
                style={{ width: `${(stats.avgBlueComfort / 5) * 50}%` }}
              />
              <div
                className="absolute top-0 right-0 h-full bg-lol-red rounded-r-full"
                style={{ width: `${(stats.avgRedComfort / 5) * 50}%` }}
              />
            </div>
            <span className="text-lol-red">{stats.avgRedComfort.toFixed(1)}</span>
          </div>
        </div>

        {/* Lane Matchups */}
        <div>
          <div className="text-gray-400 text-sm mb-2">Lane Matchups</div>
          <div className="space-y-1">
            {(['top', 'jungle', 'mid', 'adc', 'support'] as Role[]).map(role => {
              const diff = stats.laneDiffs[role] || 0
              const advantage = diff > 0 ? 'Blue' : diff < 0 ? 'Red' : null
              return (
                <div key={role} className="flex items-center justify-between text-xs">
                  <span className="text-gray-500 w-16">{ROLE_DISPLAY_NAMES[role]}</span>
                  <span className={getLaneColor(diff)}>
                    {advantage ? `${advantage} +${Math.abs(diff)}` : 'Even'}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}
