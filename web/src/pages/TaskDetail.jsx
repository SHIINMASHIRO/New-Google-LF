import React, { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, StopCircle, Play } from 'lucide-react'
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend
} from 'recharts'
import { tasksApi } from '../api/index.js'
import Badge from '../components/Badge.jsx'

function fmtDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function fmtBytes(b) {
  if (!b) return '0 B'
  if (b > 1e9) return (b / 1e9).toFixed(2) + ' GB'
  if (b > 1e6) return (b / 1e6).toFixed(2) + ' MB'
  if (b > 1e3) return (b / 1e3).toFixed(2) + ' KB'
  return b + ' B'
}

export default function TaskDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [task, setTask] = useState(null)
  const [metrics, setMetrics] = useState([])
  const [error, setError] = useState(null)

  const reload = async () => {
    try {
      const t = await tasksApi.get(id)
      setTask(t)
      // Fetch last 1h of metrics
      const from = new Date(Date.now() - 3600000).toISOString()
      const to = new Date().toISOString()
      const m = await tasksApi.getMetrics(id, from, to)
      setMetrics((m || []).map(x => ({
        ts: new Date(x.recorded_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
        rate5s: +x.rate_mbps_5s.toFixed(2),
        rate30s: +x.rate_mbps_30s.toFixed(2),
      })))
    } catch (e) {
      setError(e.message)
    }
  }

  useEffect(() => {
    reload()
    const t = setInterval(reload, 5000)
    return () => clearInterval(t)
  }, [id])

  if (!task && !error) {
    return <div className="p-6 text-gray-500 text-sm">Loading...</div>
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate('/tasks')} className="p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors">
          <ArrowLeft size={15} />
        </button>
        <div className="flex-1">
          <h1 className="text-xl font-semibold text-white">{task?.name || 'Task Detail'}</h1>
          <p className="text-sm text-gray-500 font-mono">{id}</p>
        </div>
        {task && (
          <div className="flex gap-2">
            {task.status === 'pending' && (
              <button onClick={async () => { await tasksApi.dispatch(id); reload() }}
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-green-600/20 text-green-400 hover:bg-green-600/30 text-sm transition-colors">
                <Play size={13} /> Dispatch
              </button>
            )}
            {(task.status === 'running' || task.status === 'dispatched') && (
              <button onClick={async () => { await tasksApi.stop(id); reload() }}
                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-600/20 text-red-400 hover:bg-red-600/30 text-sm transition-colors">
                <StopCircle size={13} /> Stop
              </button>
            )}
          </div>
        )}
      </div>

      {error && <div className="bg-red-500/10 border border-red-500/30 rounded-lg p-3 text-red-400 text-sm">{error}</div>}

      {task && (
        <>
          {/* Info grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[
              { label: 'Status', value: <Badge label={task.status} /> },
              { label: 'Type', value: <Badge label={task.type} /> },
              { label: 'Target Rate', value: task.target_rate_mbps + ' Mbps' },
              { label: 'Downloaded', value: fmtBytes(task.total_bytes_done) },
              { label: 'Created', value: fmtDate(task.created_at) },
              { label: 'Started', value: fmtDate(task.started_at) },
              { label: 'Finished', value: fmtDate(task.finished_at) },
              { label: 'Distribution', value: task.distribution },
            ].map(({ label, value }) => (
              <div key={label} className="bg-gray-900 border border-gray-800 rounded-xl p-4">
                <p className="text-xs text-gray-500 mb-1">{label}</p>
                <div className="text-white text-sm font-medium">{value}</div>
              </div>
            ))}
          </div>

          {/* Target URL */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-4">
            <p className="text-xs text-gray-500 mb-1">Target URL</p>
            <p className="text-sm text-blue-400 break-all">{task.target_url}</p>
          </div>

          {/* Error */}
          {task.error_message && (
            <div className="bg-red-500/10 border border-red-500/30 rounded-xl p-4">
              <p className="text-xs text-gray-500 mb-1">Error</p>
              <p className="text-sm text-red-400">{task.error_message}</p>
            </div>
          )}

          {/* Rate curve */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
            <h2 className="text-sm font-medium text-white mb-4">Rate Curve</h2>
            {metrics.length > 0 ? (
              <ResponsiveContainer width="100%" height={200}>
                <LineChart data={metrics} margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                  <XAxis dataKey="ts" tick={{ fill: '#6b7280', fontSize: 10 }} tickLine={false} />
                  <YAxis tick={{ fill: '#6b7280', fontSize: 10 }} tickLine={false} axisLine={false} unit=" Mbps" width={60} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#111827', border: '1px solid #374151', borderRadius: 8 }}
                    labelStyle={{ color: '#9ca3af' }}
                  />
                  <Legend iconType="circle" wrapperStyle={{ fontSize: 12 }} />
                  <Line type="monotone" dataKey="rate5s" name="5s Avg" stroke="#3b82f6" dot={false} strokeWidth={2} />
                  <Line type="monotone" dataKey="rate30s" name="30s Avg" stroke="#10b981" dot={false} strokeWidth={1.5} strokeDasharray="4 2" />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-32 text-gray-600 text-sm">No metrics yet</div>
            )}
          </div>

          {/* Task parameters */}
          <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
            <h2 className="text-sm font-medium text-white mb-4">Parameters</h2>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-x-8 gap-y-3 text-sm">
              {[
                ['Agent ID', task.agent_id || '—'],
                ['Duration', task.duration_sec ? task.duration_sec + 's' : '—'],
                ['Total Bytes Target', task.total_bytes_target ? fmtBytes(task.total_bytes_target) : '—'],
                ['Jitter %', task.jitter_pct],
                ['Ramp Up', task.ramp_up_sec + 's'],
                ['Ramp Down', task.ramp_down_sec + 's'],
                ['Dispatch Rate TPM', task.dispatch_rate_tpm || '—'],
                ['Fragments', task.concurrent_fragments],
                ['Retries', task.retries],
              ].map(([k, v]) => (
                <div key={k}>
                  <span className="text-gray-500">{k}: </span>
                  <span className="text-white">{v}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
