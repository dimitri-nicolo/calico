package collector

import (
	"context"
	"net"
	"testing"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestServer(t *testing.T) {
	t.Skip("skipping test, still under construction")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logEntryFn := func(log *accesslogv3.HTTPAccessLogEntry) {
	}
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	gs := grpc.NewServer()
	server := NewLoggingServer(logEntryFn)
	server.RegisterAccessLogServiceServer(gs)

	go func() {
		if err := gs.Serve(lis); err != nil {
			panic("failed to serve " + err.Error())
		}
	}()

	cc, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	client := v3.NewAccessLogServiceClient(cc)
	streamer, err := client.StreamAccessLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to stream: %v", err)
	}
	if err := streamer.Send(&v3.StreamAccessLogsMessage{}); err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	gs.GracefulStop()
	cancel()
}
