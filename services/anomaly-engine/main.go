package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const collectorRecentURL = "http://localhost:8081/telemetry/recent"
const alertURL = "http://localhost:8082/alerts"
const windowSize = 20
const stdDevThreshold = 2.0

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

type elementHistory struct {
	latencies    []float64
	packetLosses []float64
}

var history = make(map[string]*elementHistory)

// ---------- Prometheus metrics ----------

var (
	readingsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "noc_anomaly_engine_readings_processed_total",
			Help: "Total telemetry readings processed by the statistical detector, by element",
		},
		[]string{"element_id"},
	)

	anomaliesDetected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "noc_anomaly_engine_anomalies_detected_total",
			Help: "Total anomalies detected, by element",
		},
		[]string{"element_id"},
	)
)

func init() {
	prometheus.MustRegister(readingsProcessed, anomaliesDetected)
}

// ---------- helpers ----------

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

func pushWindow(window []float64, val float64) []float64 {
	window = append(window, val)
	if len(window) > windowSize {
		window = window[1:]
	}
	return window
}

func checkAnomaly(t Telemetry) (bool, string) {
	h, exists := history[t.ElementID]
	if !exists {
		h = &elementHistory{}
		history[t.ElementID] = h
	}

	isAnomaly := false
	reason := ""

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
	resp, err := http.Post(alertURL, "application/json", jsonReader(body))
	if err != nil {
		fmt.Println("failed to send alert to gateway:", err)
		return
	}
	resp.Body.Close()
}

// small helper so we don't need a separate "bytes" import alias issue
func jsonReader(b []byte) *jsonBodyReader {
	return &jsonBodyReader{data: b}
}

type jsonBodyReader struct {
	data []byte
	pos  int
}

func (r *jsonBodyReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// ---------- main ----------

func main() {
	fmt.Println("5G-NOC Anomaly Engine started. Polling", collectorRecentURL)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Anomaly Engine metrics listening on :8084")
		log.Fatal(http.ListenAndServe(":8084", nil))
	}()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	seen := make(map[string]bool)

	for range ticker.C {
		readings, err := fetchRecent()
		if err != nil {
			fmt.Println("failed to fetch telemetry:", err)
			continue
		}

		for i := len(readings) - 1; i >= 0; i-- {
			t := readings[i]
			key := t.ElementID + t.Timestamp.String()
			if seen[key] {
				continue
			}
			seen[key] = true

			readingsProcessed.WithLabelValues(t.ElementID).Inc()

			isAnomaly, reason := checkAnomaly(t)
			if isAnomaly {
				anomaliesDetected.WithLabelValues(t.ElementID).Inc()
				fmt.Printf("🚨 ALERT [%s]: %s\n", t.ElementID, reason)
				sendAlert(t.ElementID, reason)
			}
		}
	}
}
