import { useState } from 'react'
import { LobbyPlayer, Side } from '@/types'

interface CaptainControlsProps {
  players: LobbyPlayer[]
  currentUserId: string
  currentUserSide: Side
  isCaptain: boolean
  hasTeams: boolean
  isMatchmaking: boolean
  hasPendingAction: boolean
  swapMode: boolean
  onTakeCaptain: () => void
  onPromoteCaptain: (userId: string) => void
  onKickPlayer: (userId: string) => void
  onToggleSwapMode: () => void
  onProposeMatchmake: () => void
  onProposeStartDraft: () => void
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
  isMatchmaking,
  hasPendingAction,
  swapMode,
  onTakeCaptain,
  onPromoteCaptain,
  onKickPlayer,
  onToggleSwapMode,
  onProposeMatchmake,
  onProposeStartDraft,
  takingCaptain,
  promotingCaptain,
  kickingPlayer,
  proposingAction,
}: CaptainControlsProps) {
  const [showPromoteModal, setShowPromoteModal] = useState(false)
  const [showKickModal, setShowKickModal] = useState(false)

  const teamPlayers = players.filter(p => p.team === currentUserSide && p.userId !== currentUserId)

  const allReady = hasTeams && players.length === 10 && players.every(p => p.isReady)

  return (
    <div className="space-y-4">
      {/* Captain Controls */}
      <div className="bg-gray-800/50 rounded-lg p-4 border border-gray-700">
        <div className="flex items-center justify-between mb-3">
          <h4 className="text-sm font-semibold text-gray-400 uppercase tracking-wide">
            {isCaptain ? 'Captain Controls' : 'Player Actions'}
          </h4>
          {isCaptain && (
            <span className="text-lol-gold text-xs font-medium px-2 py-1 bg-lol-gold/10 rounded" data-testid="captain-badge">
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
              data-testid="captain-button-take"
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
                data-testid="captain-button-promote"
              >
                Promote Captain
              </button>

              {/* Kick Player */}
              <button
                onClick={() => setShowKickModal(true)}
                disabled={kickingPlayer || teamPlayers.length === 0}
                className="bg-red-600/20 hover:bg-red-600/30 text-red-400 px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
                data-testid="captain-button-kick"
              >
                Kick Player
              </button>

              {/* Swap */}
              <button
                onClick={onToggleSwapMode}
                disabled={!swapMode && (proposingAction || hasPendingAction || players.length < 2)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition ${
                  swapMode
                    ? 'bg-lol-gold text-black'
                    : 'bg-gray-700 hover:bg-gray-600 text-white disabled:opacity-50'
                }`}
                data-testid="captain-button-swap"
              >
                {swapMode ? 'Cancel Swap' : 'Swap'}
              </button>

              {/* Matchmake - only in waiting_for_players state */}
              {!hasTeams && !isMatchmaking && (
                <button
                  onClick={onProposeMatchmake}
                  disabled={proposingAction || hasPendingAction || players.length < 10}
                  className="bg-lol-blue/20 hover:bg-lol-blue/30 text-lol-blue px-4 py-2 rounded-lg text-sm font-medium transition disabled:opacity-50"
                  data-testid="captain-button-matchmake"
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
                  data-testid="captain-button-start-draft"
                >
                  {proposingAction ? 'Proposing...' : 'Propose Start Draft'}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Promote Modal */}
      {showPromoteModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50" data-testid="captain-modal-promote">
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
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50" data-testid="captain-modal-kick">
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
