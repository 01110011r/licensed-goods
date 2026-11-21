import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { adminAccess } from './api'

// RequireAdminToken shows its children only once a administrator token is held
// for this tab. Otherwise it asks for the token.
export default function RequireAdminToken({ children }: { children: ReactNode }) {
  const [hasToken, setHasToken] = useState(() => adminAccess.get() !== '')

  useEffect(() => {
    return adminAccess.subscribe(() => setHasToken(adminAccess.get() !== ''))
  }, [])

  if (hasToken) {
    return <>{children}</>
  }

  return <TokenForm />
}

function TokenForm() {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const trimmed = token.trim()
    if (trimmed === '') {
      setError('Enter the administrator token.')
      return
    }

    setError('')
    adminAccess.set(trimmed)
  }

  return (
    <section className="panel">
      <h2>Administrator access required</h2>
      <p>Enter the administrator token to continue. It is kept for this browser tab only.</p>

      <form onSubmit={handleSubmit} className="form-grid">
        <label>
          <span>Administrator token</span>
          <input
            type="password"
            autoComplete="off"
            value={token}
            onChange={(event) => setToken(event.target.value)}
          />
        </label>

        {error && <p className="form-error">{error}</p>}

        <button type="submit">Continue</button>
      </form>
    </section>
  )
}
