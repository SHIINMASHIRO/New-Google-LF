import React from 'react'

const variants = {
  online:     'bg-green-500/20 text-green-400 border-green-500/30',
  offline:    'bg-gray-500/20 text-gray-400 border-gray-500/30',
  running:    'bg-blue-500/20 text-blue-400 border-blue-500/30',
  pending:    'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  dispatched: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  done:       'bg-green-500/20 text-green-400 border-green-500/30',
  failed:     'bg-red-500/20 text-red-400 border-red-500/30',
  stopped:    'bg-gray-500/20 text-gray-400 border-gray-500/30',
  success:    'bg-green-500/20 text-green-400 border-green-500/30',
  provisioning: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  'ssh key':  'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
  password:   'bg-orange-500/20 text-orange-400 border-orange-500/30',
  youtube:    'bg-red-500/20 text-red-400 border-red-500/30',
  static:     'bg-blue-500/20 text-blue-400 border-blue-500/30',
}

export default function Badge({ label }) {
  const cls = variants[label?.toLowerCase()] ?? 'bg-gray-500/20 text-gray-400 border-gray-500/30'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-md text-xs font-medium border ${cls}`}>
      {label}
    </span>
  )
}
