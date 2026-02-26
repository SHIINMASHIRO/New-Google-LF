const BASE = '/api/v1'

async function req(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json' },
  }
  if (body) opts.body = JSON.stringify(body)
  const res = await fetch(BASE + path, opts)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

// ── Agents ──────────────────────────────────────────────────────────────────
export const agentsApi = {
  list: () => req('GET', '/agents'),
  get: (id) => req('GET', `/agents/${id}`),
  provision: (data) => req('POST', '/agents/provision', data),
  listProvisionJobs: () => req('GET', '/agents/provision-jobs'),
  getProvisionJob: (id) => req('GET', `/agents/provision-jobs/${id}`),
  retryProvisionJob: (id) => req('POST', `/agents/provision-jobs/${id}/retry`),
}

// ── Tasks ────────────────────────────────────────────────────────────────────
export const tasksApi = {
  list: () => req('GET', '/tasks'),
  get: (id) => req('GET', `/tasks/${id}`),
  create: (data) => req('POST', '/tasks', data),
  dispatch: (id) => req('POST', `/tasks/${id}/dispatch`),
  stop: (id) => req('POST', `/tasks/${id}/stop`),
  getMetrics: (id, from, to) => req('GET', `/tasks/${id}/metrics?from=${from}&to=${to}`),
}

// ── Traffic Profiles ─────────────────────────────────────────────────────────
export const profilesApi = {
  list: () => req('GET', '/traffic-profiles'),
  create: (data) => req('POST', '/traffic-profiles', data),
}

// ── Dashboard ────────────────────────────────────────────────────────────────
export const dashboardApi = {
  overview: () => req('GET', '/dashboard/overview'),
  bandwidthHistory: (from, to, step = '5m') =>
    req('GET', `/dashboard/bandwidth/history?from=${from}&to=${to}&step=${step}`),
}

// ── Credentials ──────────────────────────────────────────────────────────────
export const credentialsApi = {
  list: () => req('GET', '/credentials'),
  create: (data) => req('POST', '/credentials', data),
  delete: (id) => req('DELETE', `/credentials/${id}`),
}
