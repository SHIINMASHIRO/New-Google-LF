import React, { useState, useEffect } from 'react'
import { Activity, Server, ListTodo, Zap } from 'lucide-react'
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend
} from 'recharts'
import { dashboardApi } from '../api/index.js'
import StatCard from '../components/StatCard.jsx'
import Badge from '../components/Badge.jsx'

function fmtTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

const tooltipStyle = {
  contentStyle: {
    background: '#fff',
    border: '1px solid var(--border)',
    borderRadius: 10,
    boxShadow: 'var(--shadow-md)',
    fontFamily: 'var(--font-ui)',
    fontSize: 12,
    padding: '10px 14px',
  },
  labelStyle: { color: 'var(--text-muted)', marginBottom: 4, fontWeight: 500 },
  itemStyle:  { color: 'var(--text)' },
}

export default function Dashboard() {
  const [overview, setOverview] = useState(null)
  const [history,  setHistory]  = useState([])
  const [error,    setError]    = useState(null)

  const loadOverview = async () => {
    try { const ov = await dashboardApi.overview(); setOverview(ov); setError(null) }
    catch (e) { setError(e.message) }
  }

  const loadHistory = async () => {
    try {
      const hist = await dashboardApi.bandwidthHistory(
        new Date(Date.now() - 7 * 24 * 3600 * 1000).toISOString(),
        new Date().toISOString(), '5m'
      )
      setHistory((hist || []).map(p => ({
        ts:  fmtTime(p.ts),
        avg: +p.avg_mbps.toFixed(2),
        max: +p.max_mbps.toFixed(2),
      })))
    } catch (e) { setError(e.message) }
  }

  useEffect(() => {
    loadOverview()
    loadHistory()
    const t1 = setInterval(loadOverview, 3000)
    const t2 = setInterval(loadHistory, 60000)
    return () => { clearInterval(t1); clearInterval(t2) }
  }, [])

  const sortedAgents = overview?.agents
    ? [...overview.agents].sort((a, b) => b.rate_mbps - a.rate_mbps)
    : []
  const maxRate = sortedAgents[0]?.rate_mbps || 1

  return (
    <div style={{ padding: '28px 32px', display: 'flex', flexDirection: 'column', gap: 24 }}>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
            <span className="dot-live" />
            <span style={{ fontSize: 12, color: 'var(--green)', fontWeight: 500 }}>Live</span>
          </div>
          <h1 className="page-title">Dashboard</h1>
        </div>
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>
          {new Date().toLocaleString()}
        </span>
      </div>

      {error && <div className="error-bar">{error}</div>}

      {/* Stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 14 }}>
        <StatCard title="Total Bandwidth"
          value={overview ? overview.total_rate_mbps.toFixed(1) + ' Mbps' : null}
          icon={Activity} color="orange" />
        <StatCard title="Online Agents"
          value={overview?.online_agents}
          icon={Server} color="green"
          sub={`${overview?.total_agents ?? 0} registered`} />
        <StatCard title="Running Tasks"
          value={overview?.running_tasks}
          icon={Zap} color="yellow"
          sub={`${overview?.total_tasks ?? 0} total`} />
        <StatCard title="Total Tasks"
          value={overview?.total_tasks}
          icon={ListTodo} color="purple" />
      </div>

      {/* Bandwidth history */}
      <div className="card" style={{ padding: '22px 24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
          <div>
            <h2 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)', marginBottom: 2 }}>
              Bandwidth History
            </h2>
            <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>Last 7 days · 5 min resolution</span>
          </div>
        </div>

        {history.length > 0 ? (
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={history} margin={{ top: 4, right: 4, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="gAvg" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%"   stopColor="#da7756" stopOpacity={0.18} />
                  <stop offset="100%" stopColor="#da7756" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="gMax" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%"   stopColor="#3d7a52" stopOpacity={0.14} />
                  <stop offset="100%" stopColor="#3d7a52" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 6" stroke="rgba(61,57,41,0.06)" />
              <XAxis dataKey="ts"
                tick={{ fill: 'var(--text-muted)', fontSize: 11, fontFamily: 'DM Sans' }}
                tickLine={false}
                axisLine={{ stroke: 'var(--border)' }} />
              <YAxis
                tick={{ fill: 'var(--text-muted)', fontSize: 11, fontFamily: 'DM Sans' }}
                tickLine={false} axisLine={false}
                unit=" M" width={52} />
              <Tooltip
                contentStyle={tooltipStyle.contentStyle}
                labelStyle={tooltipStyle.labelStyle}
                itemStyle={tooltipStyle.itemStyle}
              />
              <Legend
                iconType="plainline"
                wrapperStyle={{ fontSize: 12, paddingTop: 16, fontFamily: 'DM Sans', color: 'var(--text-dim)' }}
              />
              <Area type="monotone" dataKey="avg" name="Avg Mbps"
                stroke="#da7756" fill="url(#gAvg)" strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="max" name="Max Mbps"
                stroke="#3d7a52" fill="url(#gMax)" strokeWidth={1.5} dot={false} strokeDasharray="5 3" />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="empty" style={{ height: 160, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            No data yet — start some tasks to see bandwidth history
          </div>
        )}
      </div>

      {/* Agent ranking */}
      <div className="card" style={{ overflow: 'hidden' }}>
        <div style={{ padding: '20px 24px 16px', borderBottom: '1px solid var(--border)' }}>
          <h2 style={{ fontFamily: 'var(--font-serif)', fontWeight: 600, fontSize: 16, color: 'var(--text)' }}>
            Agent Bandwidth Ranking
          </h2>
        </div>

        {sortedAgents.length > 0 ? (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead className="tbl-head">
              <tr>
                <th>Agent</th>
                <th>IP Address</th>
                <th>Status</th>
                <th style={{ textAlign: 'right' }}>Rate (Mbps)</th>
                <th style={{ textAlign: 'right', width: 140 }}>Share</th>
              </tr>
            </thead>
            <tbody>
              {sortedAgents.map(a => {
                const pct = (a.rate_mbps / maxRate) * 100
                return (
                  <tr key={a.id} className="tbl-row">
                    <td>
                      <span style={{ fontWeight: 500 }}>
                        {a.hostname || a.id.slice(0, 8)}
                      </span>
                    </td>
                    <td>
                      <span className="mono" style={{ color: 'var(--text-dim)' }}>{a.ip}</span>
                    </td>
                    <td><Badge label={a.status} /></td>
                    <td style={{ textAlign: 'right' }}>
                      <span className="mono" style={{ fontWeight: 500 }}>
                        {a.rate_mbps.toFixed(2)}
                      </span>
                    </td>
                    <td>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, justifyContent: 'flex-end' }}>
                        <div style={{ width: 80, height: 4, background: 'var(--bg)', borderRadius: 4 }}>
                          <div style={{
                            width: `${pct}%`, height: '100%',
                            background: 'var(--accent)',
                            borderRadius: 4, transition: 'width 0.5s',
                          }} />
                        </div>
                        <span className="mono" style={{ color: 'var(--text-muted)', width: 34, textAlign: 'right', fontSize: 11 }}>
                          {pct.toFixed(0)}%
                        </span>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        ) : (
          <div className="empty">No agents registered yet</div>
        )}
      </div>
    </div>
  )
}
