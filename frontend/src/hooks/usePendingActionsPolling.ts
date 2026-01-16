import { useEffect, useRef } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { RootState, AppDispatch } from '@/store'
import { fetchPendingActions } from '@/store/slices/pendingActionsSlice'

/**
 * Hook that sets up background polling for pending actions.
 * Should be called at the app level when user is authenticated.
 *
 * @param intervalMs - Polling interval in milliseconds (default: 5000ms)
 */
export function usePendingActionsPolling(intervalMs: number = 5000) {
  const dispatch = useDispatch<AppDispatch>()
  const { user } = useSelector((state: RootState) => state.auth)
  const intervalRef = useRef<number | null>(null)

  useEffect(() => {
    // Only poll when user is authenticated
    if (!user) {
      // Clear any existing interval when user logs out
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }

    // Initial fetch
    dispatch(fetchPendingActions())

    // Set up polling interval
    intervalRef.current = window.setInterval(() => {
      dispatch(fetchPendingActions())
    }, intervalMs)

    // Cleanup on unmount or when dependencies change
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [user, dispatch, intervalMs])
}
