import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, StopCircle, Play } from 'lucide-react'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend
} from 'recharts'
import { tasksApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) { if (!iso) return '—'; return new Date(iso).toLocaleString() }
function fmtBytes(b) {
  if (!b) return '0 B'
  if (b > 1e9) return (b / 1e9).toFixed(2) + ' GB'
  if (b > 1e6) return (b / 1e6).toFixed(2) + ' MB'
  if (b > 1e3) return (b / 1e3).toFixed(2) + ' KB'
  return b + ' B'
}

function aggregateMetrics(samples) {
  const buckets = new Map()
  for (const s of samples || []) {
    const key = new Date(s.recorded_at).toISOString().slice(0, 19)
    const cur = buckets.get(key) || {
      ts: new Date(s.recorded_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
      rate5s: 0, rate30s: 0,
    }
    cur.rate5s  += s.rate_mbps_5s  || 0
    cur.rate30s += s.rate_mbps_30s || 0
    buckets.set(key, cur)
  }
  return Array.from(buckets.values()).map(item => ({
    ...item, rate5s: +item.rate5s.toFixed(2), rate30s: +item.rate30s.toFixed(2),
  }))
}

const tooltipStyle = {
  contentStyle: {
    background: '#fff', border: '1px solid var(--border)',
    borderRadius: 10, boxShadow: 'var(--shadow-md)',
    fontFamily: 'var(--font-ui)', fontSize: 12, padding: '10px 14px',
  },
  labelStyle: { color: 'var(--text-muted)', marginBottom: 4, fontWeight: 500 },
  itemStyle:  { color: 'var(--text)' },
}

function Tile({ label, children }) {
  return (
    <div style={{
      background: 'var(--elevated)', border: '1px solid var(--border)',
      borderRadius: 10, padding: '14px 16px',
      boxShadow: 'var(--shadow-sm)',
    }}>
      <div style={{ fontSize: 11, fontWeight: 500, color: 'var(--text-muted)', marginBottom: 7 }}>{label}</div>
      <div style={{ fontSize: 14, fontWeight: 500, color: 'var(--text)' }}>{children}</div>
    </div>
  )
}

export default function TaskDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [task,    setTask]    = useState(null)
  const [metrics, setMetrics] = useState([])
  const [error,   setError]   = useState(null)

  const reload = async () => {
    try {
      const t = await tasksApi.get(id)
      setTask(t)
      const from = new Date(Date.now() - 3600000).toISOString()
      setMetrics(aggregateMetrics(await tasksApi.getMetrics(id, from, new Date().toISOString())))
    } catch (e) { setError(e.message) }
  }

  useEffect(() => { reload(); const t = setInterval(reload, 5000); return () => clearInterval(t) }, [id])

  if (!task && !error) return (
    <div style={{ padding: 32, fontSize: 13, color: 'var(--text-muted)' }}>Loading...</div>
  )

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 22 }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <button onClick={() => navigate('/tasks')} className="btn-secondary" style={{ padding: '8px 12px' }}>
          <ArrowLeft size={14} />
        </button>
        <div style={{ flex: 1 }}>
          <span className="label" style={{ display: 'block', marginBottom: 5 }}>Task Detail</span>
          <h1 className="page-title" style={{ fontSize: 22 }}>{task?.name || 'Unnamed Task'}</h1>
          <span className="mono" style={{ color: 'var(--text-muted)', fontSize: 11 }}>{id}</span>
        </div>
        {task && (
          <div style={{ display: 'flex', gap: 8 }}>
            {task.status === 'pending' && (
              <button onClick={async () => { await tasksApi.dispatch(id); reload() }} className="btn-primary">
                <Play size={13} /> Dispatch
              </button>
            )}
            {(task.status === 'running' || task.status === 'dispatched') && (
              <button
                onClick={async () => { await tasksApi.stop(id); reload() }}
                style={{
                  display: 'inline-flex', alignItems: 'center', gap: 6,
                  padding: '9px 18px', borderRadius: 8,
                  background: 'var(--red-dim)',
                  border: '1px solid rgba(181,60,43,0.25)',
                  color: 'var(--red)', fontSize: 14, fontWeight: 500,
                  cursor: 'pointer',
                }}
              >
                <StopCircle size={13} /> Stop
              </button>
            )}
          </div>
        )}
      </div>

      {error && <div className="error-bar">{error}</div>}

      {task && (<>
        {/* Info tiles */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
          <Tile label="Status"><Badge label={task.status} /></Tile>
          <Tile label="Type"><Badge label={task.type} /></Tile>
          <Tile label="Target Rate">
            <span className="mono">{task.target_rate_mbps} Mbps</span>
          </Tile>
          <Tile label="Downloaded">
            <span className="mono">{fmtBytes(task.total_bytes_done)}</span>
          </Tile>
          <Tile label="Created"><span style={{ fontSize: 13 }}>{fmtDate(task.created_at)}</span></Tile>
          <Tile label="Started"><span style={{ fontSize: 13 }}>{fmtDate(task.started_at)}</span></Tile>
          <Tile label="Finished"><span style={{ fontSize: 13 }}>{fmtDate(task.finished_at)}</span></Tile>
          <Tile label="Distribution"><span className="mono">{task.distribution}</span></Tile>
        </div>

        {/* Error */}
        {task.error_message && (
          <div style={{
            background: 'var(--red-dim)', border: '1px solid rgba(181,60,43,0.2)',
            borderLeft: '3px solid var(--red)', borderRadius: 10, padding: '14px 18px',
          }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--red)', marginBottom: 5, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
              Error
            </div>
            <span style={{ fontSize: 13, color: 'var(--red)' }}>{task.error_message}</span>
          </div>
        )}

        {/* URL Pools */}
        <div className="card" style={{ padding: '20px 22px' }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 15, marginBottom: 14 }}>
            URL Pools
          </h3>
          {(task.pools || []).length === 0 ? (
            <div className="empty" style={{ padding: '16px 0' }}>No pools</div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {(task.pools || []).map(pool => (
                <div key={pool.id} style={{
                  border: '1px solid var(--border)', borderRadius: 8, padding: '12px 14px',
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                    <span style={{ fontWeight: 500 }}>{pool.name}</span>
                    <Badge label={pool.type} />
                    <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{pool.urls?.length || 0} URLs</span>
                  </div>
                  <div style={{ maxHeight: 100, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 3 }}>
                    {(pool.urls || []).map((url, i) => (
                      <span key={i} className="mono" style={{ color: 'var(--accent-dark)', fontSize: 11, wordBreak: 'break-all' }}>
                        {url}
                      </span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Rate curve */}
        <div className="card" style={{ padding: '20px 22px' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 18 }}>
            <div>
              <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 15, marginBottom: 2 }}>Rate Curve</h3>
              <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Last 1 hour</span>
            </div>
          </div>
          {metrics.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <LineChart data={metrics} margin={{ top: 4, right: 4, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 6" stroke="rgba(61,57,41,0.06)" />
                <XAxis dataKey="ts"
                  tick={{ fill: 'var(--text-muted)', fontSize: 11, fontFamily: 'DM Sans' }}
                  tickLine={false} axisLine={{ stroke: 'var(--border)' }} />
                <YAxis
                  tick={{ fill: 'var(--text-muted)', fontSize: 11, fontFamily: 'DM Sans' }}
                  tickLine={false} axisLine={false} unit=" M" width={50} />
                <Tooltip
                  contentStyle={tooltipStyle.contentStyle}
                  labelStyle={tooltipStyle.labelStyle}
                  itemStyle={tooltipStyle.itemStyle}
                />
                <Legend iconType="plainline" wrapperStyle={{ fontSize: 12, paddingTop: 14, fontFamily: 'DM Sans', color: 'var(--text-dim)' }} />
                <Line type="monotone" dataKey="rate5s"  name="5s Avg"  stroke="#da7756" dot={false} strokeWidth={2} />
                <Line type="monotone" dataKey="rate30s" name="30s Avg" stroke="#3d7a52" dot={false} strokeWidth={1.5} strokeDasharray="5 3" />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="empty" style={{ height: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              No metrics yet
            </div>
          )}
        </div>

        {/* Parameters */}
        <div className="card" style={{ padding: '20px 22px' }}>
          <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 15, marginBottom: 16 }}>Parameters</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '10px 24px' }}>
            {[
              ['Pools',             task.pools?.length || 0],
              ['Child Tasks',       task.children?.length || 0],
              ['Scope',             task.execution_scope || 'single_agent'],
              ['Agent ID',          task.agent_id || '—'],
              ['Duration',          task.duration_sec ? (task.duration_sec / 86400).toFixed(2) + 'd' : '—'],
              ['Bytes Target',      task.total_bytes_target ? fmtBytes(task.total_bytes_target) : '—'],
              ['Jitter %',          task.jitter_pct],
              ['Ramp Up',           task.ramp_up_sec + 's'],
              ['Ramp Down',         task.ramp_down_sec + 's'],
              ['Dispatch Rate TPM', task.dispatch_rate_tpm || '—'],
              ['Fragments',         task.concurrent_fragments],
              ['Retries',           task.retries],
            ].map(([k, v]) => (
              <div key={k} style={{ display: 'flex', gap: 6, alignItems: 'baseline' }}>
                <span style={{ fontSize: 12, color: 'var(--text-muted)', flexShrink: 0 }}>{k}</span>
                <span className="mono" style={{ fontSize: 12, color: 'var(--text)' }}>{v}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Child tasks */}
        {task.children?.length > 0 && (
          <div className="card" style={{ overflow: 'hidden' }}>
            <div style={{ padding: '16px 22px 14px', borderBottom: '1px solid var(--border)' }}>
              <h3 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 15 }}>Child Tasks</h3>
            </div>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead className="tbl-head">
                <tr>
                  <th>Name</th><th>Type</th><th>Status</th>
                  <th style={{ textAlign: 'right' }}>Rate</th>
                  <th style={{ textAlign: 'right' }}>Downloaded</th>
                </tr>
              </thead>
              <tbody>
                {task.children.map(child => (
                  <tr key={child.id} className="tbl-row">
                    <td><span style={{ fontWeight: 500 }}>{child.name || child.id}</span></td>
                    <td><Badge label={child.type} /></td>
                    <td><Badge label={child.status} /></td>
                    <td style={{ textAlign: 'right' }}>
                      <span className="mono" style={{ fontSize: 12 }}>{child.target_rate_mbps}</span>
                    </td>
                    <td style={{ textAlign: 'right' }}>
                      <span className="mono" style={{ fontSize: 12 }}>{fmtBytes(child.total_bytes_done)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </>)}
    </div>
  )
}
