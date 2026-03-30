import React, { useState, useEffect } from 'react'
import { Plus, Trash2, Activity } from 'lucide-react'
import { profilesApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) { if (!iso) return '—'; return new Date(iso).toLocaleString() }

function parsePoints(raw) {
  try { return JSON.parse(raw || '[]') } catch { return [] }
}

function MiniCurve({ points, width = 120, height = 32 }) {
  if (!points || points.length < 2) return null
  const maxPct = Math.max(...points.map(p => p.rate_pct), 100)
  const maxSec = points[points.length - 1].offset_sec || 1
  const coords = points.map(p =>
    `${(p.offset_sec / maxSec) * width},${height - (p.rate_pct / maxPct) * (height - 4)}`
  ).join(' ')
  return (
    <svg width={width} height={height} style={{ display: 'block' }}>
      <polyline points={coords} fill="none" stroke="var(--accent)" strokeWidth="1.5" strokeLinejoin="round" />
    </svg>
  )
}

export default function Profiles() {
  const [profiles, setProfiles] = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error, setError] = useState(null)

  const reload = async () => {
    try { setProfiles((await profilesApi.list()) || []) }
    catch (e) { setError(e.message) }
  }

  useEffect(() => { reload() }, [])

  const handleDelete = async (id) => {
    if (!confirm('Delete this traffic profile?')) return
    try { await profilesApi.delete(id); await reload() }
    catch (err) { setError(err.message) }
  }

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <span className="label" style={{ display: 'block', marginBottom: 6 }}>Configuration</span>
          <h1 className="page-title">Traffic Profiles</h1>
        </div>
        <button onClick={() => setShowModal(true)} className="btn-primary">
          <Plus size={14} /> New Profile
        </button>
      </div>

      {error && (
        <div className="error-bar" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {error}
          <button onClick={() => setError(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--red)' }}>&#x2715;</button>
        </div>
      )}

      <div className="card" style={{ overflow: 'hidden' }}>
        {profiles.length === 0 ? (
          <div className="empty">No traffic profiles yet — create one to shape task rate curves</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead className="tbl-head">
              <tr>
                <th>Name</th><th>Distribution</th><th>Curve</th><th>Points</th><th>Created</th><th style={{ width: 50 }}></th>
              </tr>
            </thead>
            <tbody>
              {profiles.map(p => {
                const pts = parsePoints(p.points)
                return (
                  <tr key={p.id} className="tbl-row">
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Activity size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                        <div>
                          <div style={{ fontWeight: 500 }}>{p.name}</div>
                          {p.description && (
                            <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>{p.description}</div>
                          )}
                        </div>
                      </div>
                    </td>
                    <td><Badge label={p.distribution || 'diurnal'} /></td>
                    <td><MiniCurve points={pts} /></td>
                    <td><span className="mono" style={{ fontSize: 12 }}>{pts.length}</span></td>
                    <td><span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{fmtDate(p.created_at)}</span></td>
                    <td>
                      <button
                        onClick={() => handleDelete(p.id)}
                        style={{
                          padding: '6px', borderRadius: 6, background: 'none', border: 'none',
                          color: 'var(--text-muted)', cursor: 'pointer', transition: 'color 0.12s',
                        }}
                        onMouseEnter={e => e.currentTarget.style.color = 'var(--red)'}
                        onMouseLeave={e => e.currentTarget.style.color = 'var(--text-muted)'}
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <ProfileModal onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />
      )}
    </div>
  )
}

function ProfileModal({ onClose, onSuccess }) {
  const [form, setForm] = useState({ name: '', description: '', distribution: 'diurnal' })
  const [points, setPoints] = useState([
    { offset_sec: 0, rate_pct: 50 },
    { offset_sec: 3600, rate_pct: 100 },
  ])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const addPoint = () => {
    const lastSec = points.length > 0 ? points[points.length - 1].offset_sec : 0
    setPoints([...points, { offset_sec: lastSec + 600, rate_pct: 50 }])
  }

  const removePoint = (idx) => setPoints(points.filter((_, i) => i !== idx))

  const updatePoint = (idx, field, value) => {
    setPoints(points.map((p, i) => i === idx ? { ...p, [field]: +value } : p))
  }

  const submit = async e => {
    e.preventDefault(); setLoading(true); setError(null)
    try {
      const sorted = [...points].sort((a, b) => a.offset_sec - b.offset_sec)
      await profilesApi.create({
        ...form,
        points: JSON.stringify(sorted),
      })
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
        borderRadius: 14, width: '100%', maxWidth: 520,
        boxShadow: 'var(--shadow-lg)', marginTop: 24,
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            New Traffic Profile
          </h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 18 }}>&#x2715;</button>
        </div>

        <form onSubmit={submit} style={{ padding: '20px 22px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Field label="Name">
            <input required className="input" value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          </Field>
          <Field label="Description">
            <input className="input" value={form.description}
              onChange={e => setForm(f => ({ ...f, description: e.target.value }))} />
          </Field>
          <Field label="Distribution">
            <select className="input" value={form.distribution}
              onChange={e => setForm(f => ({ ...f, distribution: e.target.value }))}>
              <option value="diurnal">Diurnal</option>
              <option value="ramp">Ramp</option>
              <option value="flat">Flat</option>
            </select>
          </Field>

          <Field label="Curve Points">
            <div style={{
              border: '1px solid var(--border)', borderRadius: 8, background: 'var(--bg)',
              overflow: 'hidden',
            }}>
              <div style={{
                display: 'grid', gridTemplateColumns: '1fr 1fr 36px',
                padding: '8px 12px', fontSize: 11, fontWeight: 600,
                color: 'var(--text-muted)', borderBottom: '1px solid var(--border)',
                textTransform: 'uppercase', letterSpacing: '0.05em',
              }}>
                <span>Offset (sec)</span><span>Rate (%)</span><span></span>
              </div>
              {points.map((pt, i) => (
                <div key={i} style={{
                  display: 'grid', gridTemplateColumns: '1fr 1fr 36px', gap: 8,
                  padding: '6px 12px', borderBottom: '1px solid var(--border)',
                  alignItems: 'center',
                }}>
                  <input type="number" min="0" className="input" style={{ padding: '6px 8px', fontSize: 12 }}
                    value={pt.offset_sec} onChange={e => updatePoint(i, 'offset_sec', e.target.value)} />
                  <input type="number" min="0" max="100" className="input" style={{ padding: '6px 8px', fontSize: 12 }}
                    value={pt.rate_pct} onChange={e => updatePoint(i, 'rate_pct', e.target.value)} />
                  <button type="button" onClick={() => removePoint(i)} disabled={points.length <= 2}
                    style={{
                      padding: 4, borderRadius: 4, background: 'none', border: 'none',
                      color: points.length <= 2 ? 'var(--border)' : 'var(--text-muted)',
                      cursor: points.length <= 2 ? 'default' : 'pointer',
                    }}>
                    <Trash2 size={12} />
                  </button>
                </div>
              ))}
              <button type="button" onClick={addPoint}
                style={{
                  width: '100%', padding: '8px', background: 'none', border: 'none',
                  color: 'var(--accent)', cursor: 'pointer', fontSize: 12, fontWeight: 500,
                }}>
                + Add Point
              </button>
            </div>
          </Field>

          {points.length >= 2 && (
            <MiniCurve points={[...points].sort((a, b) => a.offset_sec - b.offset_sec)} width={476} height={60} />
          )}

          {error && <div className="error-bar">{error}</div>}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 4 }}>
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">
              {loading ? 'Creating...' : 'Create Profile'}
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
