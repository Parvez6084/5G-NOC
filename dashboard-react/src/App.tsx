import { useState, useRef } from 'react';
import { useNocSocket } from './hooks/useWebSocket';
import { ElementCard } from './components/ElementCard';
import { LatencyChart, type ChartPoint } from './components/LatencyChart';
import { AlertsPanel } from './components/AlertsPanel';
import type { Telemetry, Alert, WSMessage } from './types';

export default function App() {
  const [elements, setElements] = useState<Record<string, Telemetry>>({});
  const [chartData, setChartData] = useState<ChartPoint[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const tCounter = useRef(0);

  const { connected } = useNocSocket((msg: WSMessage) => {
    if (msg.type === 'telemetry') {
      tCounter.current += 1;
      const point: ChartPoint = { t: tCounter.current };
      const updated: Record<string, Telemetry> = {};

      msg.data.forEach((t) => {
        updated[t.element_id] = t;
        point[t.element_id] = t.latency_ms;
      });

      setElements((prev) => ({ ...prev, ...updated }));
      setChartData((prev) => [...prev.slice(-29), point]);
    }

    if (msg.type === 'alert') {
       setAlerts((prev) => {
        const isDupe = prev[0]?.element_id === msg.data.element_id &&
                      prev[0]?.reason === msg.data.reason &&
                      prev[0]?.timestamp === msg.data.timestamp;
        if (isDupe) return prev;
        return [msg.data, ...prev].slice(0, 30);
      });
    }
  });

  const elementList = Object.values(elements);

  return (
    <div style={{ minHeight: '100vh', padding: 24, fontFamily: 'var(--font-sans)' }}>
      <header style={{ display: 'flex', alignItems: 'baseline', gap: 12, marginBottom: 24 }}>
        <h1 style={{ margin: 0, fontFamily: 'var(--font-mono)', fontSize: 22, letterSpacing: -0.5 }}>
          5G–NOC
        </h1>
        <span style={{ color: 'var(--text-muted)', fontSize: 13 }}>Network Operations Center</span>
        <span
          style={{
            marginLeft: 'auto',
            fontSize: 11,
            padding: '4px 10px',
            borderRadius: 12,
            fontFamily: 'var(--font-mono)',
            background: connected ? 'rgba(63,214,140,0.12)' : 'rgba(240,85,74,0.12)',
            color: connected ? 'var(--signal-green)' : 'var(--signal-red)',
          }}
        >
          {connected ? '● LIVE' : '○ RECONNECTING'}
        </span>
      </header>

      <section style={{ marginBottom: 24 }}>
        <SectionLabel>Network Elements</SectionLabel>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: 12 }}>
          {elementList.map((t) => <ElementCard key={t.element_id} t={t} />)}
        </div>
      </section>

      <section style={{ marginBottom: 24 }}>
        <SectionLabel>Latency Trend</SectionLabel>
        <LatencyChart data={chartData} seriesIds={elementList.map((e) => e.element_id)} />
      </section>

      <section>
        <SectionLabel>Alerts</SectionLabel>
        <AlertsPanel alerts={alerts} />
      </section>
    </div>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ fontSize: 11, letterSpacing: 1.5, color: 'var(--text-muted)', marginBottom: 10, textTransform: 'uppercase' }}>
      {children}
    </div>
  );
}