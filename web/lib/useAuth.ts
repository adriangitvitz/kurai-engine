import { useState, useCallback } from 'react'
import { authenticate, APIError } from './api'

interface AuthState {
  isAuthenticated: boolean
  userId: string | null
  error: string | null
  loading: boolean
}

export function useAuth() {
  const [state, setState] = useState<AuthState>({
    isAuthenticated: false,
    userId: null,
    error: null,
    loading: false,
  })

  const login = useCallback(async (key: string) => {
    setState(prev => ({ ...prev, loading: true, error: null }))

    try {
      const resp = await authenticate(key)
      setState({
        isAuthenticated: true,
        userId: resp.user_id,
        error: null,
        loading: false,
      })
      return true
    } catch (err) {
      const message =
        err instanceof APIError
          ? err.status === 401
            ? 'Invalid API key'
            : err.status === 429
              ? 'Too many attempts. Try again later.'
              : err.message
          : 'Authentication failed'

      setState({
        isAuthenticated: false,
        userId: null,
        error: message,
        loading: false,
      })
      return false
    }
  }, [])

  const clearError = useCallback(() => {
    setState(prev => ({ ...prev, error: null }))
  }, [])

  return { ...state, login, clearError }
}
