import { LobbyPlayer, ROLE_DISPLAY_NAMES } from '@/types'

interface LobbyPlayerGridProps {
  players: LobbyPlayer[]
  currentUserId?: string
  onReady?: (ready: boolean) => void
}

export function LobbyPlayerGrid({ players, currentUserId, onReady }: LobbyPlayerGridProps) {
  const currentPlayer = players.find(p => p.userId === currentUserId)

  const getPlayerSlot = (index: number) => {
    const player = players[index]
    if (!player) {
      return (
        <div className="bg-gray-800 rounded-lg p-4 border border-gray-700 border-dashed flex items-center justify-center min-h-[100px]">
          <span className="text-gray-500">Waiting for player...</span>
        </div>
      )
    }

    const isCurrentUser = player.userId === currentUserId

    return (
      <div
        className={`bg-gray-800 rounded-lg p-4 border transition-colors ${
          player.isReady
            ? 'border-green-500'
            : isCurrentUser
            ? 'border-lol-blue'
            : 'border-gray-700'
        }`}
      >
        <div className="flex items-center gap-3">
          <div
            className={`w-10 h-10 rounded-full flex items-center justify-center text-lg font-bold ${
              player.isReady ? 'bg-green-600' : 'bg-gray-700'
            }`}
          >
            {player.displayName?.[0]?.toUpperCase() || '?'}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-white font-medium truncate">
              {player.displayName || 'Unknown'}
              {isCurrentUser && <span className="text-gray-400 text-sm ml-2">(You)</span>}
            </p>
            {player.assignedRole && (
              <p className="text-sm text-lol-gold">
                {ROLE_DISPLAY_NAMES[player.assignedRole]}
              </p>
            )}
          </div>
          <div
            className={`px-2 py-1 rounded text-xs font-medium ${
              player.isReady
                ? 'bg-green-600/20 text-green-400'
                : 'bg-gray-700 text-gray-400'
            }`}
          >
            {player.isReady ? 'Ready' : 'Not Ready'}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {Array.from({ length: 10 }).map((_, i) => (
          <div key={i}>{getPlayerSlot(i)}</div>
        ))}
      </div>

      {currentPlayer && onReady && (
        <div className="flex justify-center">
          <button
            onClick={() => onReady(!currentPlayer.isReady)}
            className={`px-8 py-3 rounded-lg font-semibold transition-colors ${
              currentPlayer.isReady
                ? 'bg-red-600 hover:bg-red-700 text-white'
                : 'bg-green-600 hover:bg-green-700 text-white'
            }`}
          >
            {currentPlayer.isReady ? 'Cancel Ready' : 'Ready Up'}
          </button>
        </div>
      )}

      <div className="text-center text-gray-400">
        {players.length}/10 players
        {players.length === 10 && (
          <span className="ml-2">
            ({players.filter(p => p.isReady).length}/10 ready)
          </span>
        )}
      </div>
    </div>
  )
}
