import { useSelector } from 'react-redux'
import { RootState } from '@/store'

export default function DraftTimer() {
  const { timerRemainingMs, isComplete, currentTeam } = useSelector((state: RootState) => state.draft)
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

  const seconds = Math.ceil(timerRemainingMs / 1000)
  const isLow = seconds <= 10
  const isCritical = seconds <= 5

  const sideColor = currentTeam === 'blue' ? 'text-blue-side' : 'text-red-side'

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
