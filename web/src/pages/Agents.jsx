import React, { useState, useEffect } from 'react'
import { Plus, RefreshCw, ChevronDown, ChevronUp, Server, RotateCw, Trash2, Loader2 } from 'lucide-react'
import { agentsApi, credentialsApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function provisionLabel(status) {
  const map = { pending: 'provisioning', running: 'provisioning', success: 'online', failed: 'failed' }
  return map[status] || status
}

export default function Agents() {
  const [agents,      setAgents]      = useState([])
  const [jobs,        setJobs]        = useState([])
  const [creds,       setCreds]       = useState([])
  const [showModal,   setShowModal]   = useState(false)
  const [error,       setError]       = useState(null)
  const [expandedRow, setExpandedRow] = useState(null)
  const [retrying,    setRetrying]    = useState(null)
  const [deleting,    setDeleting]    = useState(null)

  const reload = async () => {
    try {
      const [a, j, c] = await Promise.all([
        agentsApi.list(), agentsApi.listProvisionJobs(), credentialsApi.list(),
      ])
      setAgents(a || []); setJobs(j || []); setCreds(c || [])
    } catch (e) { setError(e.message) }
  }

  useEffect(() => { reload(); const t = setInterval(reload, 10000); return () => clearInterval(t) }, [])

  const agentIPs = new Set(agents.map(a => a.ip))
  const rows = [
    ...agents.map(a => ({ type: 'agent', key: a.id, agent: a, job: jobs.find(j => j.agent_id === a.id) })),
    ...jobs
      .filter(j => !j.agent_id && !agentIPs.has(j.host_ip) && (j.status === 'failed' || j.status === 'pending' || j.status === 'running'))
      .map(j => ({ type: 'provision', key: j.id, agent: null, job: j })),
  ]

  const handleRetry = async (jobId) => {
    setRetrying(jobId)
    try { await agentsApi.retryProvisionJob(jobId); await reload() }
    catch (err) { setError(err.message) }
    finally { setRetrying(null) }
  }

  const handleDeleteAgent = async (id) => {
    if (!confirm('Delete this agent?')) return
    setDeleting(id)
    try { await agentsApi.delete(id); await reload() }
    catch (err) { setError(err.message) }
    finally { setDeleting(null) }
  }

  const handleDeleteJob = async (jobId) => {
    if (!confirm('Delete this provision job?')) return
    setDeleting(jobId)
    try { await agentsApi.deleteProvisionJob(jobId); await reload() }
    catch (err) { setError(err.message) }
    finally { setDeleting(null) }
  }

  const existingIPs = new Set([
    ...agents.map(a => a.ip),
    ...jobs.filter(j => j.status === 'pending' || j.status === 'running').map(j => j.host_ip),
  ])

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <span className="label" style={{ display: 'block', marginBottom: 6 }}>Infrastructure</span>
          <h1 className="page-title">Agents</h1>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={reload} className="btn-secondary" style={{ padding: '9px 12px' }}>
            <RefreshCw size={14} />
          </button>
          <button onClick={() => setShowModal(true)} className="btn-primary">
            <Plus size={14} /> Add Agent
          </button>
        </div>
      </div>

      {error && (
        <div className="error-bar" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          {error}
          <button onClick={() => setError(null)} style={{ background: 'none', border: 'none', color: 'var(--red)', cursor: 'pointer' }}>✕</button>
        </div>
      )}

      <div className="card" style={{ overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead className="tbl-head">
            <tr>
              <th style={{ width: 36 }}></th>
              <th>Host / Hostname</th>
              <th>IP Address</th>
              <th>Status</th>
              <th>Step</th>
              <th>Rate</th>
              <th>Last Seen</th>
              <th style={{ width: 120 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr><td colSpan={8}><div className="empty">No agents yet — click "Add Agent" to provision one</div></td></tr>
            ) : rows.map(row => {
              const isExpanded    = expandedRow === row.key
              const isFailed      = row.job?.status === 'failed'
              const isProvisioning = row.job?.status === 'running' || row.job?.status === 'pending'
              const clickable     = !!row.job?.log

              return (
                <React.Fragment key={row.key}>
                  <tr
                    className="tbl-row"
                    style={{
                      cursor: clickable ? 'pointer' : 'default',
                      background: isFailed ? 'rgba(181,60,43,0.03)' : undefined,
                    }}
                    onClick={() => clickable && setExpandedRow(isExpanded ? null : row.key)}
                  >
                    <td style={{ padding: '13px 12px' }}>
                      {clickable
                        ? (isExpanded
                          ? <ChevronUp  size={14} style={{ color: 'var(--text-muted)' }} />
                          : <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} />)
                        : <Server size={14} style={{ color: 'var(--text-muted)' }} />
                      }
                    </td>
                    <td>
                      {row.agent ? (
                        <div>
                          <span style={{ fontWeight: 500 }}>{row.agent.hostname}</span>
                          <span className="mono" style={{ color: 'var(--text-muted)', marginLeft: 8, fontSize: 11 }}>
                            {row.agent.id.slice(0, 8)}
                          </span>
                        </div>
                      ) : (
                        <span style={{ color: 'var(--text-dim)' }}>{row.job?.host_ip}</span>
                      )}
                    </td>
                    <td>
                      <span className="mono" style={{ color: 'var(--text-dim)', fontSize: 12 }}>
                        {row.agent?.ip || row.job?.host_ip}
                      </span>
                    </td>
                    <td>
                      {row.agent
                        ? <Badge label={row.agent.status} />
                        : <Badge label={provisionLabel(row.job?.status)} />}
                    </td>
                    <td>
                      {isProvisioning ? (
                        <span style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: 'var(--amber)' }}>
                          <Loader2 size={12} style={{ animation: 'spin 1.2s linear infinite' }} />
                          {row.job?.current_step}
                        </span>
                      ) : isFailed ? (
                        <span style={{ fontSize: 12, color: 'var(--red)' }}>
                          {row.job?.failed_step || row.job?.current_step}
                        </span>
                      ) : (
                        <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>—</span>
                      )}
                    </td>
                    <td>
                      {row.agent ? (
                        <span className="mono" style={{ color: 'var(--green)', fontWeight: 500, fontSize: 12 }}>
                          {row.agent.current_rate_mbps.toFixed(2)} Mbps
                        </span>
                      ) : <span style={{ color: 'var(--text-muted)' }}>—</span>}
                    </td>
                    <td>
                      <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
                        {row.agent ? fmtDate(row.agent.last_heartbeat) : fmtDate(row.job?.created_at)}
                      </span>
                    </td>
                    <td onClick={e => e.stopPropagation()}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                        {isFailed && (
                          <button
                            onClick={() => handleRetry(row.job.id)}
                            disabled={retrying === row.job.id}
                            style={{
                              display: 'flex', alignItems: 'center', gap: 4,
                              padding: '5px 10px', borderRadius: 6,
                              background: 'var(--amber-dim)',
                              border: '1px solid rgba(157,104,32,0.2)',
                              color: 'var(--amber)', fontSize: 12, fontWeight: 500,
                              cursor: 'pointer', transition: 'opacity 0.12s',
                            }}
                          >
                            <RotateCw size={11} style={{ animation: retrying === row.job.id ? 'spin 1.2s linear infinite' : 'none' }} />
                            Retry
                          </button>
                        )}
                        {(row.agent || row.job) && (
                          <button
                            onClick={() => row.agent ? handleDeleteAgent(row.agent.id) : handleDeleteJob(row.job.id)}
                            disabled={deleting === (row.agent?.id || row.job?.id)}
                            style={{
                              padding: '6px', borderRadius: 6, background: 'none', border: 'none',
                              color: 'var(--text-muted)', cursor: 'pointer', transition: 'color 0.12s',
                            }}
                            onMouseEnter={e => e.currentTarget.style.color = 'var(--red)'}
                            onMouseLeave={e => e.currentTarget.style.color = 'var(--text-muted)'}
                          >
                            <Trash2 size={14} />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                  {isExpanded && row.job?.log && (
                    <tr>
                      <td colSpan={8} style={{ padding: '0 16px 16px', borderBottom: '1px solid var(--border)' }}>
                        <pre style={{
                          background: 'var(--bg)',
                          border: '1px solid var(--border)',
                          borderRadius: 8,
                          padding: '12px 16px',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-dim)',
                          overflow: 'auto',
                          maxHeight: 200,
                          whiteSpace: 'pre-wrap',
                          lineHeight: 1.7,
                        }}>
                          {row.job.log}
                        </pre>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              )
            })}
          </tbody>
        </table>
      </div>

      {showModal && (
        <ProvisionModal
          creds={creds}
          existingIPs={existingIPs}
          onClose={() => setShowModal(false)}
          onSuccess={() => { setShowModal(false); reload() }}
        />
      )}
    </div>
  )
}

function ProvisionModal({ creds, existingIPs, onClose, onSuccess }) {
  const [form, setForm]       = useState({ host_ip: '', ssh_port: 22, ssh_user: 'root', auth_type: 'key', credential_ref: '' })
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState(null)
  const ipDuplicate = form.host_ip && existingIPs.has(form.host_ip)

  const submit = async e => {
    e.preventDefault()
    if (ipDuplicate) { setError(`IP ${form.host_ip} already exists`); return }
    setLoading(true); setError(null)
    try { await agentsApi.provision({ ...form, ssh_port: +form.ssh_port }); onSuccess() }
    catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }

  return (
    <Modal title="Add Agent via SSH" onClose={onClose}>
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
          <Field label="Host IP">
            <input required className="input" value={form.host_ip}
              onChange={e => setForm(f => ({ ...f, host_ip: e.target.value }))}
              style={ipDuplicate ? { borderColor: 'var(--red)' } : {}} />
            {ipDuplicate && <span style={{ fontSize: 11, color: 'var(--red)' }}>This IP already exists</span>}
          </Field>
          <Field label="SSH Port">
            <input type="number" className="input" value={form.ssh_port}
              onChange={e => setForm(f => ({ ...f, ssh_port: e.target.value }))} />
          </Field>
        </div>
        <Field label="SSH User">
          <input required className="input" value={form.ssh_user}
            onChange={e => setForm(f => ({ ...f, ssh_user: e.target.value }))} />
        </Field>
        <Field label="Auth Type">
          <select className="input" value={form.auth_type}
            onChange={e => setForm(f => ({ ...f, auth_type: e.target.value }))}>
            <option value="key">SSH Key</option>
            <option value="password">Password</option>
          </select>
        </Field>
        <Field label="Credential">
          <select required className="input" value={form.credential_ref}
            onChange={e => setForm(f => ({ ...f, credential_ref: e.target.value }))}>
            <option value="">— Select credential —</option>
            {creds.map(c => <option key={c.id} value={c.id}>{c.name} ({c.type})</option>)}
          </select>
        </Field>
        {error && <div className="error-bar">{error}</div>}
        <ModalFooter onClose={onClose} loading={loading} disabled={ipDuplicate} label="Provision" />
      </form>
    </Modal>
  )
}

/* ── Shared ── */
function Modal({ title, onClose, children }) {
  return (
    <div
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(61,57,41,0.4)',
        backdropFilter: 'blur(4px)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 50, padding: 20,
      }}
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div style={{
        background: 'var(--elevated)',
        border: '1px solid var(--border)',
        borderRadius: 14,
        width: '100%', maxWidth: 460,
        boxShadow: 'var(--shadow-lg)',
      }}>
        <div style={{
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          padding: '18px 22px', borderBottom: '1px solid var(--border)',
        }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            {title}
          </h3>
          <button onClick={onClose} style={{
            background: 'none', border: 'none', cursor: 'pointer',
            color: 'var(--text-muted)', fontSize: 18, lineHeight: 1,
            width: 28, height: 28, borderRadius: 6,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>✕</button>
        </div>
        <div style={{ padding: '20px 22px' }}>{children}</div>
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

function ModalFooter({ onClose, loading, disabled, label = 'Submit' }) {
  return (
    <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', paddingTop: 4 }}>
      <button type="button" onClick={onClose} className="btn-secondary">Cancel</button>
      <button type="submit" disabled={loading || disabled} className="btn-primary">
        {loading ? 'Processing...' : label}
      </button>
    </div>
  )
}
