import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import TeamPanel from './TeamPanel'
import ChampionGrid from './ChampionGrid'
import DraftTimer from './DraftTimer'
import PhaseIndicator from './PhaseIndicator'

interface Props {
  ws: {
    isConnected: boolean
    selectChampion: (id: string) => void
    lockIn: () => void
    hoverChampion: (id: string | null) => void
    setReady: (ready: boolean) => void
    startDraft: () => void
  }
}

export default function DraftBoard({ ws }: Props) {
  const { room, players } = useSelector((state: RootState) => state.room)
  const draft = useSelector((state: RootState) => state.draft)

  if (!room) return null

  const isWaiting = room.status === 'waiting'
  const isYourTurn = draft.yourSide === draft.currentTeam

  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="bg-gray-900 border-b border-gray-800 px-4 py-3">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="text-sm text-gray-400">
            Room: <span className="text-lol-gold font-mono">{room.shortCode}</span>
          </div>
          <PhaseIndicator />
          <div className="flex items-center gap-2">
            {!ws.isConnected && (
              <span className="text-red-500 text-sm">Disconnected</span>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 flex">
        {/* Blue Side */}
        <TeamPanel
          side="blue"
          player={players.blue}
          bans={draft.blueBans}
          picks={draft.bluePicks}
          isActive={draft.currentTeam === 'blue'}
          hoveredChampion={draft.hoveredChampion.blue}
        />

        {/* Center - Champion Grid */}
        <div className="flex-1 flex flex-col">
          {/* Timer */}
          <div className="py-4">
            <DraftTimer />
          </div>

          {isWaiting ? (
            <div className="flex-1 flex flex-col items-center justify-center">
              <h2 className="text-2xl text-gray-400 mb-8">Waiting for players...</h2>

              <div className="flex gap-8 mb-8">
                <div className="text-center">
                  <div className="text-blue-side font-semibold mb-2">Blue Side</div>
                  <div className={`px-4 py-2 rounded ${players.blue ? 'bg-blue-side/20 text-blue-side' : 'bg-gray-800 text-gray-500'}`}>
                    {players.blue?.displayName || 'Waiting...'}
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-red-side font-semibold mb-2">Red Side</div>
                  <div className={`px-4 py-2 rounded ${players.red ? 'bg-red-side/20 text-red-side' : 'bg-gray-800 text-gray-500'}`}>
                    {players.red?.displayName || 'Waiting...'}
                  </div>
                </div>
              </div>

              {draft.yourSide !== 'spectator' && (
                <button
                  onClick={() => ws.setReady(true)}
                  className="bg-lol-blue text-black font-semibold py-2 px-6 rounded-lg hover:bg-opacity-80 transition"
                >
                  Ready
                </button>
              )}

              {players.blue?.ready && players.red?.ready && (
                <button
                  onClick={() => ws.startDraft()}
                  className="mt-4 bg-lol-gold text-black font-semibold py-2 px-6 rounded-lg hover:bg-opacity-80 transition"
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
              isYourTurn={isYourTurn}
              disabled={draft.isComplete}
            />
          )}
        </div>

        {/* Red Side */}
        <TeamPanel
          side="red"
          player={players.red}
          bans={draft.redBans}
          picks={draft.redPicks}
          isActive={draft.currentTeam === 'red'}
          hoveredChampion={draft.hoveredChampion.red}
        />
      </main>
    </div>
  )
}
