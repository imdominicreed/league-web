import { VotingStatus, VOTING_MODE_LABELS, VotingMode } from '@/types'
import { useEffect, useState } from 'react'

interface VotingBannerProps {
  votingStatus: VotingStatus
  isCaptain: boolean
  canForceOption: boolean
  winningOptionNum?: number
  onEndVoting: (forceOption?: number) => void
  endingVoting: boolean
}

export function VotingBanner({
  votingStatus,
  isCaptain,
  canForceOption,
  winningOptionNum,
  onEndVoting,
  endingVoting,
}: VotingBannerProps) {
  const [timeRemaining, setTimeRemaining] = useState<string | null>(null)

  useEffect(() => {
    if (!votingStatus.deadline) {
      setTimeRemaining(null)
      return
    }

    const updateTimer = () => {
      const deadline = new Date(votingStatus.deadline!)
      const now = new Date()
      const diff = Math.max(0, Math.floor((deadline.getTime() - now.getTime()) / 1000))

      if (diff === 0) {
        setTimeRemaining('Expired')
        return
      }

      const mins = Math.floor(diff / 60)
      const secs = diff % 60
      setTimeRemaining(`${mins}:${secs.toString().padStart(2, '0')}`)
    }

    updateTimer()
    const interval = setInterval(updateTimer, 1000)
    return () => clearInterval(interval)
  }, [votingStatus.deadline])

  const getVotingModeLabel = (mode: VotingMode) => {
    return VOTING_MODE_LABELS[mode] || mode
  }

  const votingProgress = votingStatus.totalPlayers > 0
    ? Math.round((votingStatus.votesCast / votingStatus.totalPlayers) * 100)
    : 0

  return (
    <div data-testid="voting-banner" className="bg-purple-900/30 border border-purple-600 rounded-lg p-4 mb-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-purple-400 font-semibold">
              Voting in Progress
            </span>
            <span className="bg-purple-600/20 text-purple-400 px-2 py-0.5 rounded text-xs font-medium">
              {getVotingModeLabel(votingStatus.votingMode)}
            </span>
          </div>

          <div className="flex items-center gap-4 mt-2 text-sm">
            <div className="flex items-center gap-2">
              <span className="text-gray-400">Votes:</span>
              <span className="text-white font-medium">
                {votingStatus.votesCast} / {votingStatus.totalPlayers}
              </span>
              <span className="text-gray-500">({votingProgress}%)</span>
            </div>

            {timeRemaining && (
              <div className="flex items-center gap-2">
                <span className="text-gray-400">Time:</span>
                <span className={`font-mono font-medium ${timeRemaining === 'Expired' ? 'text-red-400' : 'text-white'}`}>
                  {timeRemaining}
                </span>
              </div>
            )}
          </div>

          {votingStatus.winningOption !== undefined && (
            <div className="mt-2 flex items-center gap-2">
              <span className="text-gray-400 text-sm">Leading:</span>
              <span className="text-lol-gold font-medium">
                Option {votingStatus.winningOption}
              </span>
              {votingStatus.canFinalize && (
                <span className="text-green-400 text-xs">(Can finalize)</span>
              )}
            </div>
          )}
        </div>

        {isCaptain && (
          <div className="flex items-center gap-2">
            {votingStatus.canFinalize && (
              <button
                onClick={() => onEndVoting()}
                disabled={endingVoting}
                className="bg-green-600 hover:bg-green-500 text-white px-4 py-2 rounded-lg font-medium transition disabled:opacity-50"
              >
                {endingVoting ? 'Finalizing...' : 'Finalize Vote'}
              </button>
            )}
            {canForceOption && winningOptionNum !== undefined && !votingStatus.canFinalize && (
              <button
                onClick={() => onEndVoting(winningOptionNum)}
                disabled={endingVoting}
                className="bg-yellow-600 hover:bg-yellow-500 text-white px-4 py-2 rounded-lg font-medium transition disabled:opacity-50"
                title="Captain override: force the leading option"
              >
                {endingVoting ? 'Forcing...' : 'Force Option'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
