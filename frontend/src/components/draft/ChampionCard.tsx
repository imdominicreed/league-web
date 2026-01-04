import { Champion } from '@/types'

interface Props {
  champion: Champion
  size?: 'small' | 'medium' | 'large'
  selecting?: boolean
}

export default function ChampionCard({ champion, size = 'medium', selecting = false }: Props) {
  return (
    <div className={`flex items-center gap-2 w-full ${selecting ? 'opacity-60' : ''}`}>
      <img
        src={champion.imageUrl}
        alt={champion.name}
        className={`object-cover rounded ${
          size === 'small' ? 'w-12 h-12' : size === 'medium' ? 'w-16 h-16' : 'w-20 h-20'
        }`}
      />
      <div className="flex-1 min-w-0">
        <div className={`font-semibold truncate ${size === 'small' ? 'text-sm' : ''}`}>
          {champion.name}
        </div>
        {size !== 'small' && (
          <div className="text-xs text-gray-400 truncate">
            {champion.tags.join(', ')}
          </div>
        )}
      </div>
    </div>
  )
}
