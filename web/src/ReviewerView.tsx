import { useEffect, useState } from 'react'
import { api, type ReviewItem } from './api'

export default function ReviewerView() {
  const [reviews, setReviews] = useState<ReviewItem[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [reasons, setReasons] = useState<Record<string, string>>({})

  useEffect(() => {
    let cancelled = false

    api.reviews()
      .then((result) => {
        if (!cancelled) setReviews(result)
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load reviews')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  async function refreshQueue() {
    const result = await api.reviews()
    setReviews(result)
  }

  async function handleDecision(
    review: ReviewItem,
    decision: 'approved' | 'rejected',
  ) {
    const reason = reasons[review.id]?.trim() ?? ''

    if (decision === 'rejected' && !reason) {
      setError('Please provide a rejection reason.')
      return
    }

    setBusyId(review.id)
    setError('')
    setSuccess('')

    try {
      await api.decideReview(review.id, decision, reason)
      await refreshQueue()

      setSuccess(
        `Licence ${review.number || review.id} was ${decision}.`,
      )

      setReasons((previous) => {
        const next = { ...previous }
        delete next[review.id]
        return next
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Review decision failed')
    } finally {
      setBusyId('')
    }
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <p className="eyebrow">LICENCE MANAGEMENT</p>
          <h1>Review applications</h1>
          <p className="subtitle">
            Inspect submitted licences and decide whether to approve or reject them.
          </p>
        </div>
        <button
          className="secondary-button"
          onClick={() => {
            setLoading(true)
            refreshQueue()
              .catch((err: unknown) => {
                setError(err instanceof Error ? err.message : 'Refresh failed')
              })
              .finally(() => setLoading(false))
          }}
        >
          Refresh queue ↻
        </button>
      </div>

      <div className="review-summary">
        <div>
          <strong>{reviews.length}</strong>
          <span>Pending applications</span>
        </div>
        <p>Applications are ordered by submission date, oldest first.</p>
      </div>

      {error && <div className="error-message review-alert">{error}</div>}
      {success && <div className="success-message review-alert">{success}</div>}

      {loading ? (
        <div className="panel empty-state">Loading applications...</div>
      ) : reviews.length === 0 ? (
        <div className="panel empty-state">
          <strong>No pending applications</strong>
          <p>All submitted licences have been reviewed.</p>
        </div>
      ) : (
        <div className="review-list">
          {reviews.map((review) => (
            <article className="panel review-card" key={review.id}>
              <div className="review-card-header">
                <div>
                  <span className="eyebrow">LICENCE APPLICATION</span>
                  <h2>{review.number || 'Unnumbered licence'}</h2>
                  <p>{review.seller_name} · {review.group_name}</p>
                </div>
                <span className="status status-pending">Pending</span>
              </div>

              <div className="review-details">
                <div>
                  <span>Licence group</span>
                  <strong>{review.group_name || review.group_id}</strong>
                </div>
                <div>
                  <span>Issuing authority</span>
                  <strong>{review.issuer || '—'}</strong>
                </div>
                <div>
                  <span>Valid from</span>
                  <strong>{review.valid_from}</strong>
                </div>
                <div>
                  <span>Valid to</span>
                  <strong>{review.valid_to || 'No expiry date'}</strong>
                </div>
                <div>
                  <span>Submitted</span>
                  <strong>
                    {new Date(review.created_at).toLocaleString()}
                  </strong>
                </div>
              </div>

              <div className="affected-section">
                <h3>Affected products ({review.affected_products.length})</h3>

                {review.affected_products.length === 0 ? (
                  <p className="muted-text">
                    No matching products currently require this licence.
                  </p>
                ) : (
                  <div className="affected-list">
                    {review.affected_products.map((product) => (
                      <div className="affected-product" key={product.id}>
                        <div>
                          <strong>{product.name}</strong>
                          <small>
                            {product.ikpu} · {product.category_id}
                          </small>
                        </div>
                        <span
                          className={`status status-${product.on_hold ? 'on-hold' : product.status}`}
                        >
                          {product.on_hold ? 'On hold' : product.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="decision-section">
                <label>
                  Rejection reason
                  <textarea
                    value={reasons[review.id] ?? ''}
                    onChange={(event) =>
                      setReasons((previous) => ({
                        ...previous,
                        [review.id]: event.target.value,
                      }))
                    }
                    placeholder="Required only when rejecting this application"
                    rows={2}
                  />
                </label>

                <div className="decision-actions">
                  <button
                    className="reject-button"
                    disabled={busyId === review.id}
                    onClick={() => void handleDecision(review, 'rejected')}
                  >
                    Reject
                  </button>
                  <button
                    className="approve-button"
                    disabled={busyId === review.id}
                    onClick={() => void handleDecision(review, 'approved')}
                  >
                    {busyId === review.id ? 'Processing...' : 'Approve licence'}
                  </button>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}
    </>
  )
}
