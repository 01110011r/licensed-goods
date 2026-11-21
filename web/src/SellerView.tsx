
import { useEffect, useState, type FormEvent } from 'react'
import { api, type Product } from './api'

const sellers = [
  { id: 's1', name: 'Shifo Apteka' },
  { id: 's2', name: 'Vino Market' },
  { id: 's3', name: "Bog' Dunyosi" },
  { id: 's4', name: 'Universal Savdo' },
]

const groups = [
  { id: 'MED', name: 'Medicines' },
  { id: 'ALC', name: 'Alcohol' },
  { id: 'PEST', name: 'Pesticides' },
]

export default function SellerView() {
  const [sellerId, setSellerId] = useState('s1')
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    let cancelled = false

    api.sellerProducts(sellerId)
      .then((result) => {
        if (!cancelled) {
          setProducts(result)
          setError('')
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load products')
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [sellerId])

  async function loadProducts() {
    setLoading(true)
    setError('')

    try {
      const result = await api.sellerProducts(sellerId)
      setProducts(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load products')
    } finally {
      setLoading(false)
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setSuccess('')
    setSubmitting(true)

    try {
      const formData = new FormData(event.currentTarget)
      const result = await api.submitLicence(formData)

      setSuccess(`Application submitted successfully. ID: ${result.id}`)
      event.currentTarget.reset()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Submission failed')
    } finally {
      setSubmitting(false)
    }
  }

  const seller = sellers.find((item) => item.id === sellerId)

  return (
    <>
      <div className="page-heading">
        <div>
          <p className="eyebrow">SELLER WORKSPACE</p>
          <h1>Product compliance</h1>
          <p className="subtitle">
            Manage product availability and submit licence applications.
          </p>
        </div>
        <select
          className="seller-select"
          value={sellerId}
          onChange={(event) => setSellerId(event.target.value)}
        >
          {sellers.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </select>
      </div>

      <section className="panel seller-products">
        <div className="panel-heading">
          <div>
            <h2>{seller?.name} — Products</h2>
            <p>{products.length} products registered</p>
          </div>
          <button className="text-button" onClick={() => void loadProducts()}>
            Refresh ↻
          </button>
        </div>

        {loading ? (
          <div className="empty-state">Loading products...</div>
        ) : error && products.length === 0 ? (
          <div className="error-message">{error}</div>
        ) : products.length === 0 ? (
          <div className="empty-state">No products registered.</div>
        ) : (
          <div className="seller-product-list">
            {products.map((product) => (
              <article className="seller-product" key={product.id}>
                <div className="seller-product-main">
                  <div className="product-symbol">▧</div>
                  <div className="product-info">
                    <strong>{product.name}</strong>
                    <small>
                      IKPU: {product.ikpu} · Category: {product.category_id}
                    </small>
                    <div className="required-groups">
                      {product.required_groups.length > 0
                        ? product.required_groups.map((group) => (
                          <span key={group.id} className="group-chip">
                            {group.name}
                          </span>
                        ))
                        : <span className="no-requirements">No licence required</span>}
                    </div>
                  </div>
                  <span className={`status status-${product.on_hold ? 'on-hold' : product.status}`}>
                    {product.on_hold ? 'On hold' : product.status}
                  </span>
                </div>

                {product.on_hold && (
                  <div className="hold-reasons">
                    {product.hold_reasons.map((reason, index) => (
                      <p key={index}>⚠ {reason}</p>
                    ))}
                  </div>
                )}

                {product.data_issues?.map((issue, index) => (
                  <div className="data-warning" key={index}>
                    Data issue: {issue.detail}
                  </div>
                ))}
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="panel licence-form-panel">
        <div className="panel-heading">
          <div>
            <h2>Submit a licence</h2>
            <p>Send a new application for review.</p>
          </div>
        </div>

        <form className="licence-form" onSubmit={handleSubmit}>
          <input type="hidden" name="seller_id" value={sellerId} />

          <label>
            Licence group
            <select name="group_id" required defaultValue="">
              <option value="" disabled>Select group</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          </label>

          <label>
            Licence number
            <input name="number" required placeholder="e.g. MED-2026-0004" />
          </label>

          <label>
            Issuing authority
            <input name="issuer" required placeholder="Issuing authority" />
          </label>

          <div className="form-dates">
            <label>
              Valid from
              <input name="valid_from" type="date" required />
            </label>
            <label>
              Valid to
              <input name="valid_to" type="date" />
            </label>
          </div>

          <label>
            Supporting document (PDF, PNG or JPG)
            <input name="file" type="file" accept=".pdf,.png,.jpg,.jpeg" />
          </label>

          <button className="primary-button" type="submit" disabled={submitting}>
            {submitting ? 'Submitting...' : 'Submit application'}
          </button>

          {success && <p className="success-message">{success}</p>}
          {error && <p className="error-message">{error}</p>}
        </form>
      </section>
    </>
  )
}
