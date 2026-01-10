import { useSelector } from 'react-redux'
import { RootState } from '@/store'

interface PauseControlsProps {
  onPause: () => void
  onResume: () => void
}

export default function PauseControls({ onPause, onResume }: PauseControlsProps) {
  const { isPaused, pausedBy, pausedBySide, isComplete, currentPhase } = useSelector(
    (state: RootState) => state.draft
  )
  const { room, isCaptain } = useSelector((state: RootState) => state.room)

  // Only show during active draft
  const isActivePhase = room?.status === 'in_progress' && currentPhase >= 0 && !isComplete

  if (!isActivePhase || !isCaptain) {
    return null
  }

  if (isPaused) {
    return (
      <div className="flex items-center gap-2">
        <div className="bg-yellow-600/80 text-white px-3 py-1 rounded text-sm">
          Paused by {pausedBy} ({pausedBySide})
        </div>
        <button
          onClick={onResume}
          className="bg-green-600 hover:bg-green-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
        >
          Resume
        </button>
      </div>
    )
  }

  return (
    <button
      onClick={onPause}
      className="bg-yellow-600 hover:bg-yellow-500 text-white px-3 py-1 rounded text-sm font-medium transition-colors"
    >
      Pause
    </button>
  )
}
