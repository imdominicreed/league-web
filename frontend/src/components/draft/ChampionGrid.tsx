import { useState, useMemo } from 'react'
import { useSelector, useDispatch } from 'react-redux'
import { RootState } from '@/store'
import { clearEditingSlot } from '@/store/slices/draftSlice'

interface Props {
  onSelect: (championId: string) => void
  onLockIn: () => void
  onHover: (championId: string | null) => void
  onProposeEdit?: (slotType: 'ban' | 'pick', team: 'blue' | 'red', slotIndex: number, championId: string) => void
  isYourTurn: boolean
  disabled: boolean
}

const ROLES = ['Fighter', 'Tank', 'Mage', 'Assassin', 'Marksman', 'Support']

export default function ChampionGrid({ onSelect, onLockIn, onHover, onProposeEdit, isYourTurn, disabled }: Props) {
  const dispatch = useDispatch()
  const { championsList, champions } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)
  const [selectedChampion, setSelectedChampion] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])

  // Edit mode state
  const isEditMode = draft.isPaused && draft.editingSlot !== null

  // Get the champion currently in the editing slot (if editing)
  const editingSlotChampion = useMemo(() => {
    if (!draft.editingSlot) return null
    const { slotType, team, slotIndex } = draft.editingSlot
    if (slotType === 'ban') {
      return team === 'blue' ? draft.blueBans[slotIndex] : draft.redBans[slotIndex]
    } else {
      return team === 'blue' ? draft.bluePicks[slotIndex] : draft.redPicks[slotIndex]
    }
  }, [draft.editingSlot, draft.blueBans, draft.redBans, draft.bluePicks, draft.redPicks])

  const usedChampions = useMemo(() => {
    const used = new Set<string>()
    draft.blueBans.forEach(id => used.add(id))
    draft.redBans.forEach(id => used.add(id))
    draft.bluePicks.forEach(id => used.add(id))
    draft.redPicks.forEach(id => used.add(id))
    draft.fearlessBans.forEach(id => used.add(id))
    // In edit mode, exclude the champion currently in the slot being edited
    // so it shows as available for re-selection
    if (editingSlotChampion) {
      used.delete(editingSlotChampion)
    }
    return used
  }, [draft.blueBans, draft.redBans, draft.bluePicks, draft.redPicks, draft.fearlessBans, editingSlotChampion])

  const filteredChampions = useMemo(() => {
    return championsList.filter(champ => {
      if (searchTerm && !champ.name.toLowerCase().includes(searchTerm.toLowerCase())) {
        return false
      }
      if (selectedRoles.length > 0 && !selectedRoles.some(role => champ.tags.includes(role))) {
        return false
      }
      return true
    })
  }, [championsList, searchTerm, selectedRoles])

  const handleChampionClick = (championId: string) => {
    // Edit mode: propose the edit and clear editing slot
    if (isEditMode && draft.editingSlot && onProposeEdit) {
      if (usedChampions.has(championId)) return
      const { slotType, team, slotIndex } = draft.editingSlot
      onProposeEdit(slotType, team, slotIndex, championId)
      dispatch(clearEditingSlot())
      return
    }

    // Normal mode
    if (disabled || !isYourTurn || usedChampions.has(championId)) return

    setSelectedChampion(championId)
    onSelect(championId)
    onHover(championId)
  }

  // Cancel edit mode
  const handleCancelEdit = () => {
    dispatch(clearEditingSlot())
  }

  const handleMouseEnter = (championId: string) => {
    if (isYourTurn && !usedChampions.has(championId) && !selectedChampion) {
      onHover(championId)
    }
  }

  const handleMouseLeave = () => {
    if (!selectedChampion) {
      onHover(null)
    }
  }

  const handleLockIn = () => {
    if (selectedChampion) {
      onLockIn()
      setSelectedChampion(null)
    }
  }

  const toggleRole = (role: string) => {
    setSelectedRoles(prev =>
      prev.includes(role)
        ? prev.filter(r => r !== role)
        : [...prev, role]
    )
  }

  const isBanning = draft.actionType === 'ban'
  const isPicking = draft.actionType === 'pick'

  return (
    <div className="flex-1 flex flex-col min-h-0 p-6" data-testid="champion-grid">

      {/* Edit Mode Banner */}
      {isEditMode && draft.editingSlot && (
        <div className="mb-4 rounded-lg border-2 p-3 text-center bg-green-900/30 border-green-700">
          <div className="font-beaufort text-xl uppercase tracking-widest font-bold text-green-400">
            ✏️ Edit Mode
          </div>
          <div className="mt-1 text-sm font-semibold text-lol-gold">
            Select replacement for {draft.editingSlot.team === 'blue' ? 'Blue' : 'Red'}'s{' '}
            {draft.editingSlot.slotType === 'ban' ? 'Ban' : 'Pick'} #{draft.editingSlot.slotIndex + 1}
            {editingSlotChampion && champions[editingSlotChampion] && (
              <span className="text-gray-400"> (currently {champions[editingSlotChampion].name})</span>
            )}
          </div>
          <button
            onClick={handleCancelEdit}
            className="mt-2 px-4 py-1 bg-gray-600 hover:bg-gray-500 text-white rounded text-sm transition-colors"
          >
            Cancel Edit
          </button>
        </div>
      )}

      {/* Phase & Turn Indicator Banner (hidden during edit mode) */}
      {!isEditMode && (isBanning || isPicking) && (
        <div className={`mb-4 rounded-lg border-2 p-3 text-center ${
          isBanning
            ? 'bg-red-900/30 border-red-700'
            : 'bg-blue-900/30 border-blue-700'
        }`}>
          <div className={`font-beaufort text-xl uppercase tracking-widest font-bold ${
            isBanning ? 'text-red-400' : 'text-blue-400'
          }`}>
            {isBanning ? '🚫 Banning Phase' : '⚔️ Picking Phase'}
          </div>
          <div className={`mt-1 text-sm font-semibold ${
            isYourTurn
              ? 'text-lol-gold'
              : 'text-gray-400'
          }`}>
            {isYourTurn
              ? `Your turn to ${isBanning ? 'BAN' : 'PICK'}`
              : `Waiting for ${draft.currentTeam === 'blue' ? 'Blue' : 'Red'} team...`
            }
          </div>
        </div>
      )}

      {/* Filters - Compact Row */}
      <div className="mb-2 flex gap-2 items-center">
        {/* Search */}
        <input
          type="text"
          placeholder="Search..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-48 px-3 py-1.5 bg-lol-gray border border-lol-border rounded text-sm text-white placeholder-gray-500 focus:outline-none focus:border-lol-gold transition"
          data-testid="draft-champion-search"
        />

        {/* Role filters */}
        <div className="flex gap-1 flex-wrap flex-1">
          {ROLES.map(role => (
            <button
              key={role}
              onClick={() => toggleRole(role)}
              className={`px-2 py-1 rounded text-xs uppercase tracking-wider transition border ${
                selectedRoles.includes(role)
                  ? 'bg-lol-gold/20 border-lol-gold text-lol-gold'
                  : 'bg-lol-gray border-lol-border text-gray-400 hover:border-lol-gold-dark hover:text-gray-300'
              }`}
            >
              {role}
            </button>
          ))}
        </div>
      </div>

      {/* Champion Grid - Smaller icons */}
      <div className="flex-1 overflow-y-auto min-h-0 bg-lol-gray/30 rounded border border-lol-border p-3">
        <div className="grid grid-cols-8 gap-y-8 gap-x-3" data-testid="champion-grid">
          {filteredChampions.map(champion => {
            const isUsed = usedChampions.has(champion.id)
            const isSelected = selectedChampion === champion.id
            // In edit mode, enable all non-used champions
            const isDisabledNormal = isUsed || disabled || !isYourTurn
            const isDisabledEdit = isUsed
            const isDisabled = isEditMode ? isDisabledEdit : isDisabledNormal

            return (
              <div key={champion.id} className="flex flex-col items-center gap-1">
                <button
                  onClick={() => handleChampionClick(champion.id)}
                  onMouseEnter={() => handleMouseEnter(champion.id)}
                  onMouseLeave={handleMouseLeave}
                  disabled={isDisabled}
                  className={`relative w-48 h-48 overflow-hidden transition-all duration-150 border ${
                    isUsed
                      ? 'opacity-40 grayscale cursor-not-allowed border-transparent'
                      : isSelected
                      ? 'border-lol-gold shadow-[0_0_10px_rgba(200,170,110,0.5)] scale-105 z-10'
                      : isEditMode
                      ? 'border-green-600 hover:border-green-400 hover:scale-105'
                      : 'border-lol-border hover:border-lol-gold-dark hover:scale-105'
                  } ${!isEditMode && !isYourTurn && !isUsed ? 'cursor-not-allowed opacity-70' : ''}`}
                >
                  <img
                    src={champion.imageUrl}
                    alt={champion.name}
                    className="w-full h-full object-cover"
                    loading="lazy"
                  />
                </button>
                <div className="text-sm text-lol-gold font-beaufort uppercase tracking-wider text-center truncate w-48">
                  {champion.name}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Lock In Button */}
      <div className="mt-8 mb-4 flex justify-center">
        <button
          onClick={handleLockIn}
          disabled={!selectedChampion || !isYourTurn || disabled}
          className={`px-8 py-2 rounded font-beaufort font-bold uppercase tracking-wider text-sm transition-all ${
            selectedChampion && isYourTurn && !disabled
              ? 'bg-lol-gold text-lol-dark hover:brightness-110 shadow-[0_0_15px_rgba(200,170,110,0.3)]'
              : 'bg-lol-gray border border-lol-border text-gray-500 cursor-not-allowed'
          }`}
          data-testid="draft-button-lock-in"
        >
          Lock In
        </button>
      </div>
    </div>
  )
}
