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
  delete: (id) => req('DELETE', `/agents/${id}`),
  provision: (data) => req('POST', '/agents/provision', data),
  listProvisionJobs: () => req('GET', '/agents/provision-jobs'),
  getProvisionJob: (id) => req('GET', `/agents/provision-jobs/${id}`),
  retryProvisionJob: (id) => req('POST', `/agents/provision-jobs/${id}/retry`),
  deleteProvisionJob: (id) => req('DELETE', `/agents/provision-jobs/${id}`),
}

// ── Tasks ────────────────────────────────────────────────────────────────────
export const tasksApi = {
  list: () => req('GET', '/task-groups'),
  get: (id) => req('GET', `/task-groups/${id}`),
  create: (data) => req('POST', '/task-groups', data),
  dispatch: (id) => req('POST', `/task-groups/${id}/dispatch`),
  stop: (id) => req('POST', `/task-groups/${id}/stop`),
  getMetrics: (id, from, to) => req('GET', `/task-groups/${id}/metrics?from=${from}&to=${to}`),
}

// ── Traffic Profiles ─────────────────────────────────────────────────────────
export const profilesApi = {
  list: () => req('GET', '/traffic-profiles'),
  create: (data) => req('POST', '/traffic-profiles', data),
}

export const urlPoolsApi = {
  list: () => req('GET', '/url-pools'),
  get: (id) => req('GET', `/url-pools/${id}`),
  create: (data) => req('POST', '/url-pools', data),
  update: (id, data) => req('PUT', `/url-pools/${id}`, data),
  delete: (id) => req('DELETE', `/url-pools/${id}`),
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
