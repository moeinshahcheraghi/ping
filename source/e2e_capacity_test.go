package pingtunnel

import (
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Keep all connections open while every stream exchanges distinct payloads.
// This measures concurrent tunnel streams, not OpenVPN users or Internet Mbps.
func TestE2E_HighConcurrency(t *testing.T) {
	count := 128
	if value := os.Getenv("PINGTUNNEL_E2E_CONNECTIONS"); value != "" {
		var err error
		count, err = strconv.Atoi(value)
		if err != nil || count < 1 || count > 4096 {
			t.Fatal("PINGTUNNEL_E2E_CONNECTIONS must be between 1 and 4096")
		}
	}
	opts := basePairOpts()
	opts.tcpBS = 256 << 10
	opts.tcpMW = 512
	opts.crypto = CHACHA20
	opts.cryptoKey = "capacity-test-local-password"
	opts.echoMode = true
	opts.paddingMax = 32
	opts.timeout = 300
	p := startPair(t, opts)
	defer p.Close()
	conns := make([]net.Conn, count)
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.Close()
			}
		}
	}()
	jobs := make(chan int, count)
	for i := range conns {
		jobs <- i
	}
	close(jobs)
	results := make(chan error, count)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				c, err := net.DialTimeout("tcp", p.clientAddr, 10*time.Second)
				if err == nil {
					conns[i] = c
					err = capacityEcho(c, i, 0)
				}
				results <- err
			}
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("connection setup: %v", err)
		}
	}
	if p.client.connCount.Load() < int64(count) || p.server.connCount.Load() < int64(count) {
		t.Fatal("not all concurrent streams remained active")
	}
	// Exercise every retained stream again after peak concurrency is reached.
	for i, c := range conns {
		if err := capacityEcho(c, i, 1); err != nil {
			t.Fatalf("retained stream %d: %v", i, err)
		}
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("verified %d concurrent encrypted Echo TCP streams in %s; heap=%d MiB, heap system=%d MiB, goroutines=%d, GOMAXPROCS=%d",
		count, time.Since(start), mem.HeapAlloc>>20, mem.HeapSys>>20, runtime.NumGoroutine(), runtime.GOMAXPROCS(0))
}

func capacityEcho(c net.Conn, id, round int) error {
	payload := []byte(fmt.Sprintf("capacity-stream-%d-round-%d", id, round))
	if err := c.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	if _, err := c.Write(payload); err != nil {
		return err
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(c, got); err != nil {
		return err
	}
	if string(got) != string(payload) {
		return fmt.Errorf("stream %d payload mismatch", id)
	}
	return nil
}
