import React from 'react'

const variants = {
  online:       { color: 'var(--green)',  bg: 'var(--green-dim)',  pulse: true },
  offline:      { color: 'var(--text-muted)', bg: 'rgba(170,168,159,0.12)', pulse: false },
  running:      { color: 'var(--accent-dark)', bg: 'var(--accent-dim)', pulse: true },
  pending:      { color: 'var(--amber)',  bg: 'var(--amber-dim)',  pulse: false },
  dispatched:   { color: 'var(--purple)', bg: 'var(--purple-dim)', pulse: true },
  done:         { color: 'var(--green)',  bg: 'var(--green-dim)',  pulse: false },
  failed:       { color: 'var(--red)',    bg: 'var(--red-dim)',    pulse: false },
  stopped:      { color: 'var(--text-muted)', bg: 'rgba(170,168,159,0.12)', pulse: false },
  success:      { color: 'var(--green)',  bg: 'var(--green-dim)',  pulse: false },
  provisioning: { color: 'var(--amber)',  bg: 'var(--amber-dim)',  pulse: true },
  'ssh key':    { color: 'var(--blue)',   bg: 'var(--blue-dim)',   pulse: false },
  password:     { color: 'var(--purple)', bg: 'var(--purple-dim)', pulse: false },
  youtube:      { color: 'var(--red)',    bg: 'var(--red-dim)',    pulse: false },
  static:       { color: 'var(--blue)',   bg: 'var(--blue-dim)',   pulse: false },
}

export default function Badge({ label }) {
  const v = variants[label?.toLowerCase()] ?? { color: 'var(--text-muted)', bg: 'rgba(170,168,159,0.12)', pulse: false }

  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 5,
      padding: '3px 9px',
      borderRadius: 20,
      background: v.bg,
      fontFamily: 'var(--font-ui)',
      fontSize: 11,
      fontWeight: 500,
      color: v.color,
      whiteSpace: 'nowrap',
    }}>
      <span style={{
        display: 'inline-block',
        width: 5,
        height: 5,
        borderRadius: '50%',
        background: v.color,
        flexShrink: 0,
        animation: v.pulse ? 'pulse-live 2.5s ease-in-out infinite' : 'none',
      }} />
      {label}
    </span>
  )
}
