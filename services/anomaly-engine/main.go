package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

const alertURL = "http://localhost:8082/alerts"
const collectorRecentURL = "http://localhost:8081/telemetry/recent"
const windowSize = 20       // how many past readings per element to remember
const stdDevThreshold = 2.0 // flag if reading is > this many std devs above mean

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

// elementHistory keeps a rolling window of past readings for one network element
type elementHistory struct {
	latencies    []float64
	packetLosses []float64
}

// history tracks rolling windows per element_id
var history = make(map[string]*elementHistory)

func fetchRecent() ([]Telemetry, error) {
	resp, err := http.Get(collectorRecentURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []Telemetry
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

// mean calculates the average of a slice of float64
func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// stdDev calculates the standard deviation of a slice of float64
func stdDev(vals []float64, m float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range vals {
		diff := v - m
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(vals)-1))
}

// pushWindow adds a new value to a rolling window, capped at windowSize
func pushWindow(window []float64, val float64) []float64 {
	window = append(window, val)
	if len(window) > windowSize {
		window = window[1:]
	}
	return window
}

// checkAnomaly compares a new reading against the rolling history for its element.
// Returns (isAnomaly, reason) — only evaluates once we have enough history to judge fairly.
func checkAnomaly(t Telemetry) (bool, string) {
	h, exists := history[t.ElementID]
	if !exists {
		h = &elementHistory{}
		history[t.ElementID] = h
	}

	isAnomaly := false
	reason := ""

	// Only judge once we have a meaningful baseline (avoid false positives on cold start)
	if len(h.latencies) >= 5 {
		m := mean(h.latencies)
		sd := stdDev(h.latencies, m)
		if sd > 0 && t.LatencyMs > m+stdDevThreshold*sd {
			isAnomaly = true
			reason = fmt.Sprintf("latency %.1fms is %.1f std devs above baseline mean %.1fms", t.LatencyMs, (t.LatencyMs-m)/sd, m)
		}
	}

	if len(h.packetLosses) >= 5 {
		m := mean(h.packetLosses)
		sd := stdDev(h.packetLosses, m)
		if sd > 0 && t.PacketLossPct > m+stdDevThreshold*sd {
			isAnomaly = true
			if reason != "" {
				reason += "; "
			}
			reason += fmt.Sprintf("packet loss %.2f%% is %.1f std devs above baseline mean %.2f%%", t.PacketLossPct, (t.PacketLossPct-m)/sd, m)
		}
	}

	// Update rolling windows AFTER checking, so the current reading doesn't skew its own baseline
	h.latencies = pushWindow(h.latencies, t.LatencyMs)
	h.packetLosses = pushWindow(h.packetLosses, t.PacketLossPct)

	return isAnomaly, reason
}

func sendAlert(elementID, reason string) {
	alert := map[string]interface{}{
		"element_id": elementID,
		"reason":     reason,
		"timestamp":  time.Now(),
	}
	body, _ := json.Marshal(alert)
	resp, err := http.Post(alertURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("failed to send alert to gateway:", err)
		return
	}
	resp.Body.Close()
}

func main() {
	fmt.Println("5G-NOC Anomaly Engine started. Polling", collectorRecentURL)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	seen := make(map[string]bool) // avoid re-processing the same reading twice (rough dedup by timestamp+element)

	for range ticker.C {
		readings, err := fetchRecent()
		if err != nil {
			fmt.Println("failed to fetch telemetry:", err)
			continue
		}

		// Process oldest to newest so rolling windows build up in correct order
		for i := len(readings) - 1; i >= 0; i-- {
			t := readings[i]
			key := t.ElementID + t.Timestamp.String()
			if seen[key] {
				continue
			}
			seen[key] = true

			isAnomaly, reason := checkAnomaly(t)
			if isAnomaly {
				fmt.Printf("🚨 ALERT [%s]: %s\n", t.ElementID, reason)
				sendAlert(t.ElementID, reason)
			}
		}
	}
}
