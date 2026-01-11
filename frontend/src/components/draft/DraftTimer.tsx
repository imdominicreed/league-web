import { useSelector } from 'react-redux'
import { RootState } from '@/store'

export default function DraftTimer() {
  const { timerRemainingMs, isComplete, currentTeam, isBufferPeriod, isPaused, pausedBy, resumeCountdown } = useSelector((state: RootState) => state.draft)
  const { room } = useSelector((state: RootState) => state.room)

  if (!room || room.status === 'waiting') {
    return null
  }

  if (isComplete) {
    return (
      <div className="text-center">
        <div className="text-2xl font-bold text-lol-gold">Draft Complete!</div>
      </div>
    )
  }

  // Paused state
  if (isPaused) {
    // Show countdown when resuming
    if (resumeCountdown > 0) {
      return (
        <div className="text-center">
          <div className="relative">
            <div className="absolute inset-0 bg-green-600 rounded-lg animate-pulse opacity-20" />
            <div className="relative px-4 py-2">
              <div className="text-5xl font-bold font-mono text-green-500">
                {resumeCountdown}
              </div>
              <div className="text-sm text-green-400 mt-1">
                Resuming...
              </div>
            </div>
          </div>
        </div>
      )
    }

    return (
      <div className="text-center">
        <div className="relative">
          <div className="absolute inset-0 bg-yellow-600 rounded-lg animate-pulse opacity-20" />
          <div className="relative px-4 py-2">
            <div className="text-3xl font-bold font-mono text-yellow-500">
              PAUSED
            </div>
            <div className="text-sm text-yellow-400 mt-1">
              by {pausedBy}
            </div>
            <div className="text-xs text-lol-text-secondary mt-1">
              Click picks/bans to edit
            </div>
          </div>
        </div>
      </div>
    )
  }

  const seconds = Math.max(0, Math.ceil(timerRemainingMs / 1000))
  const isLow = seconds <= 10 && !isBufferPeriod
  const isCritical = seconds <= 5 && !isBufferPeriod

  const sideColor = currentTeam === 'blue' ? 'text-blue-side' : 'text-red-side'

  // Buffer period: timer hit 0, show urgent indicator
  if (isBufferPeriod) {
    return (
      <div className="text-center">
        <div className="relative">
          <div className="absolute inset-0 bg-red-600 rounded-lg animate-pulse opacity-30" />
          <div className="relative px-4 py-2">
            <div className="text-5xl font-bold font-mono text-red-500 animate-bounce">
              0
            </div>
            <div className="text-sm text-red-400 uppercase tracking-wider mt-1 font-bold animate-pulse">
              LOCK IN NOW!
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="text-center">
      <div className={`text-5xl font-bold font-mono ${
        isCritical ? 'text-red-500 animate-pulse' : isLow ? 'text-yellow-500' : 'text-white'
      }`}>
        {seconds}
      </div>
      <div className={`text-sm ${sideColor} uppercase tracking-wider mt-1`}>
        {currentTeam} side's turn
      </div>
    </div>
  )
}
