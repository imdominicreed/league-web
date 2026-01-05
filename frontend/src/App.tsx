import { useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from './store'
import { fetchCurrentUser } from './store/slices/authSlice'
import Home from './pages/Home'
import Login from './pages/Login'
import Register from './pages/Register'
import CreateDraft from './pages/CreateDraft'
import JoinDraft from './pages/JoinDraft'
import DraftRoom from './pages/DraftRoom'
import Profile from './pages/Profile'
import CreateLobby from './pages/CreateLobby'
import JoinLobby from './pages/JoinLobby'
import LobbyRoom from './pages/LobbyRoom'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useSelector((state: RootState) => state.auth)

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

function App() {
  const dispatch = useDispatch<AppDispatch>()
  const { isAuthenticated, user } = useSelector((state: RootState) => state.auth)

  useEffect(() => {
    if (isAuthenticated && !user) {
      dispatch(fetchCurrentUser())
    }
  }, [isAuthenticated, user, dispatch])

  return (
    <div className="min-h-screen bg-lol-dark relative">
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/create"
          element={
            <ProtectedRoute>
              <CreateDraft />
            </ProtectedRoute>
          }
        />
        <Route
          path="/join"
          element={
            <ProtectedRoute>
              <JoinDraft />
            </ProtectedRoute>
          }
        />
        <Route
          path="/join/:code"
          element={
            <ProtectedRoute>
              <JoinDraft />
            </ProtectedRoute>
          }
        />
        <Route
          path="/draft/:roomId"
          element={
            <ProtectedRoute>
              <DraftRoom />
            </ProtectedRoute>
          }
        />
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <Profile />
            </ProtectedRoute>
          }
        />
        <Route
          path="/create-lobby"
          element={
            <ProtectedRoute>
              <CreateLobby />
            </ProtectedRoute>
          }
        />
        <Route
          path="/join-lobby"
          element={
            <ProtectedRoute>
              <JoinLobby />
            </ProtectedRoute>
          }
        />
        <Route
          path="/join-lobby/:code"
          element={
            <ProtectedRoute>
              <JoinLobby />
            </ProtectedRoute>
          }
        />
        <Route
          path="/lobby/:lobbyId"
          element={
            <ProtectedRoute>
              <LobbyRoom />
            </ProtectedRoute>
          }
        />
      </Routes>
      <div className="fixed bottom-2 right-2 text-xs text-gray-600">
        v{__APP_VERSION__}
      </div>
    </div>
  )
}

declare const __APP_VERSION__: string

export default App
