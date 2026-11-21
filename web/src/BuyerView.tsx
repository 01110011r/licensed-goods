import { useEffect, useState } from 'react'
import { api, type Product } from './api'

export default function BuyerView() {
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')

  async function loadCatalogue() {
    setLoading(true)
    setError('')

    try {
      const result = await api.catalogue()
      setProducts(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load catalogue')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let cancelled = false

    api.catalogue()
      .then((result) => {
        if (!cancelled) setProducts(result)
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load catalogue')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [])

  const filteredProducts = products.filter((product) => {
    const query = search.trim().toLowerCase()

    return (
      product.name.toLowerCase().includes(query) ||
      product.ikpu.toLowerCase().includes(query) ||
      product.category_id.toLowerCase().includes(query) ||
      product.seller_name.toLowerCase().includes(query)
    )
  })

  return (
    <>
      <div className="page-heading">
        <div>
          <p className="eyebrow">PUBLIC MARKETPLACE</p>
          <h1>Buyer catalogue</h1>
          <p className="subtitle">
            Browse products currently available for purchase.
          </p>
        </div>
        <button
          className="secondary-button"
          onClick={() => void loadCatalogue()}
          disabled={loading}
        >
          Refresh ↻
        </button>
      </div>

      <section className="catalogue-toolbar panel">
        <div>
          <strong>{products.length}</strong>
          <span>Available products</span>
        </div>
        <input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Search products, sellers or IKPU..."
          aria-label="Search catalogue"
        />
      </section>

      {error && <div className="error-message review-alert">{error}</div>}

      {loading ? (
        <div className="panel empty-state">Loading catalogue...</div>
      ) : filteredProducts.length === 0 ? (
        <div className="panel empty-state">
          <strong>
            {products.length === 0
              ? 'No products currently available'
              : 'No matching products'}
          </strong>
          <p>
            {products.length === 0
              ? 'Products will appear here when they are active and meet all licence requirements.'
              : 'Try a different search term.'}
          </p>
        </div>
      ) : (
        <div className="catalogue-grid">
          {filteredProducts.map((product) => (
            <article className="panel catalogue-card" key={product.id}>
              <div className="catalogue-product-symbol">▧</div>

              <div className="catalogue-card-content">
                <span className="catalogue-availability">
                  <span className="availability-dot" />
                  Available
                </span>

                <h2>{product.name}</h2>
                <p className="catalogue-seller">{product.seller_name}</p>

                <div className="catalogue-metadata">
                  <div>
                    <span>Category</span>
                    <strong>{product.category_id}</strong>
                  </div>
                  <div>
                    <span>IKPU</span>
                    <strong>{product.ikpu}</strong>
                  </div>
                </div>

                {product.required_groups.length > 0 && (
                  <div className="required-groups">
                    {product.required_groups.map((group) => (
                      <span className="group-chip" key={group.id}>
                        {group.name}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            </article>
          ))}
        </div>
      )}
    </>
  )
}
