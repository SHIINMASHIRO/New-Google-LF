import React, { useState, useEffect, useRef } from 'react'
import { Plus, RefreshCw, Play, StopCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { tasksApi, agentsApi, urlPoolsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export default function Tasks() {
  const [groups,    setGroups]    = useState([])
  const [agents,    setAgents]    = useState([])
  const [pools,     setPools]     = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error,     setError]     = useState(null)
  const [metaError, setMetaError] = useState(null)
  const [metaLoaded, setMetaLoaded] = useState(false)
  const navigate = useNavigate()
  const reloadPromiseRef = useRef(null)
  const metaPromiseRef = useRef(null)
  const mountedRef = useRef(true)

  const reload = async () => {
    if (reloadPromiseRef.current) return reloadPromiseRef.current

    reloadPromiseRef.current = (async () => {
      try {
        const t = await tasksApi.list()
        if (!mountedRef.current) return
        setGroups(t || [])
        setError(null)
      } catch (e) {
        if (mountedRef.current) setError(e.message)
      } finally {
        reloadPromiseRef.current = null
      }
    })()

    return reloadPromiseRef.current
  }

  const loadMeta = async () => {
    if (metaPromiseRef.current) return metaPromiseRef.current
    if (agents.length > 0 || pools.length > 0) return

    metaPromiseRef.current = (async () => {
      try {
        const [a, p] = await Promise.all([agentsApi.list(), urlPoolsApi.list()])
        if (!mountedRef.current) return
        setAgents(a || [])
        setPools(p || [])
        setMetaError(null)
        setMetaLoaded(true)
      } catch (e) {
        if (mountedRef.current) setMetaError(e.message)
      } finally {
        metaPromiseRef.current = null
      }
    })()

    return metaPromiseRef.current
  }

  const openModal = () => {
    setShowModal(true)
    loadMeta()
  }

  useEffect(() => {
    let timeoutId

    const poll = async () => {
      await reload()
      if (!mountedRef.current) return
      timeoutId = window.setTimeout(poll, 15000)
    }

    poll()
    return () => {
      mountedRef.current = false
      window.clearTimeout(timeoutId)
    }
  }, [])

  const dispatch = async (id) => { try { await tasksApi.dispatch(id); reload() } catch (e) { alert(e.message) } }
  const stop     = async (id) => { try { await tasksApi.stop(id);     reload() } catch (e) { alert(e.message) } }

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <span className="label" style={{ display: 'block', marginBottom: 6 }}>Scheduler</span>
          <h1 className="page-title">Tasks</h1>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={reload} className="btn-secondary" style={{ padding: '9px 12px' }}>
            <RefreshCw size={14} />
          </button>
          <button onClick={openModal} className="btn-primary">
            <Plus size={14} /> New Task Group
          </button>
        </div>
      </div>

      {error && <div className="error-bar">{error}</div>}
      {metaError && <div className="error-bar">{metaError}</div>}

      <div className="card" style={{ overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead className="tbl-head">
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Status</th>
              <th>Rate</th>
              <th>Progress</th>
              <th>Created</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {groups.length === 0 ? (
              <tr><td colSpan={7}>
                <div className="empty">No task groups yet — click "New Task Group" to create one</div>
              </td></tr>
            ) : groups.map(t => (
              <tr key={t.id} className="tbl-row" style={{ cursor: 'pointer' }}
                onClick={() => navigate(`/tasks/${t.id}`)}>
                <td>
                  <div style={{ fontWeight: 500 }}>{t.name || '(unnamed)'}</div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
                    {(t.pool_count ?? t.pools?.length ?? 0)} pools · {(t.child_count ?? t.children?.length ?? 0)} tasks · {t.execution_scope}
                  </div>
                </td>
                <td><Badge label={t.type} /></td>
                <td><Badge label={t.status} /></td>
                <td>
                  <span className="mono" style={{ fontSize: 12 }}>
                    {t.target_rate_mbps} Mbps{t.execution_scope === 'global' ? ' (total)' : ''}
                  </span>
                </td>
                <td>
                  {t.total_bytes_target > 0 ? (
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <div style={{ width: 72, height: 4, background: 'var(--bg)', borderRadius: 4 }}>
                        <div style={{
                          width: `${Math.min(100, t.total_bytes_done / t.total_bytes_target * 100).toFixed(0)}%`,
                          height: '100%', background: 'var(--accent)', borderRadius: 4,
                        }} />
                      </div>
                      <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>
                        {(t.total_bytes_done / 1e6).toFixed(1)}M
                      </span>
                    </div>
                  ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                </td>
                <td><span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{fmtDate(t.created_at)}</span></td>
                <td onClick={e => e.stopPropagation()}>
                  <div style={{ display: 'flex', gap: 5 }}>
                    {t.status === 'pending' && (
                      <button onClick={() => dispatch(t.id)}
                        style={{
                          padding: '6px 8px', borderRadius: 6,
                          background: 'var(--green-dim)',
                          border: '1px solid rgba(61,122,82,0.2)',
                          color: 'var(--green)', cursor: 'pointer',
                        }}>
                        <Play size={12} />
                      </button>
                    )}
                    {(t.status === 'running' || t.status === 'dispatched') && (
                      <button onClick={() => stop(t.id)}
                        style={{
                          padding: '6px 8px', borderRadius: 6,
                          background: 'var(--red-dim)',
                          border: '1px solid rgba(181,60,43,0.2)',
                          color: 'var(--red)', cursor: 'pointer',
                        }}>
                        <StopCircle size={12} />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <CreateTaskModal
          agents={agents} pools={pools} metaLoaded={metaLoaded}
          onClose={() => setShowModal(false)}
          onSuccess={() => { setShowModal(false); reload() }}
        />
      )}
    </div>
  )
}

function CreateTaskModal({ agents, pools, metaLoaded, onClose, onSuccess }) {
  const [form, setForm] = useState({
    name: '', pool_ids: [], agent_id: '', execution_scope: 'global',
    target_rate_mbps: 10000, duration_days: 7,
    distribution: 'flat', jitter_pct: 0,
    ramp_up_sec: 0, ramp_down_sec: 0,
    concurrent_fragments: 1, retries: 3,
    total_bytes_target: 0, dispatch_rate_tpm: 0,
  })
  const [loading, setLoading] = useState(false)
  const [error,   setError]   = useState(null)

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const selectedPools = pools.filter(p => form.pool_ids.includes(p.id))
  const selectedTypes = Array.from(new Set(selectedPools.map(p => p.type)))
  const togglePool = id => setForm(f => ({
    ...f,
    pool_ids: f.pool_ids.includes(id) ? f.pool_ids.filter(x => x !== id) : [...f.pool_ids, id],
  }))

  const submit = async e => {
    e.preventDefault(); setLoading(true); setError(null)
    try {
      const payload = {
        ...form,
        target_rate_mbps: +form.target_rate_mbps,
        duration_sec: Math.max(0, Math.round(+form.duration_days * 86400)),
        jitter_pct: +form.jitter_pct,
        ramp_up_sec: +form.ramp_up_sec,
        ramp_down_sec: +form.ramp_down_sec,
        concurrent_fragments: +form.concurrent_fragments,
        retries: +form.retries,
        total_bytes_target: +form.total_bytes_target,
        dispatch_rate_tpm: +form.dispatch_rate_tpm,
      }
      delete payload.duration_days
      await tasksApi.create(payload); onSuccess()
    } catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(61,57,41,0.4)',
        backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'center',
        zIndex: 50, padding: 24, overflowY: 'auto',
      }}
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div style={{
        background: 'var(--elevated)',
        border: '1px solid var(--border)',
        borderRadius: 14,
        width: '100%', maxWidth: 560,
        boxShadow: 'var(--shadow-lg)',
        marginTop: 24,
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            New Task Group
          </h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 18 }}>✕</button>
        </div>

        <form onSubmit={submit} style={{ padding: '20px 22px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Field label="Name">
            <input className="input" value={form.name} onChange={e => set('name', e.target.value)} />
          </Field>

          <Field label={`URL Pools * (${form.pool_ids.length} selected)`}>
            <div style={{
              maxHeight: 210, overflowY: 'auto',
              border: '1px solid var(--border)', borderRadius: 8, background: 'var(--bg)',
            }}>
              {!metaLoaded ? (
                <div style={{ padding: '12px 14px', fontSize: 13, color: 'var(--text-muted)' }}>
                  Loading task form data...
                </div>
              ) : pools.length === 0 ? (
                <div style={{ padding: '12px 14px', fontSize: 13, color: 'var(--text-muted)' }}>
                  No URL pools — create one first
                </div>
              ) : pools.map(pool => (
                <label key={pool.id} style={{
                  display: 'flex', alignItems: 'flex-start', gap: 10,
                  padding: '11px 14px',
                  borderBottom: '1px solid var(--border)',
                  cursor: 'pointer',
                  background: form.pool_ids.includes(pool.id) ? 'rgba(218,119,86,0.04)' : 'transparent',
                }}>
                  <input
                    type="checkbox"
                    checked={form.pool_ids.includes(pool.id)}
                    onChange={() => togglePool(pool.id)}
                    style={{ marginTop: 3, accentColor: 'var(--accent)' }}
                  />
                  <div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontWeight: 500, fontSize: 13 }}>{pool.name}</span>
                      <Badge label={pool.type} />
                    </div>
                    <span style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
                      {pool.urls?.length || 0} URLs
                    </span>
                  </div>
                </label>
              ))}
            </div>
            {selectedTypes.length > 0 && (
              <div style={{ display: 'flex', gap: 6, marginTop: 6, alignItems: 'center' }}>
                <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>Types:</span>
                {selectedTypes.map(t => <Badge key={t} label={t} />)}
              </div>
            )}
          </Field>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <Field label="Execution Scope">
              <select className="input" value={form.execution_scope} onChange={e => set('execution_scope', e.target.value)}>
                <option value="global">Global (all agents)</option>
                <option value="single_agent">Single Agent</option>
              </select>
            </Field>
            <Field label="Agent">
              <select className="input" value={form.agent_id} disabled={form.execution_scope === 'global'}
                onChange={e => set('agent_id', e.target.value)}>
                <option value="">{form.execution_scope === 'global' ? 'All online agents' : '— select —'}</option>
                {agents.map(a => <option key={a.id} value={a.id}>{a.hostname} ({a.ip})</option>)}
              </select>
            </Field>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <Field label="Target Rate (Mbps)">
              <input type="number" step="0.1" min="0" className="input" value={form.target_rate_mbps}
                onChange={e => set('target_rate_mbps', e.target.value)} />
            </Field>
            <Field label="Duration (days)">
              <input type="number" step="0.1" min="0" className="input" value={form.duration_days}
                onChange={e => set('duration_days', e.target.value)} />
            </Field>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
            <Field label="Distribution">
              <select className="input" value={form.distribution} onChange={e => set('distribution', e.target.value)}>
                <option value="flat">Flat</option>
                <option value="ramp">Ramp</option>
                <option value="diurnal">Diurnal</option>
              </select>
            </Field>
            <Field label="Jitter %">
              <input type="number" min="0" max="100" className="input" value={form.jitter_pct}
                onChange={e => set('jitter_pct', e.target.value)} />
            </Field>
            <Field label="Ramp Up (s)">
              <input type="number" min="0" className="input" value={form.ramp_up_sec}
                onChange={e => set('ramp_up_sec', e.target.value)} />
            </Field>
          </div>

          {selectedTypes.includes('youtube') && (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <Field label="Concurrent Fragments">
                <input type="number" min="1" className="input" value={form.concurrent_fragments}
                  onChange={e => set('concurrent_fragments', e.target.value)} />
              </Field>
              <Field label="Retries">
                <input type="number" min="0" className="input" value={form.retries}
                  onChange={e => set('retries', e.target.value)} />
              </Field>
            </div>
          )}

          {error && <div className="error-bar">{error}</div>}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 4 }}>
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">
              {loading ? 'Creating...' : 'Create Task Group'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, children }) {
  return (
    <div className="field">
      <label className="field-label">{label}</label>
      {children}
    </div>
  )
}
