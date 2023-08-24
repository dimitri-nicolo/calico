package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSocketFile(t *testing.T) {
	// start test with socket file already existing
	fp := filepath.Join(t.TempDir(), "dikastes.sock")
	_, err := os.Create(fp)
	if err != nil {
		t.Fatalf("cannot start test -- error creating socket file: %v", err)
		return
	}

	// run the server. test should fail with Fatal() or timeout. if not, ready succeeds
	s := NewDikastesServer(WithListenArguments("unix", fp))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	ready := make(chan struct{}, 1)
	go s.Serve(ctx, ready)
	select {
	case <-ready:
	case <-ctx.Done():
		t.Error("waiting for server to be ready timed out")
	}
}
