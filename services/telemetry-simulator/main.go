package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

const collectorURL = "http://localhost:8081/telemetry"

// NetworkElement represents a simulated 5G infrastructure node
type NetworkElement struct {
	ID   string
	Type string // "base-station", "core-node", "edge-device"
}

// Telemetry represents a single metrics snapshot from a network element
type Telemetry struct {
	ElementID      string    `json:"element_id"`
	ElementType    string    `json:"element_type"`
	Timestamp      time.Time `json:"timestamp"`
	LatencyMs      float64   `json:"latency_ms"`
	ThroughputMbps float64   `json:"throughput_mbps"`
	PacketLossPct  float64   `json:"packet_loss_pct"`
	CPUUsagePct    float64   `json:"cpu_usage_pct"`
	MemoryUsagePct float64   `json:"memory_usage_pct"`
}

func generateTelemetry(e NetworkElement) Telemetry {
	isAnomaly := rand.Float64() < 0.05

	latency := 10 + rand.Float64()*20
	packetLoss := rand.Float64() * 0.5

	if isAnomaly {
		latency += 100 + rand.Float64()*100
		packetLoss += 5 + rand.Float64()*10
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

// sendTelemetry POSTs a telemetry reading to the Collector service
func sendTelemetry(t Telemetry) error {
	body, err := json.Marshal(t)
	if err != nil {
		return err
	}

	resp, err := http.Post(collectorURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("collector returned status %d", resp.StatusCode)
	}
	return nil
}

func runElement(e NetworkElement, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t := generateTelemetry(e)
			if err := sendTelemetry(t); err != nil {
				fmt.Printf("[%s] failed to send telemetry: %v\n", e.ID, err)
				continue
			}
			fmt.Printf("[%s] sent: latency=%.1fms loss=%.2f%%\n", e.ID, t.LatencyMs, t.PacketLossPct)
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

	done := make(chan struct{})

	fmt.Println("5G-NOC Telemetry Simulator started. Sending to", collectorURL)
	fmt.Println("Press Ctrl+C to stop.")

	for _, e := range elements {
		go runElement(e, done)
	}

	<-done // blocks forever until we implement graceful shutdown
}
