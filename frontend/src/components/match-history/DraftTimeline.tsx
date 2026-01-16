import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { DraftAction } from '@/types'

interface Props {
  actions: DraftAction[]
}

// Pro play phase info for display
const PHASE_INFO = [
  // Ban Phase 1 (phases 0-5)
  { phase: 0, team: 'blue', action: 'ban', label: 'Ban 1' },
  { phase: 1, team: 'red', action: 'ban', label: 'Ban 1' },
  { phase: 2, team: 'blue', action: 'ban', label: 'Ban 2' },
  { phase: 3, team: 'red', action: 'ban', label: 'Ban 2' },
  { phase: 4, team: 'blue', action: 'ban', label: 'Ban 3' },
  { phase: 5, team: 'red', action: 'ban', label: 'Ban 3' },
  // Pick Phase 1 (phases 6-11)
  { phase: 6, team: 'blue', action: 'pick', label: 'Pick 1' },
  { phase: 7, team: 'red', action: 'pick', label: 'Pick 1' },
  { phase: 8, team: 'red', action: 'pick', label: 'Pick 2' },
  { phase: 9, team: 'blue', action: 'pick', label: 'Pick 2' },
  { phase: 10, team: 'blue', action: 'pick', label: 'Pick 3' },
  { phase: 11, team: 'red', action: 'pick', label: 'Pick 3' },
  // Ban Phase 2 (phases 12-15)
  { phase: 12, team: 'red', action: 'ban', label: 'Ban 4' },
  { phase: 13, team: 'blue', action: 'ban', label: 'Ban 4' },
  { phase: 14, team: 'red', action: 'ban', label: 'Ban 5' },
  { phase: 15, team: 'blue', action: 'ban', label: 'Ban 5' },
  // Pick Phase 2 (phases 16-19)
  { phase: 16, team: 'red', action: 'pick', label: 'Pick 4' },
  { phase: 17, team: 'blue', action: 'pick', label: 'Pick 4' },
  { phase: 18, team: 'blue', action: 'pick', label: 'Pick 5' },
  { phase: 19, team: 'red', action: 'pick', label: 'Pick 5' },
]

export default function DraftTimeline({ actions }: Props) {
  const champions = useSelector((state: RootState) => state.champions.champions)

  const getChampion = (championId: string) => {
    return champions[championId]
  }

  const getActionByPhase = (phaseIndex: number) => {
    return actions.find(a => a.phaseIndex === phaseIndex)
  }

  // Group phases by type
  const banPhase1 = PHASE_INFO.slice(0, 6)
  const pickPhase1 = PHASE_INFO.slice(6, 12)
  const banPhase2 = PHASE_INFO.slice(12, 16)
  const pickPhase2 = PHASE_INFO.slice(16, 20)

  const renderPhaseGroup = (phases: typeof PHASE_INFO, title: string) => (
    <div className="mb-6">
      <h4 className="text-sm font-semibold text-gray-400 mb-2">{title}</h4>
      <div className="flex flex-wrap gap-2">
        {phases.map(phase => {
          const action = getActionByPhase(phase.phase)
          const champion = action ? getChampion(action.championId) : null
          const isNone = action?.championId === 'None'

          return (
            <div
              key={phase.phase}
              className={`flex items-center gap-2 px-3 py-2 rounded-lg ${
                phase.team === 'blue' ? 'bg-blue-900/30' : 'bg-red-900/30'
              }`}
            >
              <div className="flex-shrink-0">
                {champion && !isNone ? (
                  <img
                    src={champion.imageUrl}
                    alt={champion.name}
                    className={`w-10 h-10 rounded ${
                      phase.action === 'ban' ? 'grayscale opacity-60' : ''
                    }`}
                  />
                ) : (
                  <div className={`w-10 h-10 rounded bg-gray-700 flex items-center justify-center text-xs ${
                    phase.action === 'ban' ? 'line-through text-gray-500' : 'text-gray-400'
                  }`}>
                    {isNone ? '-' : '?'}
                  </div>
                )}
              </div>
              <div className="min-w-0">
                <div className={`text-xs font-medium ${
                  phase.team === 'blue' ? 'text-lol-blue' : 'text-red-400'
                }`}>
                  {phase.label}
                </div>
                <div className="text-sm truncate">
                  {champion && !isNone ? champion.name : (isNone ? 'Skipped' : '-')}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )

  return (
    <div className="bg-gray-800 rounded-lg p-4">
      <h3 className="text-lg font-bold text-white mb-4">Draft Timeline</h3>

      {actions.length === 0 ? (
        <p className="text-gray-400 text-sm">No draft actions recorded</p>
      ) : (
        <>
          {renderPhaseGroup(banPhase1, 'Ban Phase 1')}
          {renderPhaseGroup(pickPhase1, 'Pick Phase 1')}
          {renderPhaseGroup(banPhase2, 'Ban Phase 2')}
          {renderPhaseGroup(pickPhase2, 'Pick Phase 2')}
        </>
      )}
    </div>
  )
}
