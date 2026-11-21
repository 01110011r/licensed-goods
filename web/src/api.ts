const API_BASE = ''

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, options)

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

export const api = {
  products: () => request<Product[]>('/api/products'),

  sellerProducts: (sellerId: string) =>
    request<Product[]>(`/api/sellers/${sellerId}/products`),

  catalogue: () => request<Product[]>('/api/catalogue'),

  reviews: () => request<ReviewItem[]>('/api/reviews'),

  getToday: () => request<{ today: string }>('/api/settings/today'),

  updateToday: (today: string) =>
    request<{ today: string }>('/api/settings/today', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ today }),
    }),

  submitLicence: (formData: FormData) =>
    request<{ id: string; status: string }>('/api/licences', {
      method: 'POST',
      body: formData,
    }),


  decideReview: (
    id: string,
    status: 'approved' | 'rejected',
    rejectionReason = '',
  ) =>
    request<{ id: string; status: string }>(`/api/reviews/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        status,
        rejection_reason: rejectionReason,
      }),
    }),

}
