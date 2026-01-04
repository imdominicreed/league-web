import { useSelector } from 'react-redux'
import { RootState } from '@/store'

const PHASES = [
  // Ban Phase 1
  { team: 'blue', type: 'ban' },
  { team: 'red', type: 'ban' },
  { team: 'blue', type: 'ban' },
  { team: 'red', type: 'ban' },
  { team: 'blue', type: 'ban' },
  { team: 'red', type: 'ban' },
  // Pick Phase 1
  { team: 'blue', type: 'pick' },
  { team: 'red', type: 'pick' },
  { team: 'red', type: 'pick' },
  { team: 'blue', type: 'pick' },
  // Ban Phase 2
  { team: 'red', type: 'ban' },
  { team: 'blue', type: 'ban' },
  { team: 'red', type: 'ban' },
  { team: 'blue', type: 'ban' },
  // Pick Phase 2
  { team: 'red', type: 'pick' },
  { team: 'blue', type: 'pick' },
  { team: 'blue', type: 'pick' },
  { team: 'red', type: 'pick' },
  { team: 'blue', type: 'pick' },
  { team: 'red', type: 'pick' },
]

export default function PhaseIndicator() {
  const { currentPhase, actionType, isComplete } = useSelector((state: RootState) => state.draft)
  const { room } = useSelector((state: RootState) => state.room)

  if (!room || room.status === 'waiting') {
    return <div className="text-gray-400">Waiting to start...</div>
  }

  if (isComplete) {
    return <div className="text-lol-gold font-semibold">Draft Complete</div>
  }

  const phase = PHASES[currentPhase]
  if (!phase) return null

  return (
    <div className="flex items-center gap-2">
      <span className={`font-semibold ${phase.team === 'blue' ? 'text-blue-side' : 'text-red-side'}`}>
        {phase.team === 'blue' ? 'Blue' : 'Red'}
      </span>
      <span className="text-gray-400">
        {phase.type === 'ban' ? 'Ban' : 'Pick'}
      </span>
      <span className="text-gray-600 text-sm">
        ({currentPhase + 1}/20)
      </span>
    </div>
  )
}
