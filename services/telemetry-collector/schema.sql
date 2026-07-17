CREATE TABLE IF NOT EXISTS telemetry (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    element_id TEXT NOT NULL,
    element_type TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    latency_ms REAL,
    throughput_mbps REAL,
    packet_loss_pct REAL,
    cpu_usage_pct REAL,
    memory_usage_pct REAL,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_telemetry_element_id ON telemetry(element_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_timestamp ON telemetry(timestamp);