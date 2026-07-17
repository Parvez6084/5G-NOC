package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const collectorRecentURL = "http://localhost:8081/telemetry/recent"

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

type Alert struct {
	ElementID string    `json:"element_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// wsMessage wraps outgoing data so the dashboard can distinguish message types
type wsMessage struct {
	Type string      `json:"type"` // "telemetry" or "alert"
	Data interface{} `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // fine for local dev
}

// hub manages all connected WebSocket clients safely across goroutines
type hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *hub) add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

// broadcast sends a message to every connected client, dropping any that error out
func (h *hub) broadcast(msg wsMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteJSON(msg); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

var connections = newHub()

// handleWS upgrades an HTTP connection to WebSocket and registers it with the hub
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade failed:", err)
		return
	}
	connections.add(conn)
	log.Println("dashboard connected, total clients:", len(connections.clients))

	// Keep the connection open; we don't expect messages from the client, just detect disconnects
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			connections.remove(conn)
			log.Println("dashboard disconnected")
			return
		}
	}
}

// handleAlert receives anomaly alerts POSTed from the Anomaly Engine and broadcasts them
func handleAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var a Alert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	connections.broadcast(wsMessage{Type: "alert", Data: a})
	w.WriteHeader(http.StatusOK)
}

// pollAndBroadcastTelemetry periodically fetches recent telemetry and pushes it to all clients
func pollAndBroadcastTelemetry() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		resp, err := http.Get(collectorRecentURL)
		if err != nil {
			log.Println("failed to fetch telemetry:", err)
			continue
		}
		var readings []Telemetry
		if err := json.NewDecoder(resp.Body).Decode(&readings); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		connections.broadcast(wsMessage{Type: "telemetry", Data: readings})
	}
}

func main() {
	go pollAndBroadcastTelemetry()

	http.HandleFunc("/ws", handleWS)
	http.HandleFunc("/alerts", handleAlert)

	fmt.Println("API Gateway listening on :8082 (WebSocket at /ws)")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
