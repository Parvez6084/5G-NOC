package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// NetworkElement represents a simulated 5G infrastructure node
type NetworkElement struct {
	ID   string
	Type string // "base-station", "core-node", "edge-device"
}

// Telemetry represents a single metrics snapshot from a network element
type Telemetry struct {
	ElementID   string    `json:"element_id"`
	ElementType string    `json:"element_type"`
	Timestamp   time.Time `json:"timestamp"`
	LatencyMs   float64   `json:"latency_ms"`
	ThroughputMbps float64 `json:"throughput_mbps"`
	PacketLossPct  float64 `json:"packet_loss_pct"`
	CPUUsagePct    float64 `json:"cpu_usage_pct"`
	MemoryUsagePct float64 `json:"memory_usage_pct"`
}

// generateTelemetry creates a realistic-ish random metrics snapshot.
// Occasionally injects an anomaly spike so our future anomaly engine has something to detect.
func generateTelemetry(e NetworkElement) Telemetry {
	// 5% chance of an anomalous spike, to give the anomaly engine something to catch later
	isAnomaly := rand.Float64() < 0.05

	latency := 10 + rand.Float64()*20 // normal: 10-30ms
	packetLoss := rand.Float64() * 0.5 // normal: 0-0.5%

	if isAnomaly {
		latency += 100 + rand.Float64()*100 // spike: +100-200ms
		packetLoss += 5 + rand.Float64()*10 // spike: +5-15%
	}

	return Telemetry{
		ElementID:      e.ID,
		ElementType:    e.Type,
		Timestamp:      time.Now(),
		LatencyMs:      latency,
		ThroughputMbps: 50 + rand.Float64()*450,
		PacketLossPct:  packetLoss,
		CPUUsagePct:    20 + rand.Float64()*60,
		MemoryUsagePct: 30 + rand.Float64()*50,
	}
}

// runElement simulates one network element continuously emitting telemetry
func runElement(e NetworkElement, out chan<- Telemetry, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			out <- generateTelemetry(e)
		case <-done:
			return
		}
	}
}

func main() {
	elements := []NetworkElement{
		{ID: "bs-001", Type: "base-station"},
		{ID: "bs-002", Type: "base-station"},
		{ID: "core-001", Type: "core-node"},
		{ID: "edge-001", Type: "edge-device"},
	}

	telemetryChan := make(chan Telemetry, 100)
	done := make(chan struct{})

	// Start one goroutine per network element
	for _, e := range elements {
		go runElement(e, telemetryChan, done)
	}

	fmt.Println("NetPulse AI Telemetry Simulator started. Simulating", len(elements), "network elements...")
	fmt.Println("Press Ctrl+C to stop.")

	// For now, just print each telemetry reading as it comes in
	for t := range telemetryChan {
		jsonBytes, _ := json.Marshal(t)
		fmt.Println(string(jsonBytes))
	}
}