import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

export interface ChartPoint {
  t: number;
  [elementId: string]: number;
}

const LINE_COLORS = ['#4dd4ff', '#3fd68c', '#e8a33d', '#c78bff', '#f0554a'];

export function LatencyChart({ data, seriesIds }: { data: ChartPoint[]; seriesIds: string[] }) {
  return (
    <div style={{ background: 'var(--bg-panel)', border: '1px solid var(--border)', borderRadius: 6, padding: 16 }}>
      <ResponsiveContainer width="100%" height={220}>
        <LineChart data={data}>
          <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" />
          <XAxis dataKey="t" stroke="#6e7a8a" tick={{ fontSize: 11 }} />
          <YAxis stroke="#6e7a8a" tick={{ fontSize: 11 }} label={{ value: 'ms', position: 'insideLeft', fill: '#6e7a8a', fontSize: 11 }} />
          <Tooltip contentStyle={{ background: 'var(--bg-panel-raised)', border: '1px solid var(--border)', fontSize: 12 }} />
          {seriesIds.map((id, i) => (
            <Line
              key={id}
              type="monotone"
              dataKey={id}
              stroke={LINE_COLORS[i % LINE_COLORS.length]}
              dot={false}
              strokeWidth={1.5}
              isAnimationActive={false}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}