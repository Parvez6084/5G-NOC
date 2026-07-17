export interface Telemetry {
  element_id: string;
  element_type: string;
  timestamp: string;
  latency_ms: number;
  throughput_mbps: number;
  packet_loss_pct: number;
  cpu_usage_pct: number;
  memory_usage_pct: number;
}

export interface Alert {
  element_id: string;
  reason: string;
  timestamp: string;
}

export type WSMessage =
  | { type: 'telemetry'; data: Telemetry[] }
  | { type: 'alert'; data: Alert };