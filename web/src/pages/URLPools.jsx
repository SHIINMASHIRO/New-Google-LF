import React, { useEffect, useState } from 'react'
import { Plus, RefreshCw, Trash2, Pencil } from 'lucide-react'
import { urlPoolsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) { if (!iso) return '—'; return new Date(iso).toLocaleString() }
function parseURLs(text) { return text.split(/\r?\n/).map(x => x.trim()).filter(Boolean) }

export default function URLPools() {
  const [pools,       setPools]       = useState([])
  const [error,       setError]       = useState(null)
  const [showModal,   setShowModal]   = useState(false)
  const [editingPool, setEditingPool] = useState(null)

  const reload = async () => {
    try { setPools((await urlPoolsApi.list()) || []); setError(null) }
    catch (e) { setError(e.message) }
  }

  useEffect(() => { reload(); const t = setInterval(reload, 5000); return () => clearInterval(t) }, [])

  const removePool = async (id) => {
    if (!window.confirm('Delete this URL pool?')) return
    try { await urlPoolsApi.delete(id); reload() } catch (e) { alert(e.message) }
  }

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <span className="label" style={{ display: 'block', marginBottom: 6 }}>Resources</span>
          <h1 className="page-title">URL Pools</h1>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={reload} className="btn-secondary" style={{ padding: '9px 12px' }}>
            <RefreshCw size={14} />
          </button>
          <button onClick={() => setShowModal(true)} className="btn-primary">
            <Plus size={14} /> New Pool
          </button>
        </div>
      </div>

      {error && <div className="error-bar">{error}</div>}

      <div className="card" style={{ overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead className="tbl-head">
            <tr>
              <th>Name</th><th>Type</th><th>URLs</th><th>Created</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {pools.length === 0 ? (
              <tr><td colSpan={5}>
                <div className="empty">No URL pools yet — create one to start tasks</div>
              </td></tr>
            ) : pools.map(pool => (
              <tr key={pool.id} className="tbl-row">
                <td>
                  <div style={{ fontWeight: 500 }}>{pool.name}</div>
                  {pool.description && (
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{pool.description}</div>
                  )}
                </td>
                <td><Badge label={pool.type} /></td>
                <td>
                  <span className="mono" style={{ fontWeight: 500 }}>{pool.urls?.length || 0}</span>
                </td>
                <td><span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{fmtDate(pool.created_at)}</span></td>
                <td>
                  <div style={{ display: 'flex', gap: 5 }}>
                    <button
                      onClick={() => setEditingPool(pool)}
                      style={{
                        padding: '6px 8px', borderRadius: 6,
                        background: 'var(--blue-dim)', border: '1px solid rgba(45,106,173,0.2)',
                        color: 'var(--blue)', cursor: 'pointer',
                      }}
                    ><Pencil size={12} /></button>
                    <button
                      onClick={() => removePool(pool.id)}
                      style={{
                        padding: '6px 8px', borderRadius: 6, background: 'none', border: 'none',
                        color: 'var(--text-muted)', cursor: 'pointer', transition: 'color 0.12s',
                      }}
                      onMouseEnter={e => e.currentTarget.style.color = 'var(--red)'}
                      onMouseLeave={e => e.currentTarget.style.color = 'var(--text-muted)'}
                    ><Trash2 size={12} /></button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal    && <URLPoolModal onClose={() => setShowModal(false)}       onSuccess={() => { setShowModal(false);    reload() }} />}
      {editingPool  && <URLPoolModal pool={editingPool} onClose={() => setEditingPool(null)} onSuccess={() => { setEditingPool(null); reload() }} />}
    </div>
  )
}

function URLPoolModal({ pool, onClose, onSuccess }) {
  const isEditing = !!pool
  const [form, setForm] = useState({
    name:        pool?.name        || '',
    type:        pool?.type        || 'youtube',
    description: pool?.description || '',
    urlsText:    (pool?.urls || []).join('\n'),
  })
  const [loading, setLoading] = useState(false)
  const [error,   setError]   = useState(null)
  const set = (k, v) => setForm(f => ({ ...f, [k]: v }))

  const submit = async e => {
    e.preventDefault(); setLoading(true); setError(null)
    try {
      const payload = { name: form.name, type: form.type, description: form.description, urls: parseURLs(form.urlsText) }
      if (isEditing) await urlPoolsApi.update(pool.id, payload)
      else           await urlPoolsApi.create(payload)
      onSuccess()
    } catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(61,57,41,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'center',
        zIndex: 50, padding: 24, overflowY: 'auto',
      }}
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div style={{
        background: 'var(--elevated)', border: '1px solid var(--border)',
        borderRadius: 14, width: '100%', maxWidth: 560,
        boxShadow: 'var(--shadow-lg)', marginTop: 24,
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            {isEditing ? 'Edit URL Pool' : 'New URL Pool'}
          </h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 18 }}>✕</button>
        </div>
        <form onSubmit={submit} style={{ padding: '20px 22px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <Field label="Name *">
              <input required className="input" value={form.name} onChange={e => set('name', e.target.value)} />
            </Field>
            <Field label="Type">
              <select className="input" value={form.type} onChange={e => set('type', e.target.value)}>
                <option value="youtube">YouTube</option>
                <option value="static">Static</option>
              </select>
            </Field>
          </div>
          <Field label="Description">
            <input className="input" value={form.description} onChange={e => set('description', e.target.value)} />
          </Field>
          <Field label="URLs *">
            <textarea
              required rows={10}
              className="input"
              style={{ minHeight: 200, resize: 'vertical', fontFamily: 'var(--font-mono)', fontSize: 12, lineHeight: 1.6 }}
              value={form.urlsText}
              onChange={e => set('urlsText', e.target.value)}
              placeholder={form.type === 'youtube' ? 'One YouTube URL per line' : 'One static URL per line'}
            />
          </Field>
          {error && <div className="error-bar">{error}</div>}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 4 }}>
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">
              {loading ? (isEditing ? 'Saving...' : 'Creating...') : (isEditing ? 'Save Pool' : 'Create Pool')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

function Field({ label, children }) {
  return <div className="field"><label className="field-label">{label}</label>{children}</div>
}
