import { Link } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { RootState } from '@/store'

export default function Home() {
  const { isAuthenticated, user } = useSelector((state: RootState) => state.auth)

  return (
    <div className="min-h-screen flex flex-col items-center justify-center p-8">
      <h1 className="text-5xl font-bold text-lol-gold mb-4">League Draft</h1>
      <p className="text-xl text-gray-400 mb-12">
        Pro Play Pick/Ban Simulator
      </p>

      <div className="flex flex-col gap-4 w-full max-w-xs">
        {isAuthenticated ? (
          <>
            <p className="text-center text-gray-300 mb-4" data-testid="home-welcome-message">
              Welcome, <span className="text-lol-blue">{user?.displayName}</span>
            </p>
            <Link
              to="/create"
              className="bg-lol-blue text-black font-semibold py-3 px-6 rounded-lg text-center hover:bg-opacity-80 transition"
              data-testid="home-link-create-draft"
            >
              Create Draft Room
            </Link>
            <Link
              to="/join"
              className="bg-transparent border-2 border-lol-gold text-lol-gold font-semibold py-3 px-6 rounded-lg text-center hover:bg-lol-gold hover:text-black transition"
              data-testid="home-link-join-room"
            >
              Join Room
            </Link>
            <Link
              to="/profile"
              className="bg-gray-700 text-white font-semibold py-3 px-6 rounded-lg text-center hover:bg-gray-600 transition"
              data-testid="home-link-profile"
            >
              My Profile
            </Link>
            <Link
              to="/match-history"
              className="bg-gray-700 text-white font-semibold py-3 px-6 rounded-lg text-center hover:bg-gray-600 transition"
              data-testid="home-link-match-history"
            >
              Match History
            </Link>
            <Link
              to="/create-lobby"
              className="bg-purple-600 text-white font-semibold py-3 px-6 rounded-lg text-center hover:bg-purple-700 transition"
              data-testid="home-link-create-lobby"
            >
              Create 10-Man Lobby
            </Link>
            <Link
              to="/join-lobby"
              className="bg-transparent border-2 border-purple-500 text-purple-400 font-semibold py-3 px-6 rounded-lg text-center hover:bg-purple-500 hover:text-white transition"
              data-testid="home-link-join-lobby"
            >
              Join 10-Man Lobby
            </Link>
          </>
        ) : (
          <>
            <Link
              to="/login"
              className="bg-lol-blue text-black font-semibold py-3 px-6 rounded-lg text-center hover:bg-opacity-80 transition"
              data-testid="home-link-login"
            >
              Login
            </Link>
            <Link
              to="/register"
              className="bg-transparent border-2 border-lol-gold text-lol-gold font-semibold py-3 px-6 rounded-lg text-center hover:bg-lol-gold hover:text-black transition"
              data-testid="home-link-register"
            >
              Register
            </Link>
          </>
        )}
      </div>
    </div>
  )
}
