import { LobbyPlayer, ROLE_DISPLAY_NAMES, Side, ALL_ROLES, PendingAction } from '@/types'

interface TeamColumnProps {
  side: Side
  players: LobbyPlayer[]
  currentUserId?: string
  swapMode?: boolean
  selectedPlayerId?: string | null
  pendingAction?: PendingAction | null
  onPlayerClick?: (player: LobbyPlayer) => void
}

export function TeamColumn({ side, players, currentUserId, swapMode = false, selectedPlayerId, pendingAction, onPlayerClick }: TeamColumnProps) {
  const teamPlayers = players
    .filter(p => p.team === side)
    .sort((a, b) => {
      // Sort by role order: top, jungle, mid, adc, support
      const aIndex = a.assignedRole ? ALL_ROLES.indexOf(a.assignedRole) : 999
      const bIndex = b.assignedRole ? ALL_ROLES.indexOf(b.assignedRole) : 999
      return aIndex - bIndex
    })
  const sideColor = side === 'blue' ? 'lol-blue' : 'lol-red'
  const sideBg = side === 'blue' ? 'bg-blue-900/30' : 'bg-red-900/30'
  const sideBorder = side === 'blue' ? 'border-lol-blue' : 'border-lol-red'

  const getPlayerSlot = (index: number) => {
    const player = teamPlayers[index]
    if (!player) {
      return (
        <div className="bg-gray-800/50 rounded-lg p-3 border border-gray-700 border-dashed flex items-center justify-center min-h-[60px]">
          <span className="text-gray-500 text-sm">Empty slot</span>
        </div>
      )
    }

    const isCurrentUser = player.userId === currentUserId
    const isSelected = player.id === selectedPlayerId

    // Check if player is involved in a pending swap
    const isPendingSwap = pendingAction &&
      pendingAction.status === 'pending' &&
      (pendingAction.actionType === 'swap_players' || pendingAction.actionType === 'swap_roles') &&
      (pendingAction.player1Id === player.userId || pendingAction.player2Id === player.userId)

    // Determine if this player is a valid swap target
    const isValidTarget = swapMode && (() => {
      if (!selectedPlayerId) return true // No one selected yet, all players are valid first picks
      if (player.id === selectedPlayerId) return true // Can click to deselect
      return true // All other players are valid - swap type auto-detected
    })()

    const isClickable = !!onPlayerClick && isValidTarget

    return (
      <div
        onClick={isClickable ? () => onPlayerClick(player) : undefined}
        className={`bg-gray-800 rounded-lg p-3 border-2 transition-all ${
          isSelected
            ? 'border-lol-gold ring-2 ring-lol-gold/50 bg-lol-gold/10'
            : isPendingSwap
            ? 'border-purple-500 ring-2 ring-purple-500/50 bg-purple-500/10'
            : swapMode && isClickable
            ? 'border-dashed border-gray-500 hover:border-lol-gold hover:bg-gray-700/50 cursor-pointer'
            : player.isReady
            ? 'border-green-500/50'
            : isCurrentUser
            ? side === 'blue' ? 'border-lol-blue/50' : 'border-lol-red/50'
            : 'border-gray-700'
        }`}
      >
        <div className="flex items-center gap-3">
          <div
            className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
              isSelected ? 'bg-lol-gold text-black' : player.isReady ? 'bg-green-600' : 'bg-gray-700'
            }`}
          >
            {player.displayName?.[0]?.toUpperCase() || '?'}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-white font-medium truncate text-sm">
                {player.displayName || 'Unknown'}
                {isCurrentUser && <span className="text-gray-400 text-xs ml-1">(You)</span>}
              </p>
              {player.isCaptain && (
                <span className="text-lol-gold text-xs font-bold" title="Captain">C</span>
              )}
            </div>
            {player.assignedRole && (
              <p className="text-xs text-gray-400">
                {ROLE_DISPLAY_NAMES[player.assignedRole]}
              </p>
            )}
          </div>
          <div
            className={`w-2 h-2 rounded-full ${
              player.isReady ? 'bg-green-500' : 'bg-gray-500'
            }`}
            title={player.isReady ? 'Ready' : 'Not ready'}
          />
        </div>
      </div>
    )
  }

  return (
    <div className={`${sideBg} rounded-lg p-4 border ${sideBorder}`}>
      <h3 className={`text-lg font-bold mb-4 text-${sideColor} capitalize`}>
        {side} Team
        <span className="text-gray-400 text-sm font-normal ml-2">
          ({teamPlayers.length}/5)
        </span>
      </h3>
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i}>{getPlayerSlot(i)}</div>
        ))}
      </div>
    </div>
  )
}
