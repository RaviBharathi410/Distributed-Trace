export default function Dashboard() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <h1 style={{ fontSize: '24px', fontWeight: 'bold', margin: 0 }}>Metrics Overview</h1>
      
      {/* Metrics Row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
        {[1, 2, 3, 4].map(i => (
          <div key={i} style={{ background: '#0F1420', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', padding: '16px' }}>
            <div style={{ color: '#9BA8BB', fontSize: '12px', marginBottom: '8px' }}>P99 Latency</div>
            <div style={{ fontSize: '30px', color: '#E8EDF5' }}>24ms</div>
          </div>
        ))}
      </div>

      {/* Panels */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
        <div style={{ background: '#0F1420', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', minHeight: '300px', padding: '16px' }}>
          <h2 style={{ fontSize: '16px', color: '#9BA8BB', marginTop: 0 }}>Recent Anomalies</h2>
          <div style={{ marginTop: '16px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
            <div style={{ padding: '12px', border: '1px solid rgba(255,255,255,0.06)', borderRadius: '4px' }}>
              <div style={{ display: 'inline-block', padding: '2px 6px', background: '#EF4444', color: '#080C14', fontSize: '10px', fontWeight: 'bold', borderRadius: '2px', marginBottom: '4px' }}>CRITICAL</div>
              <div style={{ fontSize: '14px' }}>Auth Service Latency Spike</div>
            </div>
            <div style={{ padding: '12px', border: '1px solid rgba(255,255,255,0.06)', borderRadius: '4px' }}>
              <div style={{ display: 'inline-block', padding: '2px 6px', background: '#F97316', color: '#080C14', fontSize: '10px', fontWeight: 'bold', borderRadius: '2px', marginBottom: '4px' }}>HIGH</div>
              <div style={{ fontSize: '14px' }}>DB Connection Pool Exhausted</div>
            </div>
          </div>
        </div>
        <div style={{ background: '#0F1420', border: '1px solid rgba(255,255,255,0.1)', borderRadius: '6px', minHeight: '300px', padding: '16px' }}>
          <h2 style={{ fontSize: '16px', color: '#9BA8BB', marginTop: 0 }}>Trace Explorer Preview</h2>
          <div style={{ marginTop: '24px', textAlign: 'center', color: '#5C6B82', fontSize: '14px' }}>
            No traces actively selected. Search for a trace ID to view flame graph.
          </div>
        </div>
      </div>
    </div>
  )
}
