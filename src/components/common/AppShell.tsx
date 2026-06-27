import { Outlet, Link } from 'react-router-dom';

export default function AppShell() {
  return (
    <div style={{ display: 'flex', height: '100vh', backgroundColor: '#080C14', color: '#E8EDF5', fontFamily: '"DM Sans", sans-serif' }}>
      {/* SideNav */}
      <aside style={{ width: '240px', borderRight: '1px solid rgba(255,255,255,0.1)', display: 'flex', flexDirection: 'column' }}>
        <div style={{ padding: '24px', fontWeight: 'bold', fontSize: '18px', color: '#4F8EF7' }}>
          DistributedTrace
        </div>
        <nav style={{ flex: 1, padding: '16px 0', display: 'flex', flexDirection: 'column', gap: '8px' }}>
          <Link to="/dashboard" style={{ padding: '8px 24px', color: '#E8EDF5', textDecoration: 'none', background: 'rgba(255,255,255,0.05)', borderLeft: '2px solid #4F8EF7' }}>Dashboard</Link>
          <Link to="/dashboard/traces" style={{ padding: '8px 24px', color: '#9BA8BB', textDecoration: 'none' }}>Traces</Link>
          <Link to="/dashboard/map" style={{ padding: '8px 24px', color: '#9BA8BB', textDecoration: 'none' }}>Service Map</Link>
        </nav>
      </aside>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {/* TopBar */}
        <header style={{ height: '64px', borderBottom: '1px solid rgba(255,255,255,0.1)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 24px' }}>
          <div style={{ width: '300px', background: '#0F1420', padding: '6px 12px', borderRadius: '4px', border: '1px solid rgba(255,255,255,0.1)' }}>
            <span style={{ color: '#5C6B82', fontSize: '14px' }}>Search traces...</span>
          </div>
          <div style={{ display: 'flex', gap: '16px', alignItems: 'center' }}>
            <div style={{ fontSize: '12px', background: '#34D399', color: '#080C14', padding: '2px 8px', borderRadius: '99px', fontWeight: 'bold' }}>us-east-1</div>
            <div style={{ fontSize: '12px', background: 'rgba(255,255,255,0.05)', padding: '4px 12px', borderRadius: '4px' }}>Last 15m</div>
          </div>
        </header>

        {/* MainContent */}
        <main style={{ flex: 1, padding: '24px', overflowY: 'auto' }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}
