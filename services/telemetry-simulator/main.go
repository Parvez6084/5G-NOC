package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "netpulse/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const collectorAddr = "localhost:9090"

type NetworkElement struct {
	ID   string
	Type string
}

func generateTelemetry(e NetworkElement) *pb.TelemetryReading {
	isAnomaly := rand.Float64() < 0.05

	latency := 10 + rand.Float64()*20
	packetLoss := rand.Float64() * 0.5

	if isAnomaly {
		latency += 100 + rand.Float64()*100
		packetLoss += 5 + rand.Float64()*10
	}

	return &pb.TelemetryReading{
		ElementId:      e.ID,
		ElementType:    e.Type,
		Timestamp:      timestamppb.Now(),
		LatencyMs:      latency,
		ThroughputMbps: 50 + rand.Float64()*450,
		PacketLossPct:  packetLoss,
		CpuUsagePct:    20 + rand.Float64()*60,
		MemoryUsagePct: 30 + rand.Float64()*50,
	}
}

func runElement(e NetworkElement, client pb.TelemetryServiceClient, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t := generateTelemetry(e)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			ack, err := client.SendTelemetry(ctx, t)
			cancel()

			if err != nil {
				fmt.Printf("[%s] gRPC send failed: %v\n", e.ID, err)
				continue
			}
			fmt.Printf("[%s] sent via gRPC: latency=%.1fms loss=%.2f%% (ack: %s)\n", e.ID, t.LatencyMs, t.PacketLossPct, ack.Message)
		case <-done:
			return
		}
	}
}

func main() {
	conn, err := grpc.NewClient(collectorAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("failed to connect to collector via gRPC:", err)
		return
	}
	defer conn.Close()

	client := pb.NewTelemetryServiceClient(conn)

	elements := []NetworkElement{
		{ID: "bs-001", Type: "base-station"},
		{ID: "bs-002", Type: "base-station"},
		{ID: "core-001", Type: "core-node"},
		{ID: "edge-001", Type: "edge-device"},
	}

	done := make(chan struct{})

	fmt.Println("5G-NOC Telemetry Simulator started. Sending via gRPC to", collectorAddr)
	fmt.Println("Press Ctrl+C to stop.")

	for _, e := range elements {
		go runElement(e, client, done)
	}

	<-done
}
