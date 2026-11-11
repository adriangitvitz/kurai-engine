'use client'

import { useState, useRef, useEffect } from 'react'
import { Logo } from '@/components/folio/Logo'
import { useAuth } from '@/lib/useAuth'
import { safeNext } from '@/lib/nav'

export default function LoginPage() {
  const [key, setKey] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const { login, loading, error, clearError } = useAuth()

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!key.trim() || loading) return

    const success = await login(key.trim())
    if (success) {
      window.location.href = safeNext(new URLSearchParams(window.location.search).get('next'))
    }
  }

  const disabled = loading || !key.trim()

  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '0 20px',
      }}
    >
      <div style={{ width: '100%', maxWidth: 420, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 22 }}>
        <Logo size={40} />

        <p
          style={{
            color: 'var(--muted)',
            textAlign: 'center',
            font: "500 11px 'IBM Plex Mono', monospace",
            letterSpacing: '.1em',
            textTransform: 'uppercase',
          }}
        >
          Enter your API key to continue
        </p>

        <form onSubmit={handleSubmit} style={{ width: '100%' }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              padding: '9px 9px 9px 20px',
              borderRadius: 999,
              background: 'var(--bg)',
              boxShadow: 'var(--inset)',
              border: error ? '1px solid var(--error)' : '1px solid transparent',
            }}
          >
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor"
              strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.45, flexShrink: 0 }}>
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
            <input
              ref={inputRef}
              type="password"
              value={key}
              onChange={(e) => {
                setKey(e.target.value)
                if (error) clearError()
              }}
              placeholder="API key"
              autoComplete="off"
              style={{
                flex: 1,
                border: 'none',
                outline: 'none',
                background: 'transparent',
                fontSize: 'var(--fs-body)',
                fontFamily: "'IBM Plex Mono', monospace",
                color: 'var(--text)',
                padding: '7px 0',
              }}
            />
            <button
              type="submit"
              disabled={disabled}
              aria-label="Continue"
              style={{
                display: 'grid',
                placeItems: 'center',
                width: 44,
                height: 44,
                borderRadius: '50%',
                background: 'var(--accent)',
                flexShrink: 0,
                opacity: disabled ? 0.45 : 1,
                cursor: disabled ? 'default' : 'pointer',
                transition: 'opacity .2s',
              }}
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#161616"
                strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M5 12h13M13 6l6 6-6 6" />
              </svg>
            </button>
          </div>

          {error && (
            <p style={{ color: 'var(--error)', fontSize: 'var(--fs-meta)', marginTop: 12, textAlign: 'center' }}>{error}</p>
          )}

          <p
            style={{
              marginTop: 18,
              textAlign: 'center',
              font: "500 10px 'IBM Plex Mono', monospace",
              letterSpacing: '.06em',
              textTransform: 'uppercase',
              opacity: 0.35,
            }}
          >
            {loading ? 'Authenticating…' : 'A calmer way to search the web'}
          </p>
        </form>
      </div>
    </main>
  )
}
