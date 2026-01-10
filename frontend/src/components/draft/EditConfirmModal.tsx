import { useEffect, useState } from 'react'
import { useSelector } from 'react-redux'
import { RootState } from '@/store'

interface EditConfirmModalProps {
  onConfirm: () => void
  onReject: () => void
  champions: Map<string, { name: string; imageUrl: string }>
}

export default function EditConfirmModal({ onConfirm, onReject, champions }: EditConfirmModalProps) {
  const { pendingEdit, yourSide } = useSelector((state: RootState) => state.draft)
  const { isCaptain } = useSelector((state: RootState) => state.room)
  const [timeRemaining, setTimeRemaining] = useState(30)

  // Update countdown timer
  useEffect(() => {
    if (!pendingEdit) return

    const updateTimer = () => {
      const remaining = Math.max(0, Math.floor((pendingEdit.expiresAt - Date.now()) / 1000))
      setTimeRemaining(remaining)
    }

    updateTimer()
    const interval = setInterval(updateTimer, 1000)
    return () => clearInterval(interval)
  }, [pendingEdit])

  // Only show if there's a pending edit and you're the opposite captain
  if (!pendingEdit || !isCaptain || pendingEdit.proposedSide === yourSide) {
    return null
  }

  const oldChampion = champions.get(pendingEdit.oldChampionId)
  const newChampion = champions.get(pendingEdit.newChampionId)
  const slotLabel = `${pendingEdit.team === 'blue' ? 'Blue' : 'Red'}'s ${pendingEdit.slotType === 'ban' ? 'Ban' : 'Pick'} #${pendingEdit.slotIndex + 1}`

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="bg-lol-dark-blue border border-lol-border rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
        <h3 className="text-lg font-bold text-lol-gold mb-4">Edit Request</h3>

        <p className="text-lol-text-secondary mb-4">
          <span className="text-white font-medium">{pendingEdit.proposedBy}</span> wants to change{' '}
          <span className="text-lol-gold">{slotLabel}</span>
        </p>

        <div className="flex items-center justify-center gap-4 mb-6">
          {/* Old champion */}
          <div className="text-center">
            <div className="w-16 h-16 rounded-lg overflow-hidden border-2 border-red-500/50 bg-lol-dark">
              {oldChampion ? (
                <img
                  src={oldChampion.imageUrl}
                  alt={oldChampion.name}
                  className="w-full h-full object-cover opacity-50"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center text-xs text-lol-text-secondary">
                  {pendingEdit.oldChampionId}
                </div>
              )}
            </div>
            <span className="text-xs text-red-400 mt-1 block">
              {oldChampion?.name || pendingEdit.oldChampionId}
            </span>
          </div>

          {/* Arrow */}
          <div className="text-2xl text-lol-gold">→</div>

          {/* New champion */}
          <div className="text-center">
            <div className="w-16 h-16 rounded-lg overflow-hidden border-2 border-green-500 bg-lol-dark">
              {newChampion ? (
                <img
                  src={newChampion.imageUrl}
                  alt={newChampion.name}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center text-xs text-lol-text-secondary">
                  {pendingEdit.newChampionId}
                </div>
              )}
            </div>
            <span className="text-xs text-green-400 mt-1 block">
              {newChampion?.name || pendingEdit.newChampionId}
            </span>
          </div>
        </div>

        {/* Countdown */}
        <div className="text-center mb-4">
          <span className={`text-sm ${timeRemaining <= 10 ? 'text-red-400' : 'text-lol-text-secondary'}`}>
            Auto-reject in {timeRemaining}s
          </span>
        </div>

        {/* Buttons */}
        <div className="flex gap-3">
          <button
            onClick={onReject}
            className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded font-medium transition-colors"
          >
            Reject
          </button>
          <button
            onClick={onConfirm}
            className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-500 text-white rounded font-medium transition-colors"
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  )
}
