import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchChampions } from '@/store/slices/championsSlice'
import { useWebSocket } from '@/hooks/useWebSocket'
import { roomsApi } from '@/api/rooms'
import DraftBoard from '@/components/draft/DraftBoard'

export default function DraftRoom() {
  const { roomId } = useParams<{ roomId: string }>()
  const dispatch = useDispatch<AppDispatch>()
  const { room, connectionStatus } = useSelector((state: RootState) => state.room)
  const { championsList, loading: championsLoading } = useSelector((state: RootState) => state.champions)

  const [assignedSide, setAssignedSide] = useState<string | null>(null)
  const [joinLoading, setJoinLoading] = useState(true)
  const [joinError, setJoinError] = useState<string | null>(null)

  // Call REST join endpoint before WebSocket
  useEffect(() => {
    const joinRoom = async () => {
      if (!roomId) {
        setJoinError('Room ID not found')
        setJoinLoading(false)
        return
      }

      try {
        const response = await roomsApi.join(roomId, 'auto')
        setAssignedSide(response.yourSide)
        setJoinError(null)
      } catch (error) {
        console.error('Failed to join room:', error)
        setJoinError(error instanceof Error ? error.message : 'Failed to join room')
      } finally {
        setJoinLoading(false)
      }
    }

    joinRoom()
  }, [roomId])

  const ws = useWebSocket(roomId || '', assignedSide || '')

  useEffect(() => {
    if (championsList.length === 0) {
      dispatch(fetchChampions())
    }
  }, [dispatch, championsList.length])

  if (joinError) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="text-xl text-red-400 mb-4">Failed to join room</div>
          <div className="text-gray-400">{joinError}</div>
        </div>
      </div>
    )
  }

  if (joinLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Joining room...</div>
      </div>
    )
  }

  if (connectionStatus === 'connecting') {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Connecting...</div>
      </div>
    )
  }

  if (championsLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Loading champions...</div>
      </div>
    )
  }

  if (!room) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-xl text-gray-400">Loading room...</div>
      </div>
    )
  }

  return <DraftBoard ws={ws} />
}
