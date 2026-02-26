import React from 'react'

export default function StatCard({ title, value, sub, icon: Icon, color = 'blue' }) {
  const colors = {
    blue: 'text-blue-400 bg-blue-500/10',
    green: 'text-green-400 bg-green-500/10',
    yellow: 'text-yellow-400 bg-yellow-500/10',
    purple: 'text-purple-400 bg-purple-500/10',
    red: 'text-red-400 bg-red-500/10',
  }
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 flex items-start gap-4">
      {Icon && (
        <div className={`p-2.5 rounded-lg ${colors[color]}`}>
          <Icon size={20} className={colors[color].split(' ')[0]} />
        </div>
      )}
      <div className="flex-1 min-w-0">
        <p className="text-xs text-gray-500 mb-1">{title}</p>
        <p className="text-2xl font-bold text-white leading-none">{value ?? '—'}</p>
        {sub && <p className="text-xs text-gray-500 mt-1">{sub}</p>}
      </div>
    </div>
  )
}
