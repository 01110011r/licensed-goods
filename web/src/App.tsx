import { useEffect, useState } from 'react'
import { api, type Product, type ReviewItem } from './api'
import SellerView from './SellerView'
import ReviewerView from './ReviewerView'
import BuyerView from './BuyerView'
import SettingsView from './SettingsView'
import RequireAdminToken from './AdminTokenGate'
import './App.css'

type View = 'overview' | 'seller' | 'reviewer' | 'buyer' | 'settings'

const navigation: { id: View; label: string; icon: string }[] = [
  { id: 'overview', label: 'Overview', icon: '◫' },
  { id: 'seller', label: 'Seller workspace', icon: '▤' },
  { id: 'reviewer', label: 'Licence reviews', icon: '✓' },
  { id: 'buyer', label: 'Buyer catalogue', icon: '⌕' },
  { id: 'settings', label: 'Settings', icon: '⚙' },
]

function App() {
  const [view, setView] = useState<View>('overview')
  const [products, setProducts] = useState<Product[]>([])
  const [reviews, setReviews] = useState<ReviewItem[]>([])
  const [today, setToday] = useState('')
  const [connected, setConnected] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    async function loadDashboard() {
      try {
        const [productData, reviewData, dateData] = await Promise.all([
          api.products(),
          // The queue needs the administrator token; without one the dashboard still loads.
          api.reviews().catch(() => []),
          api.getToday(),
        ])

        setProducts(productData)
        setReviews(reviewData)
        setToday(dateData.today)
        setConnected(true)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'API connection failed')
      } finally {
        setLoading(false)
      }
    }

    loadDashboard()
  }, [])

  const heldCount = products.filter((p) => p.on_hold).length
  const activeCount = products.filter((p) => p.status === 'active').length

  const pageTitle = navigation.find((item) => item.id === view)?.label

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark">L</div>
          <div>
            <strong>Licen<span>tia</span></strong>
            <small>GOODS MARKETPLACE</small>
          </div>
        </div>

        <div className="nav-label">WORKSPACE</div>
        <nav>
          {navigation.map((item) => (
            <button
              key={item.id}
              className={`nav-item ${view === item.id ? 'active' : ''}`}
              onClick={() => setView(item.id)}
            >
              <span className="nav-icon">{item.icon}</span>
              {item.label}
              {item.id === 'reviewer' && reviews.length > 0 && (
                <span className="nav-count">{reviews.length}</span>
              )}
            </button>
          ))}
        </nav>

        <div className="sidebar-bottom">
          <div className="connection">
            <span className={`connection-dot ${connected ? 'online' : ''}`} />
            <span>{connected ? 'API connected' : 'Connecting...'}</span>
          </div>
          <small>Licensed Goods · Assessment</small>
        </div>
      </aside>

      <main className="main-area">
        <header className="topbar">
          <div>
            <span className="breadcrumb">Marketplace / </span>
            <strong>{pageTitle}</strong>
          </div>
          <div className="topbar-right">
            <span className="date-label">SYSTEM DATE</span>
            <strong>{today || '—'}</strong>
            <div className="avatar">A</div>
          </div>
        </header>

        <div className="page-content">
          {loading ? (
            <div className="loading">Loading marketplace data...</div>
          ) : error ? (
            <div className="error-panel">
              <strong>Unable to load dashboard</strong>
              <p>{error}</p>
              <button onClick={() => window.location.reload()}>Retry</button>
            </div>
          ) : view === 'overview' ? (
            <>
              <div className="page-heading">
                <div>
                  <p className="eyebrow">MARKETPLACE OVERVIEW</p>
                  <h1>Good day, Administrator</h1>
                  <p className="subtitle">
                    Here's what's happening across your marketplace today.
                  </p>
                </div>
                <span className="date-chip">Today · {today}</span>
              </div>

              <section className="stats-grid">
                <StatCard label="Total products" value={products.length} detail="Across all sellers" icon="▤" />
                <StatCard label="Active products" value={activeCount} detail="Currently published" icon="◉" />
                <StatCard label="Products on hold" value={heldCount} detail="Licence requirements unmet" icon="Ⅱ" />
                <StatCard label="Pending reviews" value={reviews.length} detail="Awaiting decision" icon="◷" />
              </section>

              <section className="content-grid">
                <div className="panel">
                  <div className="panel-heading">
                    <div>
                      <h2>Recent products</h2>
                      <p>Latest marketplace inventory</p>
                    </div>
                    <button className="text-button" onClick={() => setView('seller')}>
                      View all →
                    </button>
                  </div>
                  <div className="product-list">
                    {products.slice(0, 5).map((product) => (
                      <div className="product-row" key={product.id}>
                        <div className="product-symbol">▧</div>
                        <div className="product-info">
                          <strong>{product.name}</strong>
                          <small>{product.seller_name} · {product.ikpu}</small>
                        </div>
                        <Status status={product.on_hold ? 'On hold' : product.status} />
                      </div>
                    ))}
                  </div>
                </div>

                <div className="panel">
                  <div className="panel-heading">
                    <div>
                      <h2>Review queue</h2>
                      <p>Applications requiring attention</p>
                    </div>
                    <button className="text-button" onClick={() => setView('reviewer')}>
                      Open queue →
                    </button>
                  </div>
                  {reviews.length === 0 ? (
                    <div className="empty-state">No pending applications.</div>
                  ) : (
                    reviews.slice(0, 5).map((review) => (
                      <div className="review-row" key={review.id}>
                        <div>
                          <strong>{review.number || 'Unnumbered licence'}</strong>
                          <small>{review.seller_name} · {review.group_name}</small>
                        </div>
                        <Status status="Pending" />
                      </div>
                    ))
                  )}
                </div>
              </section>
            </>
          ) : view === 'seller' ? (
            <RequireAdminToken>
              <SellerView />
            </RequireAdminToken>
          ) : view === 'reviewer' ? (
            <RequireAdminToken>
              <ReviewerView />
            </RequireAdminToken>
          ) : view === 'buyer' ? (
            <BuyerView />
          ) : view === 'settings' ? (
            <RequireAdminToken>
              <SettingsView />
            </RequireAdminToken>
          ) : (
            <div className="placeholder-panel">
              <p className="eyebrow">WORKSPACE</p>
              <h1>{pageTitle}</h1>
              <p className="subtitle">This view is coming in the next implementation step.</p>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}

function StatCard({
  label,
  value,
  detail,
  icon,
}: {
  label: string
  value: number
  detail: string
  icon: string
}) {
  return (
    <div className="stat-card">
      <div className="stat-top">
        <span>{label}</span>
        <span className="stat-icon">{icon}</span>
      </div>
      <strong className="stat-value">{value}</strong>
      <small>{detail}</small>
    </div>
  )
}

function Status({ status }: { status: string }) {
  const normalized = status.toLowerCase().replaceAll(' ', '-')
  return <span className={`status status-${normalized}`}>{status}</span>
}

export default App
