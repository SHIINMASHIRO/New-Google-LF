import React, { useState, useEffect } from 'react'
import { Plus, RefreshCw, Play, StopCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { tasksApi, agentsApi, urlPoolsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export default function Tasks() {
  const [groups, setGroups] = useState([])
  const [agents, setAgents] = useState([])
  const [pools, setPools] = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error, setError] = useState(null)
  const navigate = useNavigate()

  const reload = async () => {
    try {
      const [t, a, p] = await Promise.all([tasksApi.list(), agentsApi.list(), urlPoolsApi.list()])
      setGroups(t || [])
      setAgents(a || [])
      setPools(p || [])
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => { reload(); const t = setInterval(reload, 5000); return () => clearInterval(t) }, [])

  const dispatch = async (id) => {
    try { await tasksApi.dispatch(id); reload() } catch (e) { alert(e.message) }
  }
  const stop = async (id) => {
    try { await tasksApi.stop(id); reload() } catch (e) { alert(e.message) }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">Tasks</h1>
          <p className="text-sm text-gray-500 mt-0.5">{groups.length} total task groups</p>
        </div>
        <div className="flex gap-2">
          <button onClick={reload} className="p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors">
            <RefreshCw size={15} />
          </button>
          <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm transition-colors">
            <Plus size={15} /> New Task Group
          </button>
        </div>
      </div>

      {error && <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{error}</div>}

      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="border-b border-gray-800 bg-gray-800/50">
            <tr className="text-gray-400 text-xs">
              <th className="px-4 py-3 text-left font-medium">Name</th>
              <th className="px-4 py-3 text-left font-medium">Type</th>
              <th className="px-4 py-3 text-left font-medium">Status</th>
              <th className="px-4 py-3 text-left font-medium">Rate</th>
              <th className="px-4 py-3 text-left font-medium">Progress</th>
              <th className="px-4 py-3 text-left font-medium">Created</th>
              <th className="px-4 py-3 text-left font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800/50">
            {groups.length === 0 ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-600">No task groups yet. Click "New Task Group" to create one.</td></tr>
            ) : groups.map(t => (
              <tr key={t.id} className="hover:bg-gray-800/30 transition-colors cursor-pointer" onClick={() => navigate(`/tasks/${t.id}`)}>
                <td className="px-4 py-3">
                  <span className="text-white font-medium">{t.name || '(unnamed)'}</span>
                  <p className="text-xs text-gray-500 truncate max-w-xs mt-0.5">
                    {(t.pools?.length || 0)} pools · {(t.children?.length || 0)} child tasks · {t.execution_scope}
                  </p>
                </td>
                <td className="px-4 py-3"><Badge label={t.type} /></td>
                <td className="px-4 py-3"><Badge label={t.status} /></td>
                <td className="px-4 py-3 font-mono text-xs text-gray-300">
                  {t.target_rate_mbps} Mbps{t.execution_scope === 'global' ? ' total' : ''}
                </td>
                <td className="px-4 py-3">
                  {t.total_bytes_target > 0 ? (
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-gray-800 rounded-full h-1.5 w-20">
                        <div className="bg-blue-500 h-1.5 rounded-full" style={{ width: `${Math.min(100, t.total_bytes_done / t.total_bytes_target * 100).toFixed(0)}%` }} />
                      </div>
                      <span className="text-xs text-gray-500">{(t.total_bytes_done / 1e6).toFixed(1)}M</span>
                    </div>
                  ) : <span className="text-xs text-gray-600">—</span>}
                </td>
                <td className="px-4 py-3 text-xs text-gray-500">{fmtDate(t.created_at)}</td>
                <td className="px-4 py-3" onClick={e => e.stopPropagation()}>
                  <div className="flex gap-1">
                    {t.status === 'pending' && (
                      <button onClick={() => dispatch(t.id)} title="Dispatch"
                        className="p-1.5 rounded bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors">
                        <Play size={12} />
                      </button>
                    )}
                    {(t.status === 'running' || t.status === 'dispatched') && (
                      <button onClick={() => stop(t.id)} title="Stop"
                        className="p-1.5 rounded bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors">
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

      {showModal && <CreateTaskModal agents={agents} pools={pools} onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />}
    </div>
  )
}

function CreateTaskModal({ agents, pools, onClose, onSuccess }) {
  const [form, setForm] = useState({
    name: '', pool_ids: [], agent_id: '', execution_scope: 'global',
    target_rate_mbps: 10000, duration_days: 7,
    distribution: 'flat', jitter_pct: 0,
    ramp_up_sec: 0, ramp_down_sec: 0,
    concurrent_fragments: 1, retries: 3,
    total_bytes_target: 0, dispatch_rate_tpm: 0,
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))
  const selectedPools = pools.filter(p => form.pool_ids.includes(p.id))
  const selectedTypes = Array.from(new Set(selectedPools.map(p => p.type)))

  const togglePool = (poolID) => {
    setForm(f => ({
      ...f,
      pool_ids: f.pool_ids.includes(poolID)
        ? f.pool_ids.filter(id => id !== poolID)
        : [...f.pool_ids, poolID],
    }))
  }

  const submit = async e => {
    e.preventDefault()
    setLoading(true)
    setError(null)
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
      await tasksApi.create(payload)
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4 overflow-y-auto"
      onClick={e => e.target === e.currentTarget && onClose()}>
      <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-lg shadow-2xl my-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
          <h3 className="text-sm font-semibold text-white">New Task Group</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-white">✕</button>
        </div>
        <form onSubmit={submit} className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Name"><input className="input" value={form.name} onChange={e => set('name', e.target.value)} /></Field>
            <Field label="Selected Pools">
              <div className="input flex items-center">{selectedPools.length} selected</div>
            </Field>
          </div>
          <Field label="Pools *">
            <div className="max-h-56 overflow-y-auto rounded-lg border border-gray-800 bg-gray-950/70 divide-y divide-gray-800">
              {pools.map(pool => (
                <label key={pool.id} className="flex items-start gap-3 px-3 py-3 cursor-pointer hover:bg-gray-900/60">
                  <input
                    type="checkbox"
                    checked={form.pool_ids.includes(pool.id)}
                    onChange={() => togglePool(pool.id)}
                    className="mt-0.5"
                  />
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-white">{pool.name}</span>
                      <Badge label={pool.type} />
                    </div>
                    <div className="text-xs text-gray-500 mt-1">{pool.urls?.length || 0} URLs</div>
                  </div>
                </label>
              ))}
            </div>
          </Field>
          {selectedPools.length > 0 && (
            <div className="rounded-lg border border-gray-800 bg-gray-950/70 p-3 text-xs text-gray-400">
              <div className="flex items-center gap-2">
                <span>Types:</span>
                {selectedTypes.map(type => <Badge key={type} label={type} />)}
                <span>{selectedPools.length} pools</span>
              </div>
            </div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <Field label="Execution Scope">
              <select className="input" value={form.execution_scope} onChange={e => set('execution_scope', e.target.value)}>
                <option value="global">Global (all matching agents)</option>
                <option value="single_agent">Single Agent</option>
              </select>
            </Field>
            <Field label="Agent">
            <select
              className="input"
              value={form.agent_id}
              disabled={form.execution_scope === 'global'}
              onChange={e => set('agent_id', e.target.value)}
            >
              <option value="">{form.execution_scope === 'global' ? 'Used by all online agents' : '— select agent —'}</option>
              {agents.map(a => <option key={a.id} value={a.id}>{a.hostname} ({a.ip})</option>)}
            </select>
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Target Rate (Mbps)">
              <input type="number" step="0.1" min="0" className="input" value={form.target_rate_mbps} onChange={e => set('target_rate_mbps', e.target.value)} />
            </Field>
            <Field label="Duration (days)">
              <input type="number" step="0.1" min="0" className="input" value={form.duration_days} onChange={e => set('duration_days', e.target.value)} />
            </Field>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <Field label="Distribution">
              <select className="input" value={form.distribution} onChange={e => set('distribution', e.target.value)}>
                <option value="flat">Flat</option>
                <option value="ramp">Ramp</option>
                <option value="diurnal">Diurnal</option>
              </select>
            </Field>
            <Field label="Jitter %"><input type="number" min="0" max="100" className="input" value={form.jitter_pct} onChange={e => set('jitter_pct', e.target.value)} /></Field>
            <Field label="Ramp Up (s)"><input type="number" min="0" className="input" value={form.ramp_up_sec} onChange={e => set('ramp_up_sec', e.target.value)} /></Field>
          </div>
          {selectedTypes.includes('youtube') && (
            <div className="grid grid-cols-2 gap-4">
              <Field label="Concurrent Fragments"><input type="number" min="1" className="input" value={form.concurrent_fragments} onChange={e => set('concurrent_fragments', e.target.value)} /></Field>
              <Field label="Retries"><input type="number" min="0" className="input" value={form.retries} onChange={e => set('retries', e.target.value)} /></Field>
            </div>
          )}
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <div className="flex gap-2 justify-end pt-2">
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">{loading ? 'Creating...' : 'Create Task Group'}</button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, children }) {
  return (
    <label className="block">
      <span className="text-xs text-gray-400 font-medium mb-1 block">{label}</span>
      {children}
    </label>
  )
}
