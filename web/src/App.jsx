import React from 'react'
import { Routes, Route, NavLink } from 'react-router-dom'
import { LayoutDashboard, Server, ListTodo, Key, Link2 } from 'lucide-react'
import Dashboard from './pages/Dashboard.jsx'
import Agents from './pages/Agents.jsx'
import Tasks from './pages/Tasks.jsx'
import TaskDetail from './pages/TaskDetail.jsx'
import Credentials from './pages/Credentials.jsx'
import URLPools from './pages/URLPools.jsx'

const navItems = [
  { to: '/',            icon: LayoutDashboard, label: 'Dashboard',   exact: true },
  { to: '/agents',      icon: Server,          label: 'Agents' },
  { to: '/url-pools',   icon: Link2,           label: 'URL Pools' },
  { to: '/tasks',       icon: ListTodo,        label: 'Tasks' },
  { to: '/credentials', icon: Key,             label: 'Credentials' },
]

export default function App() {
  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden', background: 'var(--bg)' }}>
      {/* ── Sidebar ── */}
      <aside style={{
        width: 220,
        flexShrink: 0,
        background: 'var(--surface)',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
      }}>
        {/* Logo */}
        <div style={{
          padding: '20px 18px 16px',
          borderBottom: '1px solid var(--border)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <div style={{
              width: 32,
              height: 32,
              background: 'var(--accent-dim)',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}>
              {/* Simple traffic icon */}
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M2 8h12M8 2v12M4.5 4.5l7 7M11.5 4.5l-7 7" stroke="var(--accent)" strokeWidth="1.5" strokeLinecap="round"/>
              </svg>
            </div>
            <div>
              <div style={{
                fontFamily: 'var(--font-serif)',
                fontWeight: 600,
                fontSize: 15,
                color: 'var(--text)',
                letterSpacing: '-0.01em',
                lineHeight: 1.1,
              }}>ngoogle</div>
              <div style={{
                fontFamily: 'var(--font-ui)',
                fontSize: 11,
                color: 'var(--text-muted)',
                marginTop: 1,
                fontWeight: 400,
              }}>Traffic System</div>
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, padding: '10px 10px', display: 'flex', flexDirection: 'column', gap: 2 }}>
          {navItems.map(({ to, icon: Icon, label, exact }) => (
            <NavLink
              key={to}
              to={to}
              end={exact}
              style={({ isActive }) => ({
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '9px 12px',
                borderRadius: 8,
                textDecoration: 'none',
                fontFamily: 'var(--font-ui)',
                fontSize: 14,
                fontWeight: isActive ? 500 : 400,
                color: isActive ? 'var(--accent-dark)' : 'var(--text-dim)',
                background: isActive ? 'var(--accent-dim)' : 'transparent',
                transition: 'all 0.12s',
              })}
            >
              {({ isActive }) => (
                <>
                  <Icon size={15} style={{ opacity: isActive ? 1 : 0.7 }} />
                  {label}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div style={{
          padding: '12px 18px',
          borderTop: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          gap: 7,
        }}>
          <span className="dot-live" />
          <span style={{
            fontFamily: 'var(--font-ui)',
            fontSize: 12,
            color: 'var(--text-muted)',
          }}>v1.0.0 · Online</span>
        </div>
      </aside>

      {/* ── Main ── */}
      <main style={{ flex: 1, overflowY: 'auto' }}>
        <Routes>
          <Route path="/"            element={<Dashboard />} />
          <Route path="/agents"      element={<Agents />} />
          <Route path="/url-pools"   element={<URLPools />} />
          <Route path="/tasks"       element={<Tasks />} />
          <Route path="/tasks/:id"   element={<TaskDetail />} />
          <Route path="/credentials" element={<Credentials />} />
        </Routes>
      </main>
    </div>
  )
}
