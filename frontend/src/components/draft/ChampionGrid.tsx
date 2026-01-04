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
  const { championsList } = useSelector((state: RootState) => state.champions)
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

  const handleMouseEnter = (championId: string) => {
    if (isYourTurn && !usedChampions.has(championId)) {
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

  return (
    <div className="flex-1 flex flex-col min-h-0 p-6">

      {/* Filters - Compact Row */}
      <div className="mb-2 flex gap-2 items-center">
        {/* Search */}
        <input
          type="text"
          placeholder="Search..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-48 px-3 py-1.5 bg-lol-gray border border-lol-border rounded text-sm text-white placeholder-gray-500 focus:outline-none focus:border-lol-gold transition"
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
        <div className="grid grid-cols-8 gap-y-8 gap-x-3">
          {filteredChampions.map(champion => {
            const isUsed = usedChampions.has(champion.id)
            const isSelected = selectedChampion === champion.id

            return (
              <div key={champion.id} className="flex flex-col items-center gap-1">
                <button
                  onClick={() => handleChampionClick(champion.id)}
                  onMouseEnter={() => handleMouseEnter(champion.id)}
                  onMouseLeave={handleMouseLeave}
                  disabled={isUsed || disabled || !isYourTurn}
                  className={`relative w-48 h-48 overflow-hidden transition-all duration-150 border ${
                    isUsed
                      ? 'opacity-40 grayscale cursor-not-allowed border-transparent'
                      : isSelected
                      ? 'border-lol-gold shadow-[0_0_10px_rgba(200,170,110,0.5)] scale-105 z-10'
                      : 'border-lol-border hover:border-lol-gold-dark hover:scale-105'
                  } ${!isYourTurn && !isUsed ? 'cursor-not-allowed opacity-70' : ''}`}
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
        >
          Lock In
        </button>
      </div>
    </div>
  )
}
