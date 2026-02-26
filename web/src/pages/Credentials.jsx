import React, { useState, useEffect } from 'react'
import { Plus, Trash2, Key } from 'lucide-react'
import { credentialsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export default function Credentials() {
  const [creds, setCreds] = useState([])
  const [showModal, setShowModal] = useState(false)
  const [error, setError] = useState(null)

  const reload = async () => {
    try {
      const c = await credentialsApi.list()
      setCreds(c || [])
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => { reload() }, [])

  const handleDelete = async (id) => {
    if (!confirm('Are you sure you want to delete this credential?')) return
    try {
      await credentialsApi.delete(id)
      await reload()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">Credentials</h1>
          <p className="text-sm text-gray-500 mt-0.5">{creds.length} stored credentials</p>
        </div>
        <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm transition-colors">
          <Plus size={15} /> Add Credential
        </button>
      </div>

      {error && <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{error}<button onClick={() => setError(null)} className="ml-2 text-red-500 hover:text-red-300">✕</button></div>}

      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        {creds.length === 0 ? (
          <div className="px-4 py-12 text-center text-gray-600 text-sm">
            No credentials yet. Add one to start provisioning agents.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-b border-gray-800 bg-gray-800/50">
              <tr className="text-gray-400 text-xs">
                <th className="px-4 py-3 text-left font-medium">Name</th>
                <th className="px-4 py-3 text-left font-medium">Type</th>
                <th className="px-4 py-3 text-left font-medium">Created</th>
                <th className="px-4 py-3 text-left font-medium w-16"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800/50">
              {creds.map(c => (
                <tr key={c.id} className="hover:bg-gray-800/30 transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <Key size={14} className="text-gray-500" />
                      <span className="text-white">{c.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <Badge label={c.type === 'key' ? 'SSH Key' : 'Password'} />
                  </td>
                  <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(c.created_at)}</td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleDelete(c.id)}
                      className="p-1 rounded text-gray-600 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                      title="Delete credential"
                    >
                      <Trash2 size={13} />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {showModal && <CredentialModal onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />}
    </div>
  )
}

function CredentialModal({ onClose, onSuccess }) {
  const [form, setForm] = useState({ name: '', type: 'key', payload: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const submit = async e => {
    e.preventDefault()
    setLoading(true)
    try {
      await credentialsApi.create(form)
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal title="Add Credential" onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <Field label="Name"><input required className="input" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} /></Field>
        <Field label="Type">
          <select className="input" value={form.type} onChange={e => setForm(f => ({ ...f, type: e.target.value }))}>
            <option value="key">SSH Private Key</option>
            <option value="password">Password</option>
          </select>
        </Field>
        <Field label={form.type === 'key' ? 'Private Key (PEM)' : 'Password'}>
          {form.type === 'key'
            ? <textarea required className="input font-mono text-xs" rows={6} value={form.payload} onChange={e => setForm(f => ({ ...f, payload: e.target.value }))} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..." />
            : <input required type="password" className="input" value={form.payload} onChange={e => setForm(f => ({ ...f, payload: e.target.value }))} />}
        </Field>
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <div className="flex gap-2 justify-end pt-2">
          <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
          <button type="submit" disabled={loading} className="btn-primary">{loading ? 'Saving...' : 'Save'}</button>
        </div>
      </form>
    </Modal>
  )
}

function Modal({ title, onClose, children }) {
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4" onClick={e => e.target === e.currentTarget && onClose()}>
      <div className="bg-gray-900 border border-gray-700 rounded-xl w-full max-w-md shadow-2xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-gray-800">
          <h3 className="text-sm font-semibold text-white">{title}</h3>
          <button onClick={onClose} className="text-gray-500 hover:text-white">✕</button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  )
}

function Field({ label, required, children }) {
  return (
    <label className="block">
      <span className="text-xs text-gray-400 font-medium mb-1 block">{label}{required && ' *'}</span>
      {children}
    </label>
  )
}
