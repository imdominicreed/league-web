import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import TeamPanel from './TeamPanel'
import ChampionGrid from './ChampionGrid'
import BanBar from './BanBar'
import PauseControls from './PauseControls'
import EditConfirmModal from './EditConfirmModal'
import DraftTimer from './DraftTimer'

interface Props {
  ws: {
    isConnected: boolean
    selectChampion: (id: string) => void
    lockIn: () => void
    hoverChampion: (id: string | null) => void
    setReady: (ready: boolean) => void
    startDraft: () => void
    pauseDraft: () => void
    resumeDraft: () => void
    proposeEdit: (slotType: 'ban' | 'pick', team: 'blue' | 'red', slotIndex: number, championId: string) => void
    confirmEdit: () => void
    rejectEdit: () => void
  }
}

export default function DraftBoard({ ws }: Props) {
  const { room, players, isCaptain } = useSelector((state: RootState) => state.room)
  const draft = useSelector((state: RootState) => state.draft)
  const { champions } = useSelector((state: RootState) => state.champions)

  if (!room) return null

  const isWaiting = room.status === 'waiting'
  const isYourTurn = isCaptain && draft.yourSide === draft.currentTeam
  const isDraftActive = room.status === 'in_progress' && !draft.isComplete

  // Create a Map for EditConfirmModal
  const championsMap = new Map(
    Object.entries(champions).map(([id, champ]) => [id, { name: champ.name, imageUrl: champ.imageUrl }])
  )

  return (
    <div className="h-screen flex flex-col bg-lol-dark overflow-hidden">
      {/* Top Bar with Room Code, Pause Controls, and Connection Status */}
      <header className="bg-lol-dark-blue/80 border-b border-lol-border px-3 py-1 flex items-center justify-between">
        <div className="text-xs text-lol-gold-light">
          Room: <span className="font-mono text-lol-gold">{room.shortCode}</span>
        </div>

        {/* Center: Timer and Pause Controls */}
        <div className="flex items-center gap-4">
          {isDraftActive && (
            <>
              <DraftTimer />
              <PauseControls
                onPause={ws.pauseDraft}
                onResume={ws.resumeDraft}
              />
            </>
          )}
        </div>

        <div className="flex items-center gap-2">
          {!ws.isConnected && (
            <span className="text-red-team text-xs">Disconnected</span>
          )}
        </div>
      </header>

      {/* Edit Confirm Modal */}
      <EditConfirmModal
        onConfirm={ws.confirmEdit}
        onReject={ws.rejectEdit}
        champions={championsMap}
      />

      {/* Ban Bar with Timer */}
      <BanBar
        blueBans={draft.blueBans}
        redBans={draft.redBans}
        isBlueActive={draft.currentTeam === 'blue'}
        isRedActive={draft.currentTeam === 'red'}
      />

      {/* Main Content */}
      <main className="flex-1 flex min-h-0 px-4">
        {/* Blue Side Picks */}
        <TeamPanel
          side="blue"
          player={players.blue}
          picks={draft.bluePicks}
          isActive={draft.currentTeam === 'blue'}
          hoveredChampion={draft.hoveredChampion.blue}
        />

        {/* Center - Champion Grid or Waiting State */}
        <div className="flex-1 flex flex-col min-h-0 bg-lol-dark">
          {isWaiting ? (
            <div className="flex-1 flex flex-col items-center justify-center">
              <h2 className="font-beaufort text-lg text-lol-gold-light mb-6 uppercase tracking-wider">
                Waiting for Players
              </h2>

              <div className="flex gap-8 mb-6">
                <div className="text-center">
                  <div className="font-beaufort text-blue-team text-sm uppercase tracking-wider mb-2">
                    Blue Side
                  </div>
                  <div className={`px-4 py-2 rounded border-2 text-sm ${
                    players.blue
                      ? 'border-blue-team bg-blue-team/10 text-blue-team'
                      : 'border-lol-gray bg-lol-gray/20 text-gray-500'
                  }`}>
                    {players.blue?.displayName || 'Waiting...'}
                    {players.blue?.ready && (
                      <span className="ml-2 text-green-500">✓</span>
                    )}
                  </div>
                </div>
                <div className="text-center">
                  <div className="font-beaufort text-red-team text-sm uppercase tracking-wider mb-2">
                    Red Side
                  </div>
                  <div className={`px-4 py-2 rounded border-2 text-sm ${
                    players.red
                      ? 'border-red-team bg-red-team/10 text-red-team'
                      : 'border-lol-gray bg-lol-gray/20 text-gray-500'
                  }`}>
                    {players.red?.displayName || 'Waiting...'}
                    {players.red?.ready && (
                      <span className="ml-2 text-green-500">✓</span>
                    )}
                  </div>
                </div>
              </div>

              {isCaptain && (() => {
                const myPlayer = draft.yourSide === 'blue' ? players.blue : players.red;
                const isMyReady = myPlayer?.ready ?? false;
                return isMyReady ? (
                  <div className="bg-green-600 text-white font-beaufort font-bold py-2 px-6 rounded text-sm uppercase tracking-wider flex items-center gap-2">
                    <span>Ready</span>
                    <span className="text-lg">✓</span>
                  </div>
                ) : (
                  <button
                    onClick={() => ws.setReady(true)}
                    className="bg-lol-blue-accent text-lol-dark font-beaufort font-bold py-2 px-6 rounded text-sm uppercase tracking-wider hover:brightness-110 transition"
                  >
                    Ready
                  </button>
                );
              })()}

              {isCaptain && players.blue?.ready && players.red?.ready && (
                <button
                  onClick={() => ws.startDraft()}
                  className="mt-3 bg-lol-gold text-lol-dark font-beaufort font-bold py-2 px-6 rounded text-sm uppercase tracking-wider hover:brightness-110 transition"
                >
                  Start Draft
                </button>
              )}
            </div>
          ) : (
            <ChampionGrid
              onSelect={ws.selectChampion}
              onLockIn={ws.lockIn}
              onHover={ws.hoverChampion}
              onProposeEdit={ws.proposeEdit}
              isYourTurn={isYourTurn}
              disabled={draft.isComplete}
            />
          )}
        </div>

        {/* Red Side Picks */}
        <TeamPanel
          side="red"
          player={players.red}
          picks={draft.redPicks}
          isActive={draft.currentTeam === 'red'}
          hoveredChampion={draft.hoveredChampion.red}
        />
      </main>
    </div>
  )
}
