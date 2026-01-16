import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

/**
 * Hook that handles auto-login from URL token parameter.
 * Used by the simulator to allow clicking a link to instantly log in as a user.
 *
 * When a ?token=JWT parameter is present in the URL:
 * 1. Clears any existing session (localStorage and cookies)
 * 2. Stores the new token in localStorage
 * 3. Removes the token from the URL for security
 * 4. Reloads the page to ensure fresh auth state
 */
export function useAutoLogin() {
  const location = useLocation()

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const token = params.get('token')

    if (token) {
      // Clear any existing session data
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')

      // Clear any auth-related cookies by setting them to expire
      document.cookie.split(';').forEach((cookie) => {
        const name = cookie.split('=')[0].trim()
        if (name) {
          document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`
        }
      })

      // Store the new token
      localStorage.setItem('accessToken', token)

      // Remove token from URL for security (keep other params if any)
      params.delete('token')
      const newSearch = params.toString()
      const newUrl = location.pathname + (newSearch ? `?${newSearch}` : '')

      // Force a page reload to ensure auth state is fresh
      // This is needed because the initialState reads from localStorage on load
      window.location.href = newUrl
    }
  }, []) // Only run once on mount
}
