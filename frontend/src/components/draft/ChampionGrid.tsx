import { useState, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { RootState } from '@/store'

interface Props {
  onSelect: (championId: string) => void
  onLockIn: () => void
  onHover: (championId: string | null) => void
  isYourTurn: boolean
  disabled: boolean
}

const ROLES = ['Fighter', 'Tank', 'Mage', 'Assassin', 'Marksman', 'Support']

export default function ChampionGrid({ onSelect, onLockIn, onHover, isYourTurn, disabled }: Props) {
  const { championsList, filters } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)
  const [selectedChampion, setSelectedChampion] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [selectedRoles, setSelectedRoles] = useState<string[]>([])

  const usedChampions = useMemo(() => {
    const used = new Set<string>()
    draft.blueBans.forEach(id => used.add(id))
    draft.redBans.forEach(id => used.add(id))
    draft.bluePicks.forEach(id => used.add(id))
    draft.redPicks.forEach(id => used.add(id))
    draft.fearlessBans.forEach(id => used.add(id))
    return used
  }, [draft.blueBans, draft.redBans, draft.bluePicks, draft.redPicks, draft.fearlessBans])

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
    if (disabled || !isYourTurn || usedChampions.has(championId)) return

    setSelectedChampion(championId)
    onSelect(championId)
    onHover(championId)
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

  return (
    <div className="flex-1 flex flex-col px-4 pb-4">
      {/* Filters */}
      <div className="mb-4 space-y-3">
        {/* Search */}
        <input
          type="text"
          placeholder="Search champions..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-lol-blue"
        />

        {/* Role filters */}
        <div className="flex gap-2 flex-wrap">
          {ROLES.map(role => (
            <button
              key={role}
              onClick={() => toggleRole(role)}
              className={`px-3 py-1 rounded text-sm transition ${
                selectedRoles.includes(role)
                  ? 'bg-lol-blue text-black'
                  : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
              }`}
            >
              {role}
            </button>
          ))}
        </div>
      </div>

      {/* Champion Grid */}
      <div className="flex-1 overflow-y-auto">
        <div className="grid grid-cols-8 gap-2">
          {filteredChampions.map(champion => {
            const isUsed = usedChampions.has(champion.id)
            const isSelected = selectedChampion === champion.id

            return (
              <button
                key={champion.id}
                onClick={() => handleChampionClick(champion.id)}
                onMouseEnter={() => isYourTurn && !isUsed && onHover(champion.id)}
                onMouseLeave={() => onHover(null)}
                disabled={isUsed || disabled || !isYourTurn}
                className={`relative aspect-square rounded overflow-hidden transition ${
                  isUsed
                    ? 'opacity-30 grayscale cursor-not-allowed'
                    : isSelected
                    ? 'ring-2 ring-lol-gold scale-105'
                    : 'hover:ring-2 hover:ring-white/50 hover:scale-105'
                } ${!isYourTurn ? 'cursor-not-allowed' : ''}`}
              >
                <img
                  src={champion.imageUrl}
                  alt={champion.name}
                  className="w-full h-full object-cover"
                />
                <div className="absolute bottom-0 left-0 right-0 bg-black/70 text-xs text-center py-0.5 truncate">
                  {champion.name}
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Lock In Button */}
      {isYourTurn && !disabled && (
        <div className="mt-4 flex justify-center">
          <button
            onClick={handleLockIn}
            disabled={!selectedChampion}
            className={`px-8 py-3 rounded-lg font-semibold transition ${
              selectedChampion
                ? 'bg-lol-gold text-black hover:bg-opacity-80'
                : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }`}
          >
            Lock In
          </button>
        </div>
      )}
    </div>
  )
}
