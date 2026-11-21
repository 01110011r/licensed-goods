import { useEffect, useState, type FormEvent } from 'react'
import { api } from './api'

export default function SettingsView() {
  const [today, setToday] = useState('')
  const [savedToday, setSavedToday] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  useEffect(() => {
    let cancelled = false

    api.getToday()
      .then((result) => {
        if (!cancelled) {
          setToday(result.today)
          setSavedToday(result.today)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load settings')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setSuccess('')
    setSaving(true)

    try {
      const result = await api.updateToday(today)
      setToday(result.today)
      setSavedToday(result.today)
      setSuccess('Marketplace date updated successfully.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update date')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <p className="eyebrow">MARKETPLACE CONFIGURATION</p>
          <h1>Settings</h1>
          <p className="subtitle">
            Manage global marketplace behaviour.
          </p>
        </div>
      </div>

      <section className="panel settings-panel">
        <div className="settings-section-heading">
          <div className="settings-icon">◷</div>
          <div>
            <h2>Marketplace date</h2>
            <p>
              Controls licence validity calculations across the marketplace.
            </p>
          </div>
        </div>

        {loading ? (
          <div className="empty-state">Loading settings...</div>
        ) : (
          <form className="settings-form" onSubmit={handleSubmit}>
            <label>
              Current marketplace date
              <input
                type="date"
                value={today}
                onChange={(event) => {
                  setToday(event.target.value)
                  setSuccess('')
                }}
                required
              />
            </label>

            <div className="settings-date-info">
              <span>Saved date</span>
              <strong>{savedToday || 'Not available'}</strong>
            </div>

            <p className="settings-description">
              Products requiring licences are placed on hold when their seller
              does not have an approved licence valid on this date. Changing
              this date also affects whether reviewers can approve applications.
            </p>

            <div className="settings-actions">
              <button
                type="button"
                className="secondary-button"
                disabled={saving || today === savedToday}
                onClick={() => {
                  setToday(savedToday)
                  setError('')
                  setSuccess('')
                }}
              >
                Reset
              </button>

              <button
                type="submit"
                className="primary-button"
                disabled={saving || !today || today === savedToday}
              >
                {saving ? 'Saving...' : 'Save changes'}
              </button>
            </div>
          </form>
        )}

        {error && <p className="error-message">{error}</p>}
        {success && <p className="success-message">{success}</p>}
      </section>
    </>
  )
}
