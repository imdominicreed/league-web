import { useState } from 'react'
import { RoleProfile, Role, LeagueRank, ALL_RANKS, ROLE_DISPLAY_NAMES } from '@/types'

interface RoleProfileEditorProps {
  profile: RoleProfile
  onUpdate: (role: Role, data: { leagueRank?: LeagueRank; mmr?: number; comfortRating?: number }) => void
  isUpdating: boolean
}

export function RoleProfileEditor({ profile, onUpdate, isUpdating }: RoleProfileEditorProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editedRank, setEditedRank] = useState<LeagueRank>(profile.leagueRank)
  const [editedComfort, setEditedComfort] = useState(profile.comfortRating)

  const handleSave = () => {
    onUpdate(profile.role, {
      leagueRank: editedRank,
      comfortRating: editedComfort,
    })
    setIsEditing(false)
  }

  const handleCancel = () => {
    setEditedRank(profile.leagueRank)
    setEditedComfort(profile.comfortRating)
    setIsEditing(false)
  }

  const renderStars = (rating: number, editable: boolean = false) => {
    return (
      <div className="flex gap-1">
        {[1, 2, 3, 4, 5].map((star) => (
          <button
            key={star}
            type="button"
            disabled={!editable}
            onClick={() => editable && setEditedComfort(star)}
            className={`text-2xl transition-colors ${
              star <= (editable ? editedComfort : rating)
                ? 'text-yellow-400'
                : 'text-gray-600'
            } ${editable ? 'cursor-pointer hover:text-yellow-300' : 'cursor-default'}`}
          >
            ★
          </button>
        ))}
      </div>
    )
  }

  const getRankColor = (rank: LeagueRank): string => {
    if (rank === 'Unranked') return 'text-gray-400'
    if (rank.startsWith('Iron')) return 'text-gray-500'
    if (rank.startsWith('Bronze')) return 'text-amber-700'
    if (rank.startsWith('Silver')) return 'text-gray-300'
    if (rank.startsWith('Gold')) return 'text-yellow-500'
    if (rank.startsWith('Platinum')) return 'text-teal-400'
    if (rank.startsWith('Emerald')) return 'text-emerald-500'
    if (rank.startsWith('Diamond')) return 'text-blue-400'
    if (rank === 'Master') return 'text-purple-500'
    if (rank === 'Grandmaster') return 'text-red-500'
    if (rank === 'Challenger') return 'text-cyan-400'
    return 'text-white'
  }

  return (
    <div className="bg-gray-800 rounded-lg p-4 border border-gray-700" data-testid={`role-profile-${profile.role}`}>
      <div className="flex items-center gap-3 mb-4">
        <div className="w-12 h-12 bg-gray-700 rounded-full flex items-center justify-center">
          <span className="text-2xl">{ROLE_DISPLAY_NAMES[profile.role][0]}</span>
        </div>
        <div>
          <h3 className="text-lg font-semibold text-white">{ROLE_DISPLAY_NAMES[profile.role]}</h3>
          <p className="text-sm text-gray-400">MMR: {profile.mmr}</p>
        </div>
      </div>

      {isEditing ? (
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Rank</label>
            <select
              value={editedRank}
              onChange={(e) => setEditedRank(e.target.value as LeagueRank)}
              className="w-full bg-gray-700 border border-gray-600 rounded px-3 py-2 text-white focus:outline-none focus:border-blue-500"
            >
              {ALL_RANKS.map((rank) => (
                <option key={rank} value={rank}>
                  {rank}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Comfort Level</label>
            {renderStars(editedComfort, true)}
          </div>

          <div className="flex gap-2">
            <button
              onClick={handleSave}
              disabled={isUpdating}
              className="flex-1 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 text-white py-2 px-4 rounded transition-colors"
            >
              {isUpdating ? 'Saving...' : 'Save'}
            </button>
            <button
              onClick={handleCancel}
              disabled={isUpdating}
              className="flex-1 bg-gray-600 hover:bg-gray-500 text-white py-2 px-4 rounded transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <span className="text-gray-400">Rank</span>
            <span className={`font-semibold ${getRankColor(profile.leagueRank)}`}>
              {profile.leagueRank}
            </span>
          </div>

          <div className="flex justify-between items-center">
            <span className="text-gray-400">Comfort</span>
            {renderStars(profile.comfortRating)}
          </div>

          <button
            onClick={() => setIsEditing(true)}
            className="w-full mt-2 bg-gray-700 hover:bg-gray-600 text-white py-2 px-4 rounded transition-colors"
          >
            Edit
          </button>
        </div>
      )}
    </div>
  )
}
