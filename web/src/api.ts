const API_BASE = ''

const TOKEN_KEY = 'adminToken'
const ACCESS_EVENT = 'admin-access-changed'

// The administrator token is kept in sessionStorage: it lasts for this browser tab
// only and is never written into the bundle.
export const adminAccess = {
  get(): string {
    try {
      return sessionStorage.getItem(TOKEN_KEY) ?? ''
    } catch {
      return ''
    }
  },
  set(token: string) {
    try {
      sessionStorage.setItem(TOKEN_KEY, token)
    } catch {
      // Storage unavailable: the token lasts only for this page load.
    }
    window.dispatchEvent(new Event(ACCESS_EVENT))
  },
  clear() {
    try {
      sessionStorage.removeItem(TOKEN_KEY)
    } catch {
      // Nothing to clear.
    }
    window.dispatchEvent(new Event(ACCESS_EVENT))
  },
  subscribe(listener: () => void) {
    window.addEventListener(ACCESS_EVENT, listener)
    return () => window.removeEventListener(ACCESS_EVENT, listener)
  },
}

interface RequestOptions extends RequestInit {
  admin?: boolean
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { admin, ...init } = options
  const headers = new Headers(init.headers)

  if (admin) {
    const token = adminAccess.get()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
  }

  const response = await fetch(`${API_BASE}${path}`, { ...init, headers })

  if (response.status === 401 && admin) {
    adminAccess.clear()
    throw new Error('The administrator token was rejected. Enter it again.')
  }

  if (!response.ok) {
    const message = await response.text()
    throw new Error(message || `Request failed: ${response.status}`)
  }

  return response.json() as Promise<T>
}

export interface RequiredGroup {
  id: string
  name: string
}

export interface Product {
  id: string
  name: string
  seller_id: string
  seller_name: string
  category_id: string
  ikpu: string
  status: 'draft' | 'active' | 'archived'
  required_groups: RequiredGroup[]
  on_hold: boolean
  hold_reasons: string[]
  data_issues: { kind: string; detail: string }[] | null
}

export interface ReviewItem {
  id: string
  seller_id: string
  seller_name: string
  group_id: string
  group_name: string
  number: string
  issuer: string
  valid_from: string
  valid_to: string | null
  created_at: string
  affected_products: Product[]
}

export interface Category {
  id: string
  name: string
  parent_id: string | null
}

export interface NewProductInput {
  seller_id: string
  name: string
  category_id: string
  ikpu: string
  status: 'draft' | 'active' | 'archived'
}

export const api = {
  products: () => request<Product[]>('/api/products'),

  categories: () => request<Category[]>('/api/categories'),

  createProduct: (input: NewProductInput) =>
    request<{ id: string }>('/api/products', {
      method: 'POST',
      admin: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }),

  sellerProducts: (sellerId: string) =>
    request<Product[]>(`/api/sellers/${sellerId}/products`),

  catalogue: () => request<Product[]>('/api/catalogue'),

  reviews: () => request<ReviewItem[]>('/api/reviews', { admin: true }),

  getToday: () => request<{ today: string }>('/api/settings/today'),

  updateToday: (today: string) =>
    request<{ today: string }>('/api/settings/today', {
      method: 'PUT',
      admin: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ today }),
    }),

  submitLicence: (formData: FormData) =>
    request<{ id: string; status: string }>('/api/licences', {
      method: 'POST',
      admin: true,
      body: formData,
    }),

  decideReview: (
    id: string,
    status: 'approved' | 'rejected',
    rejectionReason = '',
  ) =>
    request<{ id: string; status: string }>(`/api/reviews/${id}`, {
      method: 'PATCH',
      admin: true,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        status,
        rejection_reason: rejectionReason,
      }),
    }),
}
