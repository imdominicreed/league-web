import { useSelector } from 'react-redux'
import { RootState } from '@/store'

interface PauseControlsProps {
  onPause: () => void
  onReadyToResume: (ready: boolean) => void
}

export default function PauseControls({ onPause, onReadyToResume }: PauseControlsProps) {
  const {
    isPaused,
    pausedBy,
    pausedBySide,
    isComplete,
    currentPhase,
    blueResumeReady,
    redResumeReady,
    resumeCountdown,
    yourSide,
  } = useSelector((state: RootState) => state.draft)
  const { room, isCaptain } = useSelector((state: RootState) => state.room)

  // Only show during active draft
  const isActivePhase = room?.status === 'in_progress' && currentPhase >= 0 && !isComplete

  if (!isActivePhase || !isCaptain) {
    return null
  }

  if (isPaused) {
    const myReady = yourSide === 'blue' ? blueResumeReady : redResumeReady

    // During countdown
    if (resumeCountdown > 0) {
      return (
        <div className="flex items-center gap-3">
          <div className="bg-green-600/80 text-white px-3 py-1 rounded text-sm animate-pulse">
            Resuming in {resumeCountdown}...
          </div>
          <button
            onClick={() => onReadyToResume(false)}
            className="bg-red-600 hover:bg-red-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
          >
            Cancel
          </button>
        </div>
      )
    }

    return (
      <div className="flex items-center gap-3">
        <div className="bg-yellow-600/80 text-white px-3 py-1 rounded text-sm">
          Paused by {pausedBy} ({pausedBySide})
        </div>

        {/* Ready indicators */}
        <div className="flex gap-2 text-xs">
          <span
            className={`px-2 py-1 rounded ${
              blueResumeReady ? 'bg-blue-600 text-white' : 'bg-gray-600 text-gray-300'
            }`}
          >
            Blue: {blueResumeReady ? 'Ready' : 'Not Ready'}
          </span>
          <span
            className={`px-2 py-1 rounded ${
              redResumeReady ? 'bg-red-600 text-white' : 'bg-gray-600 text-gray-300'
            }`}
          >
            Red: {redResumeReady ? 'Ready' : 'Not Ready'}
          </span>
        </div>

        {/* Ready button */}
        {myReady ? (
          <button
            onClick={() => onReadyToResume(false)}
            className="bg-gray-600 hover:bg-gray-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
          >
            Cancel Ready
          </button>
        ) : (
          <button
            onClick={() => onReadyToResume(true)}
            className="bg-green-600 hover:bg-green-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
          >
            Ready to Resume
          </button>
        )}
      </div>
    )
  }

  // Not paused - show pause button
  return (
    <button
      onClick={onPause}
      className="bg-yellow-600 hover:bg-yellow-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
    >
      Pause
    </button>
  )
}
