import { useState, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { RootState } from '@/store'
import { Champion } from '@/types'

interface Props {
  onSelect: (championId: string) => void
  onLockIn: () => void
  onHover: (championId: string | null) => void
  isYourTurn: boolean
  disabled: boolean
}

const ROLES = ['Fighter', 'Tank', 'Mage', 'Assassin', 'Marksman', 'Support']

// Get splash art URL for the center preview
function getSplashUrl(champion: Champion): string {
  return `https://ddragon.leagueoflegends.com/cdn/img/champion/splash/${champion.id}_0.jpg`
}

export default function ChampionGrid({ onSelect, onLockIn, onHover, isYourTurn, disabled }: Props) {
  const { championsList, champions } = useSelector((state: RootState) => state.champions)
  const draft = useSelector((state: RootState) => state.draft)
  const [selectedChampion, setSelectedChampion] = useState<string | null>(null)
  const [hoveredChampion, setHoveredChampion] = useState<string | null>(null)
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
      setHoveredChampion(championId)
      onHover(championId)
    }
  }

  const handleMouseLeave = () => {
    setHoveredChampion(null)
    if (!selectedChampion) {
      onHover(null)
    }
  }

  const handleLockIn = () => {
    if (selectedChampion) {
      onLockIn()
      setSelectedChampion(null)
      setHoveredChampion(null)
    }
  }

  const toggleRole = (role: string) => {
    setSelectedRoles(prev =>
      prev.includes(role)
        ? prev.filter(r => r !== role)
        : [...prev, role]
    )
  }

  // Champion to show in the preview area
  const previewChampion = selectedChampion || hoveredChampion
  const previewChampionData = previewChampion ? champions[previewChampion] : null

  return (
    <div className="flex-1 flex flex-col min-h-0 p-4">
      {/* Champion Preview Area */}
      <div className="relative h-48 mb-4 rounded-lg overflow-hidden bg-lol-dark-blue border border-lol-border">
        {previewChampionData ? (
          <>
            <img
              src={getSplashUrl(previewChampionData)}
              alt={previewChampionData.name}
              className="absolute inset-0 w-full h-full object-cover object-center"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/30 to-transparent" />
            <div className="absolute bottom-0 left-0 right-0 p-4">
              <h2 className="font-beaufort text-2xl text-lol-gold uppercase tracking-wider">
                {previewChampionData.name}
              </h2>
              <p className="text-lol-gold-light text-sm italic">
                {previewChampionData.title}
              </p>
            </div>
          </>
        ) : (
          <div className="absolute inset-0 flex items-center justify-center">
            <span className="font-beaufort text-lol-gray text-lg uppercase tracking-wider">
              {isYourTurn ? 'Select a Champion' : 'Waiting...'}
            </span>
          </div>
        )}
      </div>

      {/* Filters */}
      <div className="mb-3 space-y-2">
        {/* Search */}
        <input
          type="text"
          placeholder="Search champions..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full px-4 py-2 bg-lol-gray border border-lol-border rounded text-white placeholder-gray-500 focus:outline-none focus:border-lol-gold transition"
        />

        {/* Role filters */}
        <div className="flex gap-1 flex-wrap">
          {ROLES.map(role => (
            <button
              key={role}
              onClick={() => toggleRole(role)}
              className={`px-3 py-1 rounded text-xs uppercase tracking-wider transition border ${
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

      {/* Champion Grid */}
      <div className="flex-1 overflow-y-auto min-h-0 bg-lol-gray/30 rounded border border-lol-border p-2">
        <div className="grid grid-cols-8 gap-1">
          {filteredChampions.map(champion => {
            const isUsed = usedChampions.has(champion.id)
            const isSelected = selectedChampion === champion.id

            return (
              <button
                key={champion.id}
                onClick={() => handleChampionClick(champion.id)}
                onMouseEnter={() => handleMouseEnter(champion.id)}
                onMouseLeave={handleMouseLeave}
                disabled={isUsed || disabled || !isYourTurn}
                className={`relative aspect-square overflow-hidden transition-all duration-150 border ${
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
            className={`px-12 py-3 rounded font-beaufort font-bold uppercase tracking-wider text-lg transition-all ${
              selectedChampion
                ? 'bg-lol-gold text-lol-dark hover:brightness-110 shadow-[0_0_20px_rgba(200,170,110,0.3)]'
                : 'bg-lol-gray border border-lol-border text-gray-500 cursor-not-allowed'
            }`}
          >
            Lock In
          </button>
        </div>
      )}
    </div>
  )
}
