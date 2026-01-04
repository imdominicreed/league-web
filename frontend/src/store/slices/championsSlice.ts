import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit'
import { championsApi } from '@/api/champions'
import { Champion } from '@/types'

interface ChampionsState {
  champions: Record<string, Champion>
  championsList: Champion[]
  loading: boolean
  version: string | null
  filters: {
    search: string
    roles: string[]
  }
}

const initialState: ChampionsState = {
  champions: {},
  championsList: [],
  loading: false,
  version: null,
  filters: {
    search: '',
    roles: [],
  },
}

export const fetchChampions = createAsyncThunk(
  'champions/fetchAll',
  async () => {
    return await championsApi.getAll()
  }
)

const championsSlice = createSlice({
  name: 'champions',
  initialState,
  reducers: {
    setSearchFilter: (state, action: PayloadAction<string>) => {
      state.filters.search = action.payload
    },
    setRoleFilter: (state, action: PayloadAction<string[]>) => {
      state.filters.roles = action.payload
    },
    toggleRoleFilter: (state, action: PayloadAction<string>) => {
      const role = action.payload
      if (state.filters.roles.includes(role)) {
        state.filters.roles = state.filters.roles.filter(r => r !== role)
      } else {
        state.filters.roles.push(role)
      }
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchChampions.pending, (state) => {
        state.loading = true
      })
      .addCase(fetchChampions.fulfilled, (state, action) => {
        state.loading = false
        state.championsList = action.payload.champions
        state.version = action.payload.version
        state.champions = action.payload.champions.reduce((acc, champ) => {
          acc[champ.id] = champ
          return acc
        }, {} as Record<string, Champion>)
      })
      .addCase(fetchChampions.rejected, (state) => {
        state.loading = false
      })
  },
})

export const { setSearchFilter, setRoleFilter, toggleRoleFilter } = championsSlice.actions
export default championsSlice.reducer
