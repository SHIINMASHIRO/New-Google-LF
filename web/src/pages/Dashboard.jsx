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
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export default function Dashboard() {
  const [overview, setOverview] = useState(null)
  const [history, setHistory] = useState([])
  const [error, setError] = useState(null)

  const loadData = async () => {
    try {
      const [ov, hist] = await Promise.all([
        dashboardApi.overview(),
        dashboardApi.bandwidthHistory(
          new Date(Date.now() - 7 * 24 * 3600 * 1000).toISOString(),
          new Date().toISOString(),
          '5m'
        ),
      ])
      setOverview(ov)
      setHistory((hist || []).map(p => ({
        ts: fmtTime(p.ts),
        avg: +p.avg_mbps.toFixed(2),
        max: +p.max_mbps.toFixed(2),
      })))
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => {
    loadData()
    const t = setInterval(loadData, 10000)
    return () => clearInterval(t)
  }, [])

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-white">Dashboard</h1>
        <p className="text-sm text-gray-500 mt-0.5">Real-time overview of all agents and tasks</p>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{error}</div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard title="Total Agents" value={overview?.total_agents} icon={Server} color="blue"
          sub={`${overview?.online_agents ?? 0} online`} />
        <StatCard title="Total Tasks" value={overview?.total_tasks} icon={ListTodo} color="purple"
          sub={`${overview?.running_tasks ?? 0} running`} />
        <StatCard title="Total Bandwidth" value={overview ? overview.total_rate_mbps.toFixed(1) + ' Mbps' : null}
          icon={Activity} color="green" />
        <StatCard title="Active" value={overview?.online_agents} icon={Zap} color="yellow"
          sub="Online agents" />
      </div>

      {/* Bandwidth history chart */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
        <h2 className="text-sm font-medium text-white mb-4">Bandwidth — Last 7 Days</h2>
        {history.length > 0 ? (
          <ResponsiveContainer width="100%" height={240}>
            <AreaChart data={history} margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="avgGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="maxGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
              <XAxis dataKey="ts" tick={{ fill: '#6b7280', fontSize: 11 }} tickLine={false} />
              <YAxis tick={{ fill: '#6b7280', fontSize: 11 }} tickLine={false} axisLine={false}
                unit=" Mbps" width={65} />
              <Tooltip
                contentStyle={{ backgroundColor: '#111827', border: '1px solid #374151', borderRadius: 8 }}
                labelStyle={{ color: '#9ca3af' }}
                itemStyle={{ color: '#e5e7eb' }}
              />
              <Legend iconType="circle" wrapperStyle={{ fontSize: 12, paddingTop: 8 }} />
              <Area type="monotone" dataKey="avg" name="Avg Mbps" stroke="#3b82f6"
                fill="url(#avgGrad)" strokeWidth={2} dot={false} />
              <Area type="monotone" dataKey="max" name="Max Mbps" stroke="#10b981"
                fill="url(#maxGrad)" strokeWidth={1.5} dot={false} strokeDasharray="4 2" />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex items-center justify-center h-40 text-gray-600 text-sm">
            No data yet — start some tasks to see bandwidth history
          </div>
        )}
      </div>

      {/* Agent bandwidth ranking */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
        <h2 className="text-sm font-medium text-white mb-4">Agent Bandwidth Ranking</h2>
        {overview?.agents?.length > 0 ? (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-gray-500 text-xs border-b border-gray-800">
                <th className="pb-2 text-left font-medium">Agent</th>
                <th className="pb-2 text-left font-medium">IP</th>
                <th className="pb-2 text-left font-medium">Status</th>
                <th className="pb-2 text-right font-medium">Rate (Mbps)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-800/50">
              {[...overview.agents]
                .sort((a, b) => b.rate_mbps - a.rate_mbps)
                .map(a => (
                  <tr key={a.id}>
                    <td className="py-2.5 font-mono text-xs text-gray-400">{a.hostname || a.id.slice(0, 8)}</td>
                    <td className="py-2.5 text-gray-400">{a.ip}</td>
                    <td className="py-2.5"><Badge label={a.status} /></td>
                    <td className="py-2.5 text-right">
                      <span className="text-white font-mono">{a.rate_mbps.toFixed(2)}</span>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        ) : (
          <p className="text-gray-600 text-sm">No agents registered yet</p>
        )}
      </div>
    </div>
  )
}
