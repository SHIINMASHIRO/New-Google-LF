import React, { useEffect, useState } from 'react'
import { Plus, RefreshCw, Trash2, Pencil } from 'lucide-react'
import { urlPoolsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function parseURLs(text) {
  return text
    .split(/\r?\n/)
    .map(x => x.trim())
    .filter(Boolean)
}

export default function URLPools() {
  const [pools, setPools] = useState([])
  const [error, setError] = useState(null)
  const [showModal, setShowModal] = useState(false)
  const [editingPool, setEditingPool] = useState(null)

  const reload = async () => {
    try {
      const list = await urlPoolsApi.list()
      setPools(list || [])
      setError(null)
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => {
    reload()
    const t = setInterval(reload, 5000)
    return () => clearInterval(t)
  }, [])

  const removePool = async (id) => {
    if (!window.confirm('Delete this URL pool?')) return
    try {
      await urlPoolsApi.delete(id)
      reload()
    } catch (e) {
      alert(e.message)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">URL Pools</h1>
          <p className="text-sm text-gray-500 mt-0.5">{pools.length} total pools</p>
        </div>
        <div className="flex gap-2">
          <button onClick={reload} className="p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors">
            <RefreshCw size={15} />
          </button>
          <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm transition-colors">
            <Plus size={15} /> New Pool
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
              <th className="px-4 py-3 text-left font-medium">URLs</th>
              <th className="px-4 py-3 text-left font-medium">Created</th>
              <th className="px-4 py-3 text-left font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800/50">
            {pools.length === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-600">No URL pools yet.</td></tr>
            ) : pools.map(pool => (
              <tr key={pool.id} className="hover:bg-gray-800/30 transition-colors">
                <td className="px-4 py-3">
                  <div className="text-white font-medium">{pool.name}</div>
                  <div className="text-xs text-gray-500 mt-0.5">{pool.description || '—'}</div>
                </td>
                <td className="px-4 py-3"><Badge label={pool.type} /></td>
                <td className="px-4 py-3 text-xs text-gray-400">{pool.urls?.length || 0}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{fmtDate(pool.created_at)}</td>
                <td className="px-4 py-3">
                  <button onClick={() => setEditingPool(pool)} className="p-1.5 rounded bg-blue-500/20 text-blue-400 hover:bg-blue-500/30 transition-colors mr-2">
                    <Pencil size={12} />
                  </button>
                  <button onClick={() => removePool(pool.id)} className="p-1.5 rounded bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors">
                    <Trash2 size={12} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && <URLPoolModal onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />}
      {editingPool && <URLPoolModal pool={editingPool} onClose={() => setEditingPool(null)} onSuccess={() => { setEditingPool(null); reload() }} />}
    </div>
  )
}

function URLPoolModal({ pool, onClose, onSuccess }) {
  const isEditing = !!pool
  const [form, setForm] = useState({
    name: pool?.name || '',
    type: pool?.type || 'youtube',
    description: pool?.description || '',
    urlsText: (pool?.urls || []).join('\n'),
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))

  const submit = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      const payload = {
        name: form.name,
        type: form.type,
        description: form.description,
        urls: parseURLs(form.urlsText),
      }
      if (isEditing) {
        await urlPoolsApi.update(pool.id, payload)
      } else {
        await urlPoolsApi.create(payload)
      }
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
      <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-2xl shadow-2xl my-4">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
          <h3 className="text-sm font-semibold text-white">{isEditing ? 'Edit URL Pool' : 'New URL Pool'}</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-white">✕</button>
        </div>
        <form onSubmit={submit} className="p-5 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Name *"><input required className="input" value={form.name} onChange={e => set('name', e.target.value)} /></Field>
            <Field label="Type">
              <select className="input" value={form.type} onChange={e => set('type', e.target.value)}>
                <option value="youtube">YouTube</option>
                <option value="static">Static</option>
              </select>
            </Field>
          </div>
          <Field label="Description"><input className="input" value={form.description} onChange={e => set('description', e.target.value)} /></Field>
          <Field label="URLs *">
            <textarea
              required
              rows={10}
              className="input min-h-[220px] font-mono text-xs"
              value={form.urlsText}
              onChange={e => set('urlsText', e.target.value)}
              placeholder={form.type === 'youtube' ? 'One YouTube URL per line' : 'One static URL per line'}
            />
          </Field>
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <div className="flex gap-2 justify-end pt-2">
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">{loading ? (isEditing ? 'Saving...' : 'Creating...') : (isEditing ? 'Save Pool' : 'Create Pool')}</button>
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
