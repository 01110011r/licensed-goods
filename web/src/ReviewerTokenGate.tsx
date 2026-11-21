import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { reviewerAccess } from './api'

// RequireReviewerToken shows its children only once a reviewer token is held
// for this tab. Otherwise it asks for the token.
export default function RequireReviewerToken({ children }: { children: ReactNode }) {
  const [hasToken, setHasToken] = useState(() => reviewerAccess.get() !== '')

  useEffect(() => {
    return reviewerAccess.subscribe(() => setHasToken(reviewerAccess.get() !== ''))
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
      setError('Enter the reviewer token.')
      return
    }

    setError('')
    reviewerAccess.set(trimmed)
  }

  return (
    <section className="panel">
      <h2>Reviewer access required</h2>
      <p>Enter the reviewer token to continue. It is kept for this browser tab only.</p>

      <form onSubmit={handleSubmit} className="form-grid">
        <label>
          <span>Reviewer token</span>
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
