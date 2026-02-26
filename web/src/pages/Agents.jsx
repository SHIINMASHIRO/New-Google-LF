import React, { useState, useEffect } from 'react'
import { Plus, RefreshCw, ChevronDown, ChevronUp, Server } from 'lucide-react'
import { agentsApi, credentialsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

export default function Agents() {
  const [agents, setAgents] = useState([])
  const [jobs, setJobs] = useState([])
  const [creds, setCreds] = useState([])
  const [showModal, setShowModal] = useState(false)
  const [showCredModal, setShowCredModal] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [expandedJob, setExpandedJob] = useState(null)

  const reload = async () => {
    try {
      const [a, j, c] = await Promise.all([
        agentsApi.list(),
        agentsApi.listProvisionJobs(),
        credentialsApi.list(),
      ])
      setAgents(a || [])
      setJobs(j || [])
      setCreds(c || [])
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => { reload(); const t = setInterval(reload, 10000); return () => clearInterval(t) }, [])

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">Agents</h1>
          <p className="text-sm text-gray-500 mt-0.5">{agents.length} registered agents</p>
        </div>
        <div className="flex gap-2">
          <button onClick={reload} className="p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors">
            <RefreshCw size={15} />
          </button>
          <button onClick={() => setShowCredModal(true)} className="px-3 py-2 rounded-lg bg-gray-800 text-gray-300 text-sm hover:bg-gray-700 transition-colors">
            + Credential
          </button>
          <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-sm transition-colors">
            <Plus size={15} /> Add Agent
          </button>
        </div>
      </div>

      {error && <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{error}</div>}

      {/* Agent Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="border-b border-gray-800 bg-gray-800/50">
            <tr className="text-gray-400 text-xs">
              <th className="px-4 py-3 text-left font-medium">ID</th>
              <th className="px-4 py-3 text-left font-medium">Hostname</th>
              <th className="px-4 py-3 text-left font-medium">IP</th>
              <th className="px-4 py-3 text-left font-medium">Status</th>
              <th className="px-4 py-3 text-left font-medium">Rate</th>
              <th className="px-4 py-3 text-left font-medium">Last Seen</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800/50">
            {agents.length === 0 ? (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-600">
                No agents yet. Click "Add Agent" to provision one.
              </td></tr>
            ) : agents.map(a => (
              <tr key={a.id} className="hover:bg-gray-800/30 transition-colors">
                <td className="px-4 py-3 font-mono text-xs text-gray-500">{a.id.slice(0, 8)}</td>
                <td className="px-4 py-3 text-white">{a.hostname}</td>
                <td className="px-4 py-3 text-gray-400">{a.ip}</td>
                <td className="px-4 py-3"><Badge label={a.status} /></td>
                <td className="px-4 py-3 font-mono text-green-400">{a.current_rate_mbps.toFixed(2)} Mbps</td>
                <td className="px-4 py-3 text-gray-500 text-xs">{fmtDate(a.last_heartbeat)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Provision Jobs */}
      {jobs.length > 0 && (
        <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
          <div className="px-4 py-3 border-b border-gray-800">
            <h2 className="text-sm font-medium text-white">Provision Jobs</h2>
          </div>
          <div className="divide-y divide-gray-800/50">
            {jobs.map(j => (
              <div key={j.id}>
                <div
                  className="flex items-center gap-4 px-4 py-3 cursor-pointer hover:bg-gray-800/30"
                  onClick={() => setExpandedJob(expandedJob === j.id ? null : j.id)}
                >
                  <Server size={14} className="text-gray-500 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-white text-sm">{j.host_ip}</span>
                      <Badge label={j.status} />
                    </div>
                    <p className="text-xs text-gray-500 mt-0.5">Step: {j.current_step}</p>
                  </div>
                  <span className="text-xs text-gray-600">{fmtDate(j.created_at)}</span>
                  {expandedJob === j.id ? <ChevronUp size={14} className="text-gray-500" /> : <ChevronDown size={14} className="text-gray-500" />}
                </div>
                {expandedJob === j.id && (
                  <div className="px-4 pb-3">
                    <pre className="bg-gray-950 rounded-lg p-3 text-xs text-gray-400 font-mono overflow-auto max-h-48 whitespace-pre-wrap">
                      {j.log || '(no log yet)'}
                    </pre>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Provision Modal */}
      {showModal && <ProvisionModal creds={creds} onClose={() => setShowModal(false)} onSuccess={() => { setShowModal(false); reload() }} />}
      {showCredModal && <CredentialModal onClose={() => setShowCredModal(false)} onSuccess={() => { setShowCredModal(false); reload() }} />}
    </div>
  )
}

function ProvisionModal({ creds, onClose, onSuccess }) {
  const [form, setForm] = useState({ host_ip: '', ssh_port: 22, ssh_user: 'root', auth_type: 'key', credential_ref: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const submit = async e => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    try {
      await agentsApi.provision({ ...form, ssh_port: +form.ssh_port })
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal title="Add Agent via SSH" onClose={onClose}>
      <form onSubmit={submit} className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Host IP" required><input required className="input" value={form.host_ip} onChange={e => setForm(f => ({ ...f, host_ip: e.target.value }))} /></Field>
          <Field label="SSH Port"><input type="number" className="input" value={form.ssh_port} onChange={e => setForm(f => ({ ...f, ssh_port: e.target.value }))} /></Field>
        </div>
        <Field label="SSH User"><input required className="input" value={form.ssh_user} onChange={e => setForm(f => ({ ...f, ssh_user: e.target.value }))} /></Field>
        <Field label="Auth Type">
          <select className="input" value={form.auth_type} onChange={e => setForm(f => ({ ...f, auth_type: e.target.value }))}>
            <option value="key">SSH Key</option>
            <option value="password">Password</option>
          </select>
        </Field>
        <Field label="Credential">
          <select required className="input" value={form.credential_ref} onChange={e => setForm(f => ({ ...f, credential_ref: e.target.value }))}>
            <option value="">— select credential —</option>
            {creds.map(c => <option key={c.id} value={c.id}>{c.name} ({c.type})</option>)}
          </select>
        </Field>
        {error && <p className="text-red-400 text-sm">{error}</p>}
        <div className="flex gap-2 justify-end pt-2">
          <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
          <button type="submit" disabled={loading} className="btn-primary">{loading ? 'Provisioning...' : 'Provision'}</button>
        </div>
      </form>
    </Modal>
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
