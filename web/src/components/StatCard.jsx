import React from 'react'

const colorMap = {
  blue:   { accent: 'var(--blue)',   dim: 'var(--blue-dim)' },
  green:  { accent: 'var(--green)',  dim: 'var(--green-dim)' },
  yellow: { accent: 'var(--amber)',  dim: 'var(--amber-dim)' },
  purple: { accent: 'var(--purple)', dim: 'var(--purple-dim)' },
  red:    { accent: 'var(--red)',    dim: 'var(--red-dim)' },
  orange: { accent: 'var(--accent)', dim: 'var(--accent-dim)' },
}

export default function StatCard({ title, value, sub, icon: Icon, color = 'orange' }) {
  const { accent, dim } = colorMap[color] || colorMap.orange

  return (
    <div style={{
      background: 'var(--elevated)',
      border: '1px solid var(--border)',
      borderRadius: 12,
      padding: '18px 20px',
      boxShadow: 'var(--shadow-sm)',
    }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 12 }}>
        <span style={{
          fontFamily: 'var(--font-ui)',
          fontSize: 12,
          fontWeight: 500,
          color: 'var(--text-muted)',
        }}>{title}</span>
        {Icon && (
          <div style={{
            width: 30,
            height: 30,
            borderRadius: 8,
            background: dim,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}>
            <Icon size={14} style={{ color: accent }} />
          </div>
        )}
      </div>
      <div style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 24,
        fontWeight: 500,
        color: 'var(--text)',
        lineHeight: 1,
        letterSpacing: '-0.02em',
      }}>{value ?? '—'}</div>
      {sub && (
        <div style={{
          fontFamily: 'var(--font-ui)',
          fontSize: 12,
          color: 'var(--text-muted)',
          marginTop: 7,
        }}>{sub}</div>
      )}
    </div>
  )
}
