package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

// Telemetry mirrors the struct sent by the simulator service
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

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "telemetry.db")
	if err != nil {
		log.Fatal("failed to open database:", err)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatal("failed to read schema.sql:", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal("failed to apply schema:", err)
	}

	log.Println("Database initialized: telemetry.db")
}

// handleIngest receives a telemetry reading via POST and stores it
func handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var t Telemetry
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		`INSERT INTO telemetry (element_id, element_type, timestamp, latency_ms, throughput_mbps, packet_loss_pct, cpu_usage_pct, memory_usage_pct)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ElementID, t.ElementType, t.Timestamp.Format(time.RFC3339),
		t.LatencyMs, t.ThroughputMbps, t.PacketLossPct, t.CPUUsagePct, t.MemoryUsagePct,
	)
	if err != nil {
		http.Error(w, "failed to insert: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"stored"}`))
}

// handleRecent returns the most recent telemetry readings
func handleRecent(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(
		`SELECT element_id, element_type, timestamp, latency_ms, throughput_mbps, packet_loss_pct, cpu_usage_pct, memory_usage_pct
		 FROM telemetry ORDER BY id DESC LIMIT 50`,
	)
	if err != nil {
		http.Error(w, "query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Telemetry
	for rows.Next() {
		var t Telemetry
		var ts string
		if err := rows.Scan(&t.ElementID, &t.ElementType, &ts, &t.LatencyMs, &t.ThroughputMbps, &t.PacketLossPct, &t.CPUUsagePct, &t.MemoryUsagePct); err != nil {
			http.Error(w, "scan failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		t.Timestamp, _ = time.Parse(time.RFC3339, ts)
		results = append(results, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/telemetry", handleIngest)        // POST to store
	http.HandleFunc("/telemetry/recent", handleRecent) // GET to query

	log.Println("Telemetry Collector listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
