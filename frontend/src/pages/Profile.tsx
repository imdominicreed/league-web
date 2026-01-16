import { useEffect } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { Link } from 'react-router-dom'
import { RootState, AppDispatch } from '@/store'
import { fetchProfile, updateRoleProfile } from '@/store/slices/profileSlice'
import { RoleProfileEditor } from '@/components/profile/RoleProfileEditor'
import { Role, LeagueRank, ALL_ROLES } from '@/types'

export default function Profile() {
  const dispatch = useDispatch<AppDispatch>()
  const { user, roleProfiles, loading, error, updating } = useSelector(
    (state: RootState) => state.profile
  )
  const { isAuthenticated, user: authUser } = useSelector(
    (state: RootState) => state.auth
  )

  useEffect(() => {
    if (isAuthenticated) {
      dispatch(fetchProfile())
    }
  }, [dispatch, isAuthenticated])

  const handleUpdateProfile = (
    role: Role,
    data: { leagueRank?: LeagueRank; mmr?: number; comfortRating?: number }
  ) => {
    dispatch(updateRoleProfile({ role, data }))
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center p-8">
        <h1 className="text-3xl font-bold text-lol-gold mb-4">Profile</h1>
        <p className="text-gray-400 mb-6">Please log in to view your profile.</p>
        <Link
          to="/login"
          className="bg-lol-blue text-black font-semibold py-2 px-6 rounded-lg hover:bg-opacity-80 transition"
        >
          Login
        </Link>
      </div>
    )
  }

  if (loading && roleProfiles.length === 0) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Loading profile...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h1 className="text-3xl font-bold text-lol-gold">Profile</h1>
            <p className="text-gray-400">
              Welcome, {user?.displayName || authUser?.displayName}
            </p>
          </div>
          <Link
            to="/"
            className="text-gray-400 hover:text-white transition-colors"
          >
            &larr; Back to Home
          </Link>
        </div>

        {error && (
          <div className="bg-red-900/50 border border-red-500 text-red-200 px-4 py-3 rounded mb-6">
            {error}
          </div>
        )}

        <div className="mb-8">
          <h2 className="text-xl font-semibold text-white mb-4">Role Profiles</h2>
          <p className="text-gray-400 mb-6">
            Set your rank and comfort level for each role. This will be used for
            matchmaking in 10-man lobbies.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {ALL_ROLES.map((role) => {
              const profile = roleProfiles.find((p) => p.role === role)
              if (!profile) {
                return (
                  <div
                    key={role}
                    className="bg-gray-800 rounded-lg p-4 border border-gray-700 animate-pulse"
                  >
                    <div className="h-20 bg-gray-700 rounded"></div>
                  </div>
                )
              }
              return (
                <RoleProfileEditor
                  key={role}
                  profile={profile}
                  onUpdate={handleUpdateProfile}
                  isUpdating={updating === role}
                />
              )
            })}
          </div>
        </div>

        <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
          <h2 className="text-xl font-semibold text-white mb-4">
            How Matchmaking Works
          </h2>
          <ul className="text-gray-400 space-y-2">
            <li>
              <span className="text-lol-gold">Rank:</span> Your skill level for
              each role. Used to balance teams.
            </li>
            <li>
              <span className="text-lol-gold">Comfort (Stars):</span> How
              comfortable you are playing each role. Higher comfort means you're
              more likely to be assigned that role.
            </li>
            <li>
              <span className="text-lol-gold">MMR:</span> Calculated from your
              rank. Used internally for fair matchmaking.
            </li>
          </ul>
        </div>
      </div>
    </div>
  )
}
