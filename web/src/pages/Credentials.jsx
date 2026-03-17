import React, { useState, useEffect } from 'react'
import { Plus, Trash2, Key } from 'lucide-react'
import { credentialsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) { if (!iso) return '—'; return new Date(iso).toLocaleString() }

export default function Credentials() {
  const [creds,     setCreds]     = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error,     setError]     = useState(null)

  const reload = async () => {
    try { setCreds((await credentialsApi.list()) || []) }
    catch (e) { setError(e.message) }
  }

  useEffect(() => { reload() }, [])

  const handleDelete = async (id) => {
    if (!confirm('Delete this credential?')) return
    try { await credentialsApi.delete(id); await reload() }
    catch (err) { setError(err.message) }
  }

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <span className="label" style={{ display: 'block', marginBottom: 6 }}>Security</span>
          <h1 className="page-title">Credentials</h1>
        </div>
        <button onClick={() => setShowModal(true)} className="btn-primary">
          <Plus size={14} /> Add Credential
        </button>
      </div>

      {error && (
        <div className="error-bar" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {error}
          <button onClick={() => setError(null)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--red)' }}>✕</button>
        </div>
      )}

      <div className="card" style={{ overflow: 'hidden' }}>
        {creds.length === 0 ? (
          <div className="empty">No credentials yet — add one to start provisioning agents</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead className="tbl-head">
              <tr>
                <th>Name</th><th>Type</th><th>Created</th><th style={{ width: 50 }}></th>
              </tr>
            </thead>
            <tbody>
              {creds.map(c => (
                <tr key={c.id} className="tbl-row">
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Key size={13} style={{ color: 'var(--text-muted)', flexShrink: 0 }} />
                      <span style={{ fontWeight: 500 }}>{c.name}</span>
                    </div>
                  </td>
                  <td><Badge label={c.type === 'key' ? 'SSH Key' : 'Password'} /></td>
                  <td><span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{fmtDate(c.created_at)}</span></td>
                  <td>
                    <button
                      onClick={() => handleDelete(c.id)}
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
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showModal && (
        <CredentialModal onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />
      )}
    </div>
  )
}

function CredentialModal({ onClose, onSuccess }) {
  const [form, setForm]       = useState({ name: '', type: 'key', payload: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState(null)

  const submit = async e => {
    e.preventDefault(); setLoading(true)
    try { await credentialsApi.create(form); onSuccess() }
    catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }

  return (
    <div
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(61,57,41,0.4)', backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 50, padding: 20,
      }}
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div style={{
        background: 'var(--elevated)', border: '1px solid var(--border)',
        borderRadius: 14, width: '100%', maxWidth: 440,
        boxShadow: 'var(--shadow-lg)',
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            Add Credential
          </h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 18 }}>✕</button>
        </div>
        <form onSubmit={submit} style={{ padding: '20px 22px', display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Field label="Name">
            <input required className="input" value={form.name}
              onChange={e => setForm(f => ({ ...f, name: e.target.value }))} />
          </Field>
          <Field label="Type">
            <select className="input" value={form.type}
              onChange={e => setForm(f => ({ ...f, type: e.target.value }))}>
              <option value="key">SSH Private Key</option>
              <option value="password">Password</option>
            </select>
          </Field>
          <Field label={form.type === 'key' ? 'Private Key (PEM)' : 'Password'}>
            {form.type === 'key' ? (
              <textarea
                required rows={7} className="input"
                style={{ fontFamily: 'var(--font-mono)', fontSize: 12, resize: 'vertical', lineHeight: 1.6 }}
                value={form.payload}
                onChange={e => setForm(f => ({ ...f, payload: e.target.value }))}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..."
              />
            ) : (
              <input required type="password" className="input"
                value={form.payload}
                onChange={e => setForm(f => ({ ...f, payload: e.target.value }))} />
            )}
          </Field>
          {error && <div className="error-bar">{error}</div>}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 4 }}>
            <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={loading} className="btn-primary">
              {loading ? 'Saving...' : 'Save'}
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
