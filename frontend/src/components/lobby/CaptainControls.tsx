import { useState } from 'react'
import { LobbyPlayer, Side } from '@/types'

interface CaptainControlsProps {
  players: LobbyPlayer[]
  currentUserId: string
  currentUserSide: Side
  isCaptain: boolean
  hasTeams: boolean
  hasPendingAction: boolean
  onTakeCaptain: () => void
  onPromoteCaptain: (userId: string) => void
  onKickPlayer: (userId: string) => void
  onProposeSwap: (player1Id: string, player2Id: string, swapType: 'players' | 'roles') => void
  onProposeMatchmake: () => void
  onProposeStartDraft: () => void
  onSetReady: (ready: boolean) => void
  isReady: boolean
  takingCaptain: boolean
  promotingCaptain: boolean
  kickingPlayer: boolean
  proposingAction: boolean
}

export function CaptainControls({
  players,
  currentUserId,
  currentUserSide,
  isCaptain,
  hasTeams,
  hasPendingAction,
  onTakeCaptain,
  onPromoteCaptain,
  onKickPlayer,
  onProposeSwap,
  onProposeMatchmake,
  onProposeStartDraft,
  onSetReady,
  isReady,
  takingCaptain,
  promotingCaptain,
  kickingPlayer,
  proposingAction,
}: CaptainControlsProps) {
  const [showSwapModal, setShowSwapModal] = useState(false)
  const [showPromoteModal, setShowPromoteModal] = useState(false)
  const [showKickModal, setShowKickModal] = useState(false)
  const [selectedPlayer1, setSelectedPlayer1] = useState<string | null>(null)
  const [selectedPlayer2, setSelectedPlayer2] = useState<string | null>(null)
  const [swapType, setSwapType] = useState<'players' | 'roles'>('players')

  const teamPlayers = players.filter(p => p.team === currentUserSide && p.userId !== currentUserId)
  const allPlayers = players.filter(p => p.userId !== currentUserId)

  const handleSwapConfirm = () => {
    if (selectedPlayer1 && selectedPlayer2) {
      onProposeSwap(selectedPlayer1, selectedPlayer2, swapType)
      setShowSwapModal(false)
      setSelectedPlayer1(null)
      setSelectedPlayer2(null)
    }
  }

  const allReady = hasTeams && players.length === 10 && players.every(p => p.isReady)

  return (
    <div className="space-y-4">
      {/* Ready Button - always visible */}
      <div className="flex justify-center">
        <button
          onClick={() => onSetReady(!isReady)}
          className={`px-8 py-3 rounded-lg font-semibold transition-colors ${
            isReady
              ? 'bg-red-600 hover:bg-red-700 text-white'
              : 'bg-green-600 hover:bg-green-700 text-white'
          }`}
        >
          {isReady ? 'Cancel Ready' : 'Ready Up'}
        </button>
      </div>

      {/* Captain Controls */}
      <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">
            {isCaptain ? 'Captain Controls' : 'Player Actions'}
          </h4>
          {isCaptain && (
            <span className="text-lol-gold text-xs font-medium px-2 py-1 bg-lol-gold/10 rounded">
              Captain
            </span>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          {/* Take Captain - always available if not captain */}
          {!isCaptain && (
            <button
              onClick={onTakeCaptain}
              disabled={takingCaptain}
              className="bg-lol-gold/20 hover:bg-lol-gold/30 text-lol-gold px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
            >
              {takingCaptain ? 'Taking...' : 'Take Captain'}
            </button>
          )}

          {/* Captain-only actions */}
          {isCaptain && (
            <>
              {/* Promote Captain */}
              <button
                onClick={() => setShowPromoteModal(true)}
                disabled={promotingCaptain || teamPlayers.length === 0}
                className="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
              >
                Promote Captain
              </button>

              {/* Kick Player */}
              <button
                onClick={() => setShowKickModal(true)}
                disabled={kickingPlayer || teamPlayers.length === 0}
                className="bg-red-600/20 hover:bg-red-600/30 text-red-400 px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
              >
                Kick Player
              </button>

              {/* Swap */}
              <button
                onClick={() => setShowSwapModal(true)}
                disabled={proposingAction || hasPendingAction || players.length < 2}
                className="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
              >
                Propose Swap
              </button>

              {/* Matchmake */}
              {!hasTeams && (
                <button
                  onClick={onProposeMatchmake}
                  disabled={proposingAction || hasPendingAction || players.length < 10}
                  className="bg-lol-blue/20 hover:bg-lol-blue/30 text-lol-blue px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
                >
                  {proposingAction ? 'Proposing...' : 'Propose Matchmake'}
                </button>
              )}

              {/* Start Draft */}
              {hasTeams && allReady && (
                <button
                  onClick={onProposeStartDraft}
                  disabled={proposingAction || hasPendingAction}
                  className="bg-green-600 hover:bg-green-500 text-white px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
                >
                  {proposingAction ? 'Proposing...' : 'Propose Start Draft'}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Swap Modal */}
      {showSwapModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-4">Propose Swap</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-2">Swap Type</label>
                <div className="flex gap-2">
                  <button
                    onClick={() => setSwapType('players')}
                    className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition ${
                      swapType === 'players'
                        ? 'bg-lol-blue text-white'
                        : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                    }`}
                  >
                    Between Teams
                  </button>
                  <button
                    onClick={() => setSwapType('roles')}
                    className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition ${
                      swapType === 'roles'
                        ? 'bg-lol-blue text-white'
                        : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                    }`}
                  >
                    Swap Roles
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-2">Player 1</label>
                <select
                  value={selectedPlayer1 || ''}
                  onChange={e => setSelectedPlayer1(e.target.value || null)}
                  className="w-full bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white"
                >
                  <option value="">Select player...</option>
                  {allPlayers.map(p => (
                    <option key={p.userId} value={p.userId}>
                      {p.displayName} ({p.team})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-sm text-gray-400 mb-2">Player 2</label>
                <select
                  value={selectedPlayer2 || ''}
                  onChange={e => setSelectedPlayer2(e.target.value || null)}
                  className="w-full bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white"
                >
                  <option value="">Select player...</option>
                  {allPlayers.filter(p => p.userId !== selectedPlayer1).map(p => (
                    <option key={p.userId} value={p.userId}>
                      {p.displayName} ({p.team})
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="flex gap-2 mt-6">
              <button
                onClick={() => setShowSwapModal(false)}
                className="flex-1 bg-gray-700 hover:bg-gray-600 text-white py-2 rounded-lg font-medium transition"
              >
                Cancel
              </button>
              <button
                onClick={handleSwapConfirm}
                disabled={!selectedPlayer1 || !selectedPlayer2}
                className="flex-1 bg-lol-gold text-black py-2 rounded-lg font-medium transition disabled:opacity-50"
              >
                Propose
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Promote Modal */}
      {showPromoteModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-4">Promote Captain</h3>
            <p className="text-gray-400 text-sm mb-4">Select a teammate to become captain:</p>

            <div className="space-y-2">
              {teamPlayers.map(p => (
                <button
                  key={p.userId}
                  onClick={() => {
                    onPromoteCaptain(p.userId)
                    setShowPromoteModal(false)
                  }}
                  disabled={promotingCaptain}
                  className="w-full bg-gray-800 hover:bg-gray-700 border border-gray-600 rounded-lg px-4 py-3 text-left transition disabled:opacity-50"
                >
                  <span className="text-white font-medium">{p.displayName}</span>
                </button>
              ))}
            </div>

            <button
              onClick={() => setShowPromoteModal(false)}
              className="w-full mt-4 bg-gray-700 hover:bg-gray-600 text-white py-2 rounded-lg font-medium transition"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Kick Modal */}
      {showKickModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-gray-900 border border-gray-700 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-white mb-4">Kick Player</h3>
            <p className="text-gray-400 text-sm mb-4">Select a teammate to kick from the lobby:</p>

            <div className="space-y-2">
              {teamPlayers.map(p => (
                <button
                  key={p.userId}
                  onClick={() => {
                    onKickPlayer(p.userId)
                    setShowKickModal(false)
                  }}
                  disabled={kickingPlayer}
                  className="w-full bg-gray-800 hover:bg-red-900/30 border border-gray-600 hover:border-red-600 rounded-lg px-4 py-3 text-left transition disabled:opacity-50"
                >
                  <span className="text-white font-medium">{p.displayName}</span>
                </button>
              ))}
            </div>

            <button
              onClick={() => setShowKickModal(false)}
              className="w-full mt-4 bg-gray-700 hover:bg-gray-600 text-white py-2 rounded-lg font-medium transition"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
