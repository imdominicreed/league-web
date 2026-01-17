import { useSelector, useDispatch } from 'react-redux'
import { RootState } from '@/store'
import { setEditingSlot } from '@/store/slices/draftSlice'
import { Player, Champion, TeamPlayer, Role, ALL_ROLES, ROLE_ABBREVIATIONS } from '@/types'

interface Props {
  side: 'blue' | 'red'
  player: Player | null
  picks: string[]
  isActive: boolean
  hoveredChampion: string | null
}

// Get splash art URL for a champion
function getSplashUrl(champion: Champion): string {
  return `https://ddragon.leagueoflegends.com/cdn/img/champion/loading/${champion.id}_0.jpg`
}

// Role order for pick slots: top=0, jungle=1, mid=2, adc=3, support=4
const ROLE_TO_SLOT_INDEX: Record<Role, number> = {
  top: 0,
  jungle: 1,
  mid: 2,
  adc: 3,
  support: 4,
}

export default function TeamPanel({ side, player, picks, isActive, hoveredChampion }: Props) {
  const dispatch = useDispatch()
  const { champions } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)
  const { isCaptain } = useSelector((state: RootState) => state.room)

  const teamColor = side === 'blue' ? 'blue-team' : 'red-team'
  const borderColor = side === 'blue' ? 'border-blue-team' : 'border-red-team'

  // Check if slots are editable (paused and user is captain)
  const canEdit = draft.isPaused && isCaptain

  // For current pick, show hovered champion
  const currentActionChampion = isActive && hoveredChampion ? hoveredChampion : null

  // Filter team players for this side
  const teamPlayers = draft.teamPlayers.filter((p: TeamPlayer) => p.team === side)
  const isTeamDraft = draft.isTeamDraft && teamPlayers.length > 0

  // Get captain for this team
  const captain = teamPlayers.find((p: TeamPlayer) => p.isCaptain)

  // Create a map of role to player for team draft mode
  const playersByRole: Record<Role, TeamPlayer | undefined> = {} as Record<Role, TeamPlayer | undefined>
  teamPlayers.forEach((p: TeamPlayer) => {
    playersByRole[p.assignedRole] = p
  })

  // Handle click on a pick slot for editing
  const handleSlotClick = (slotIndex: number) => {
    if (!canEdit || !picks[slotIndex]) return
    dispatch(setEditingSlot({
      slotType: 'pick',
      team: side,
      slotIndex,
    }))
  }

  // Render a single pick slot
  const renderPickSlot = (
    slotIndex: number,
    championId: string | undefined,
    role?: Role,
    teamPlayer?: TeamPlayer
  ) => {
    const champion = championId ? champions[championId] : null
    const isCurrentPick = draft.actionType === 'pick' && isActive && slotIndex === picks.length
    const showHovered = isCurrentPick && currentActionChampion && champions[currentActionChampion]

    // In team draft mode, highlight captain's row when it's their team's turn
    const isCaptainRow = isTeamDraft && teamPlayer?.isCaptain && isActive

    // Check if this slot is being edited
    const isEditing = draft.editingSlot?.slotType === 'pick' &&
      draft.editingSlot?.team === side &&
      draft.editingSlot?.slotIndex === slotIndex

    // Slot is clickable if paused, user is captain, and slot has a champion
    const isClickable = canEdit && champion

    return (
      <div
        key={slotIndex}
        onClick={() => isClickable && handleSlotClick(slotIndex)}
        className={`flex-1 relative overflow-hidden border-b border-lol-border last:border-b-0 ${
          isCurrentPick ? 'ring-2 ring-inset ring-lol-gold animate-pulse' : ''
        } ${isCaptainRow ? 'ring-1 ring-inset ring-lol-gold/50' : ''} ${
          isEditing ? 'ring-2 ring-inset ring-green-500' : ''
        } ${isClickable ? 'cursor-pointer hover:brightness-110' : ''}`}
        data-testid={`draft-pick-slot-${side}-${slotIndex}`}
      >
        {/* Role and Player Info overlay */}
        {role && (
          <div className="absolute top-0 left-0 right-0 z-10 flex items-center gap-1 px-1.5 py-0.5 bg-black/60">
            <span className={`text-xs font-bold ${side === 'blue' ? 'text-blue-team' : 'text-red-team'}`}>
              {ROLE_ABBREVIATIONS[role]}
            </span>
            {teamPlayer && (
              <>
                <span className="text-lol-text-secondary text-xs truncate flex-1">
                  {teamPlayer.displayName}
                </span>
                {teamPlayer.isCaptain && (
                  <span className="text-lol-gold text-xs font-bold" title="Captain">
                    C
                  </span>
                )}
              </>
            )}
          </div>
        )}

        {champion ? (
          // Picked champion - show splash art
          <>
            <img
              src={getSplashUrl(champion)}
              alt={champion.name}
              className="absolute inset-0 w-full h-full object-cover object-top"
            />
            {/* Gradient overlay for text readability */}
            <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent" />
            {/* Edit icon overlay when slot is clickable */}
            {isClickable && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 hover:opacity-100 transition-opacity">
                <span className="text-white text-2xl">✏️</span>
              </div>
            )}
            {/* Champion name */}
            <div className="absolute bottom-0 left-0 right-0 p-1">
              <div className="text-lol-gold text-sm font-beaufort uppercase tracking-wider font-semibold truncate">
                {champion.name}
              </div>
            </div>
          </>
        ) : showHovered ? (
          // Currently hovering - show preview
          <>
            <img
              src={getSplashUrl(champions[currentActionChampion])}
              alt="Selecting..."
              className="absolute inset-0 w-full h-full object-cover object-top opacity-60"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent" />
            <div className="absolute bottom-0 left-0 right-0 p-1">
              <div className="text-lol-gold text-sm font-beaufort uppercase tracking-wider font-semibold truncate">
                {champions[currentActionChampion].name}
              </div>
            </div>
          </>
        ) : (
          // Empty slot
          <div className="absolute inset-0 bg-lol-gray/30" />
        )}
      </div>
    )
  }

  return (
    <div className={`w-72 bg-lol-dark-blue flex flex-col border-l border-r border-lol-border pb-4 ${
      isActive ? borderColor : ''
    }`} data-testid={`draft-team-panel-${side}`}>
      {/* Team Header */}
      <div className={`px-2 py-1.5 border-b border-lol-border bg-${teamColor}/10`}>
        <div className={`font-beaufort text-${teamColor} text-xs uppercase tracking-wider`}>
          {side === 'blue' ? 'Blue' : 'Red'}
        </div>
        <div className="text-lol-gold-light text-xs truncate">
          {isTeamDraft ? (
            captain ? `${captain.displayName} (Captain)` : 'Team'
          ) : (
            player?.displayName || 'Waiting...'
          )}
        </div>
      </div>

      {/* Picks - Large Splash Art Slots */}
      <div className="flex-1 flex flex-col">
        {isTeamDraft ? (
          // Team draft mode: Show 5 rows with roles
          ALL_ROLES.map((role) => {
            const slotIndex = ROLE_TO_SLOT_INDEX[role]
            const championId = picks[slotIndex]
            const teamPlayer = playersByRole[role]
            return renderPickSlot(slotIndex, championId, role, teamPlayer)
          })
        ) : (
          // 1v1 mode: Show 5 pick slots with role indicators
          ALL_ROLES.map((role) => {
            const slotIndex = ROLE_TO_SLOT_INDEX[role]
            const championId = picks[slotIndex]
            return renderPickSlot(slotIndex, championId, role)
          })
        )}
      </div>
    </div>
  )
}
