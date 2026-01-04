import { api } from './client'
import { Room } from '@/types'

interface CreateRoomRequest {
  draftMode?: 'pro_play' | 'fearless'
  timerDuration?: number
}

interface JoinRoomResponse {
  room: Room
  yourSide: string
  websocketUrl: string
}

export const roomsApi = {
  create: (data: CreateRoomRequest = {}): Promise<Room> =>
    api.post('/rooms', data),

  get: (idOrCode: string): Promise<Room> =>
    api.get(`/rooms/${idOrCode}`),

  join: (idOrCode: string, side: string): Promise<JoinRoomResponse> =>
    api.post(`/rooms/${idOrCode}/join`, { side }),

  getByCode: (code: string): Promise<Room> =>
    api.get(`/rooms/code/${code}`),

  getUserRooms: (): Promise<Room[]> =>
    api.get('/users/me/drafts'),
}
