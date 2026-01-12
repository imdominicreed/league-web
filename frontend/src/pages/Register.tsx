import { useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useNavigate, Link } from 'react-router-dom'
import { register } from '@/store/slices/authSlice'
import { RootState, AppDispatch } from '@/store'

export default function Register() {
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const dispatch = useDispatch<AppDispatch>()
  const navigate = useNavigate()
  const { loading, error } = useSelector((state: RootState) => state.auth)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await dispatch(register({ displayName, password })).unwrap()
      navigate('/')
    } catch {
      // Error is handled by the slice
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <div className="w-full max-w-md">
        <h1 className="text-3xl font-bold text-center text-lol-gold mb-8">Register</h1>

        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className="bg-red-500/20 border border-red-500 text-red-300 px-4 py-2 rounded" data-testid="register-error-message">
              {error}
            </div>
          )}

          <div>
            <label htmlFor="displayName" className="block text-sm font-medium text-gray-300 mb-2">
              Username
            </label>
            <input
              type="text"
              id="displayName"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-lol-blue"
              required
              data-testid="register-input-username"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-gray-300 mb-2">
              Password
            </label>
            <input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:border-lol-blue"
              required
              data-testid="register-input-password"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-lol-blue text-black font-semibold py-3 px-6 rounded-lg hover:bg-opacity-80 transition disabled:opacity-50"
            data-testid="register-button-submit"
          >
            {loading ? 'Creating account...' : 'Register'}
          </button>
        </form>

        <p className="text-center text-gray-400 mt-6">
          Already have an account?{' '}
          <Link to="/login" className="text-lol-gold hover:underline">
            Login
          </Link>
        </p>
      </div>
    </div>
  )
}
