import type { Alert } from '../types';

export function AlertsPanel({ alerts }: { alerts: Alert[] }) {
  return (
    <div style={{ background: 'var(--bg-panel)', border: '1px solid var(--border)', borderRadius: 6, padding: 16, maxHeight: 320, overflowY: 'auto' }}>
      {alerts.length === 0 && (
        <p style={{ color: 'var(--text-muted)', fontSize: 13 }}>No alerts — all elements nominal.</p>
      )}
      {alerts.map((a, i) => (
        <div
          key={i}
          style={{
            borderLeft: '3px solid var(--signal-red)',
            background: 'rgba(240,85,74,0.08)',
            padding: '8px 12px',
            marginBottom: 8,
            borderRadius: 4,
            fontSize: 13,
          }}
        >
          <div>
            <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 600 }}>{a.element_id}</span>
            {' — '}{a.reason}
          </div>
          <div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 2 }}>
            {new Date(a.timestamp).toLocaleTimeString()}
          </div>
        </div>
      ))}
    </div>
  );
}