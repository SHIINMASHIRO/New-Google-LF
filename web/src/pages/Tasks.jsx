import React, { useState, useEffect } from 'react'
import { Plus, RefreshCw, Play, StopCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { tasksApi, agentsApi, profilesApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export default function Tasks() {
  const [tasks, setTasks] = useState([])
  const [agents, setAgents] = useState([])
  const [profiles, setProfiles] = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error, setError] = useState(null)
  const navigate = useNavigate()

  const reload = async () => {
    try {
      const [t, a, p] = await Promise.all([tasksApi.list(), agentsApi.list(), profilesApi.list()])
      setTasks(t || [])
      setAgents(a || [])
      setProfiles(p || [])
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
          <p className="text-sm text-gray-500 mt-0.5">{tasks.length} total tasks</p>
        </div>
        <div className="flex gap-2">
          <button onClick={reload} className="p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors">
            <RefreshCw size={15} />
          </button>
          <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm transition-colors">
            <Plus size={15} /> New Task
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
            {tasks.length === 0 ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-600">No tasks yet. Click "New Task" to create one.</td></tr>
            ) : tasks.map(t => (
              <tr key={t.id} className="hover:bg-gray-800/30 transition-colors cursor-pointer" onClick={() => navigate(`/tasks/${t.id}`)}>
                <td className="px-4 py-3">
                  <span className="text-white font-medium">{t.name || '(unnamed)'}</span>
                  <p className="text-xs text-gray-500 truncate max-w-xs mt-0.5">{t.target_url}</p>
                </td>
                <td className="px-4 py-3"><Badge label={t.type} /></td>
                <td className="px-4 py-3"><Badge label={t.status} /></td>
                <td className="px-4 py-3 font-mono text-xs text-gray-300">{t.target_rate_mbps} Mbps</td>
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

      {showModal && <CreateTaskModal agents={agents} profiles={profiles} onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />}
    </div>
  )
}

function CreateTaskModal({ agents, profiles, onClose, onSuccess }) {
  const [form, setForm] = useState({
    name: '', type: 'static', target_url: '', agent_id: '',
    target_rate_mbps: 10, duration_sec: 60,
    distribution: 'flat', jitter_pct: 0,
    ramp_up_sec: 0, ramp_down_sec: 0,
    concurrent_fragments: 1, retries: 3,
    total_bytes_target: 0, dispatch_rate_tpm: 0,
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))

  const submit = async e => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const payload = {
        ...form,
        target_rate_mbps: +form.target_rate_mbps,
        duration_sec: +form.duration_sec,
        jitter_pct: +form.jitter_pct,
        ramp_up_sec: +form.ramp_up_sec,
        ramp_down_sec: +form.ramp_down_sec,
        concurrent_fragments: +form.concurrent_fragments,
        retries: +form.retries,
        total_bytes_target: +form.total_bytes_target,
        dispatch_rate_tpm: +form.dispatch_rate_tpm,
      }
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
          <h3 className="text-sm font-semibold text-white">New Task</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-white">✕</button>
        </div>
        <form onSubmit={submit} className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Name"><input className="input" value={form.name} onChange={e => set('name', e.target.value)} /></Field>
            <Field label="Type">
              <select className="input" value={form.type} onChange={e => set('type', e.target.value)}>
                <option value="static">Static (HTTP)</option>
                <option value="youtube">YouTube (yt-dlp)</option>
              </select>
            </Field>
          </div>
          <Field label="Target URL *"><input required className="input" value={form.target_url} onChange={e => set('target_url', e.target.value)} placeholder="https://..." /></Field>
          <Field label="Agent">
            <select className="input" value={form.agent_id} onChange={e => set('agent_id', e.target.value)}>
              <option value="">— any available —</option>
              {agents.map(a => <option key={a.id} value={a.id}>{a.hostname} ({a.ip})</option>)}
            </select>
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Target Rate (Mbps)"><input type="number" step="0.1" min="0" className="input" value={form.target_rate_mbps} onChange={e => set('target_rate_mbps', e.target.value)} /></Field>
            <Field label="Duration (sec)"><input type="number" min="0" className="input" value={form.duration_sec} onChange={e => set('duration_sec', e.target.value)} /></Field>
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
          {form.type === 'youtube' && (
            <div className="grid grid-cols-2 gap-4">
              <Field label="Concurrent Fragments"><input type="number" min="1" className="input" value={form.concurrent_fragments} onChange={e => set('concurrent_fragments', e.target.value)} /></Field>
              <Field label="Retries"><input type="number" min="0" className="input" value={form.retries} onChange={e => set('retries', e.target.value)} /></Field>
            </div>
          )}
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <div className="flex gap-2 justify-end pt-2">
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">{loading ? 'Creating...' : 'Create Task'}</button>
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
