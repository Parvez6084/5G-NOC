import type { Telemetry } from '../types';

const typeLabels: Record<string, string> = {
  'base-station': 'BASE STATION',
  'core-node': 'CORE NODE',
  'edge-device': 'EDGE DEVICE',
};

function metricStatus(latency: number, loss: number): 'ok' | 'warn' | 'critical' {
  if (latency > 130 || loss > 8) return 'critical';
  if (latency > 60 || loss > 2) return 'warn';
  return 'ok';
}

const statusColor = {
  ok: 'var(--signal-green)',
  warn: 'var(--signal-amber)',
  critical: 'var(--signal-red)',
};

export function ElementCard({ t }: { t: Telemetry }) {
  const status = metricStatus(t.latency_ms, t.packet_loss_pct);

  return (
    <div
      style={{
        background: 'var(--bg-panel)',
        border: '1px solid var(--border)',
        borderRadius: 6,
        padding: 16,
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* signature signal pulse bar */}
      <div
        style={{
          position: 'absolute',
          top: 0, left: 0, right: 0,
          height: 2,
          background: statusColor[status],
          opacity: 0.9,
          animation: status !== 'ok' ? 'pulse 1.4s ease-in-out infinite' : undefined,
        }}
      />
      <style>{`
        @keyframes pulse { 0%,100% { opacity: 0.4; } 50% { opacity: 1; } }
      `}</style>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 15, fontWeight: 600 }}>
          {t.element_id}
        </span>
        <span style={{ fontSize: 10, letterSpacing: 1, color: 'var(--text-muted)' }}>
          {typeLabels[t.element_type] ?? t.element_type.toUpperCase()}
        </span>
      </div>

      <div style={{ marginTop: 12, display: 'grid', gap: 6 }}>
        <Metric label="Latency" value={`${t.latency_ms.toFixed(1)} ms`} warn={status !== 'ok'} />
        <Metric label="Throughput" value={`${t.throughput_mbps.toFixed(0)} Mbps`} />
        <Metric label="Packet loss" value={`${t.packet_loss_pct.toFixed(2)}%`} warn={status !== 'ok'} />
        <Metric label="CPU" value={`${t.cpu_usage_pct.toFixed(0)}%`} />
        <Metric label="Memory" value={`${t.memory_usage_pct.toFixed(0)}%`} />
      </div>
    </div>
  );
}

function Metric({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
      <span style={{ color: 'var(--text-muted)' }}>{label}</span>
      <span
        style={{
          fontFamily: 'var(--font-mono)',
          color: warn ? 'var(--signal-amber)' : 'var(--text-primary)',
        }}
      >
        {value}
      </span>
    </div>
  );
}