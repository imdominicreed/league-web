const API_BASE = '/api/v1'

interface RequestOptions {
  method?: string
  body?: unknown
  headers?: Record<string, string>
}

async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
  const token = localStorage.getItem('accessToken')

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers,
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const fullUrl = `${API_BASE}${endpoint}`
  console.log(`📡 API Request: ${options.method || 'GET'} ${fullUrl}`)
  console.log(`   Full URL: ${new URL(fullUrl, window.location.origin).href}`)

  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined,
  })

  console.log(`   Response: ${response.status} ${response.statusText}`)

  if (!response.ok) {
    const error = await response.text()
    console.error(`❌ API Error: ${response.status} ${options.method || 'GET'} ${endpoint}`, error)
    throw new Error(error || 'Request failed')
  }

  console.log(`   ✓ Success`)
  return response.json()
}

export const api = {
  get: <T>(endpoint: string) => request<T>(endpoint),
  post: <T>(endpoint: string, body?: unknown) => request<T>(endpoint, { method: 'POST', body }),
  put: <T>(endpoint: string, body?: unknown) => request<T>(endpoint, { method: 'PUT', body }),
  delete: <T>(endpoint: string) => request<T>(endpoint, { method: 'DELETE' }),
}
