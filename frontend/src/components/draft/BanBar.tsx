import { useSelector } from 'react-redux'
import { RootState } from '@/store'

interface Props {
  blueBans: string[]
  redBans: string[]
  isBlueActive: boolean
  isRedActive: boolean
}

export default function BanBar({ blueBans, redBans, isBlueActive, isRedActive }: Props) {
  const { champions } = useSelector((state: RootState) => state.champions)
  const { timerRemainingMs, isComplete, currentTeam, actionType } = useSelector((state: RootState) => state.draft)
  const { room } = useSelector((state: RootState) => state.room)

  const isWaiting = !room || room.status === 'waiting'
  const isBanning = actionType === 'ban'

  const seconds = Math.ceil(timerRemainingMs / 1000)
  const isLow = seconds <= 10
  const isCritical = seconds <= 5

  const renderBanSlot = (championId: string | undefined, index: number, isActive: boolean, side: 'blue' | 'red') => {
    const champion = championId ? champions[championId] : null
    const isCurrentBan = isBanning && isActive && (side === 'blue' ? blueBans.length : redBans.length) === index

    return (
      <div key={index} className="flex flex-col items-center gap-1">
        <div
          className={`relative w-16 h-16 rounded-full overflow-hidden border-2 transition-all ${
            isCurrentBan
              ? 'border-lol-gold shadow-[0_0_8px_rgba(200,170,110,0.5)] animate-pulse'
              : champion
              ? 'border-lol-border'
              : 'border-lol-gray'
          } bg-lol-dark-blue`}
        >
          {champion ? (
            <>
              <img
                src={champion.imageUrl}
                alt={champion.name}
                className="w-full h-full object-cover grayscale opacity-60"
              />
              {/* Red X overlay for banned champions */}
              <div className="absolute inset-0 flex items-center justify-center">
                <svg className="w-10 h-10 text-red-team" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
                  <path d="M6 6L18 18M6 18L18 6" />
                </svg>
              </div>
            </>
          ) : null}
        </div>
        {champion && (
          <div className="text-xs text-lol-gold font-beaufort uppercase tracking-wider text-center truncate max-w-[80px]">
            {champion.name}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="bg-lol-dark-blue border-b border-lol-border px-6 py-4 pb-6">
      <div className="flex items-center justify-between max-w-5xl mx-auto">
        {/* Blue Side Bans */}
        <div className="flex items-center gap-3">
          <span className="text-blue-team font-beaufort text-sm uppercase tracking-wider mr-2">
            Bans
          </span>
          <div className="flex gap-3">
            {[0, 1, 2, 3, 4].map((i) => renderBanSlot(blueBans[i], i, isBlueActive, 'blue'))}
          </div>
        </div>

        {/* Timer Center */}
        <div className="text-center px-4">
          {isWaiting ? (
            <div className="font-beaufort text-lol-gold text-sm uppercase tracking-wider">
              Waiting
            </div>
          ) : isComplete ? (
            <div className="font-beaufort text-lol-gold text-lg uppercase tracking-wider">
              Complete
            </div>
          ) : (
            <>
              <div
                className={`font-beaufort text-3xl font-bold ${
                  isCritical ? 'text-red-team animate-pulse' : isLow ? 'text-yellow-500' : 'text-lol-gold'
                }`}
              >
                {seconds}
              </div>
              <div className={`text-xs uppercase tracking-wider ${
                currentTeam === 'blue' ? 'text-blue-team' : 'text-red-team'
              }`}>
                {currentTeam === 'blue' ? 'Blue' : 'Red'} {actionType === 'ban' ? 'Ban' : 'Pick'}
              </div>
            </>
          )}
        </div>

        {/* Red Side Bans */}
        <div className="flex items-center gap-3">
          <div className="flex gap-3">
            {[0, 1, 2, 3, 4].map((i) => renderBanSlot(redBans[i], i, isRedActive, 'red'))}
          </div>
          <span className="text-red-team font-beaufort text-sm uppercase tracking-wider ml-2">
            Bans
          </span>
        </div>
      </div>
    </div>
  )
}
