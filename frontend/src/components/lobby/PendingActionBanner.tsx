import { PendingAction, LobbyPlayer, Side } from '@/types'

interface PendingActionBannerProps {
  action: PendingAction
  players: LobbyPlayer[]
  currentUserId?: string
  currentUserSide?: Side | null
  isCaptain: boolean
  onApprove: () => void
  onCancel: () => void
  approving: boolean
  cancelling: boolean
}

const ACTION_LABELS: Record<string, string> = {
  swap_players: 'Swap Players Between Teams',
  swap_roles: 'Swap Roles Within Team',
  matchmake: 'Generate Team Options',
  select_option: 'Select Team Composition',
  start_draft: 'Start Draft',
}

export function PendingActionBanner({
  action,
  players,
  currentUserId,
  currentUserSide,
  isCaptain,
  onApprove,
  onCancel,
  approving,
  cancelling,
}: PendingActionBannerProps) {
  const proposer = players.find(p => p.userId === action.proposedByUser)
  const player1 = action.player1Id ? players.find(p => p.id === action.player1Id) : null
  const player2 = action.player2Id ? players.find(p => p.id === action.player2Id) : null

  const isProposer = action.proposedByUser === currentUserId
  const hasApproved = currentUserSide === 'blue' ? action.approvedByBlue : action.approvedByRed
  const canApprove = isCaptain && !hasApproved
  const canCancel = isProposer

  const getActionDescription = () => {
    switch (action.actionType) {
      case 'swap_players':
        return player1 && player2
          ? `Swap ${player1.displayName} and ${player2.displayName} between teams`
          : 'Swap players between teams'
      case 'swap_roles':
        return player1 && player2
          ? `Swap roles between ${player1.displayName} and ${player2.displayName}`
          : 'Swap roles within team'
      case 'matchmake':
        return 'Generate balanced team options using matchmaking'
      case 'select_option':
        return action.matchOptionNum !== undefined
          ? `Select team composition option #${action.matchOptionNum}`
          : 'Select a team composition'
      case 'start_draft':
        return 'Start the champion draft'
      default:
        return 'Pending action'
    }
  }

  const getTimeRemaining = () => {
    const expiresAt = new Date(action.expiresAt)
    const now = new Date()
    const diff = Math.max(0, Math.floor((expiresAt.getTime() - now.getTime()) / 1000))
    const mins = Math.floor(diff / 60)
    const secs = diff % 60
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <div className="bg-yellow-900/30 border border-yellow-600 rounded-lg p-4 mb-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-yellow-400 font-semibold">
              {ACTION_LABELS[action.actionType] || 'Pending Action'}
            </span>
            <span className="text-gray-400 text-sm">
              by {proposer?.displayName || 'Unknown'}
            </span>
          </div>
          <p className="text-gray-300 text-sm">{getActionDescription()}</p>

          <div className="flex items-center gap-4 mt-3 text-sm">
            <div className="flex items-center gap-2">
              <span className={`w-3 h-3 rounded-full ${action.approvedByBlue ? 'bg-green-500' : 'bg-gray-500'}`} />
              <span className="text-gray-400">Blue Captain</span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`w-3 h-3 rounded-full ${action.approvedByRed ? 'bg-green-500' : 'bg-gray-500'}`} />
              <span className="text-gray-400">Red Captain</span>
            </div>
            <span className="text-gray-500">Expires: {getTimeRemaining()}</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {canApprove && (
            <button
              onClick={onApprove}
              disabled={approving}
              className="bg-green-600 hover:bg-green-500 text-white px-4 py-2 rounded-lg font-medium transition disabled:opacity-50"
            >
              {approving ? 'Approving...' : 'Approve'}
            </button>
          )}
          {hasApproved && isCaptain && (
            <span className="text-green-400 text-sm font-medium px-3">Approved</span>
          )}
          {canCancel && (
            <button
              onClick={onCancel}
              disabled={cancelling}
              className="bg-gray-600 hover:bg-gray-500 text-white px-4 py-2 rounded-lg font-medium transition disabled:opacity-50"
            >
              {cancelling ? 'Cancelling...' : 'Cancel'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
