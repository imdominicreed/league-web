import { useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchChampions } from '@/store/slices/championsSlice'
import { useWebSocket } from '@/hooks/useWebSocket'
import DraftBoard from '@/components/draft/DraftBoard'

export default function DraftRoom() {
  const { roomId } = useParams<{ roomId: string }>()
  const dispatch = useDispatch<AppDispatch>()
  const { room, connectionStatus } = useSelector((state: RootState) => state.room)
  const { championsList, loading: championsLoading } = useSelector((state: RootState) => state.champions)

  const ws = useWebSocket(roomId || '')

  useEffect(() => {
    if (championsList.length === 0) {
      dispatch(fetchChampions())
    }
  }, [dispatch, championsList.length])

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
