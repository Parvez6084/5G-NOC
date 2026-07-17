package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	pb "netpulse/proto"

	"google.golang.org/grpc"
	_ "modernc.org/sqlite"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Telemetry mirrors the struct used by the HTTP endpoints
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

	db.SetMaxOpenConns(1)

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatal("failed to read schema.sql:", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal("failed to apply schema:", err)
	}

	log.Println("Database initialized: telemetry.db")
}

// ---------- HTTP handlers (kept for testing/debugging + dashboard reads) ----------

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

	telemetryReceived.WithLabelValues(t.ElementType, "http").Inc()

	if err := insertTelemetry(t); err != nil {
		http.Error(w, "failed to insert: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"stored"}`))
}

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

// insertTelemetry is shared by both the HTTP and gRPC paths
func insertTelemetry(t Telemetry) error {
	start := time.Now()
	_, err := db.Exec(
		`INSERT INTO telemetry (element_id, element_type, timestamp, latency_ms, throughput_mbps, packet_loss_pct, cpu_usage_pct, memory_usage_pct)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ElementID, t.ElementType, t.Timestamp.Format(time.RFC3339),
		t.LatencyMs, t.ThroughputMbps, t.PacketLossPct, t.CPUUsagePct, t.MemoryUsagePct,
	)
	telemetryInsertDuration.Observe(time.Since(start).Seconds())
	return err
}

// ---------- gRPC server ----------

type grpcServer struct {
	pb.UnimplementedTelemetryServiceServer
}

func (s *grpcServer) SendTelemetry(ctx context.Context, r *pb.TelemetryReading) (*pb.TelemetryAck, error) {
	t := Telemetry{
		ElementID:      r.ElementId,
		ElementType:    r.ElementType,
		Timestamp:      r.Timestamp.AsTime(),
		LatencyMs:      r.LatencyMs,
		ThroughputMbps: r.ThroughputMbps,
		PacketLossPct:  r.PacketLossPct,
		CPUUsagePct:    r.CpuUsagePct,
		MemoryUsagePct: r.MemoryUsagePct,
	}

	telemetryReceived.WithLabelValues(t.ElementType, "grpc").Inc()

	if err := insertTelemetry(t); err != nil {
		return &pb.TelemetryAck{Stored: false, Message: err.Error()}, nil
	}
	return &pb.TelemetryAck{Stored: true, Message: "ok"}, nil
}

func startGRPCServer() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal("failed to listen on :9090:", err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterTelemetryServiceServer(grpcSrv, &grpcServer{})

	log.Println("gRPC server listening on :9090")
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatal("grpc serve error:", err)
	}
}

var (
	telemetryReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "noc_telemetry_received_total",
			Help: "Total telemetry readings received, by element type and protocol",
		},
		[]string{"element_type", "protocol"},
	)

	telemetryInsertDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "noc_telemetry_insert_duration_seconds",
			Help:    "Time taken to insert a telemetry reading into SQLite",
			Buckets: prometheus.DefBuckets,
		},
	)

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "noc_http_requests_total",
			Help: "Total HTTP requests, by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)
)

func init() {
	prometheus.MustRegister(telemetryReceived, telemetryInsertDuration, httpRequestsTotal)
}

// ---------- main ----------

func main() {
	initDB()
	defer db.Close()

	go startGRPCServer()

	http.HandleFunc("/telemetry", handleIngest)
	http.HandleFunc("/telemetry/recent", handleRecent)
	http.Handle("/metrics", promhttp.Handler()) // NEW

	log.Println("Telemetry Collector (HTTP) listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
