package pingtunnel

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestE2E_EchoMode(t *testing.T) {
	for _, kind := range []string{"tcp", "socks5", "udp"} {
		t.Run(kind, func(t *testing.T) {
			opts := basePairOpts()
			opts.crypto = CHACHA20
			opts.cryptoKey = "echo-e2e-test-password"
			opts.echoMode = true
			opts.direct = kind != "socks5"
			opts.udp = kind == "udp"
			p := startPair(t, opts)
			defer p.Close()
			if opts.udp {
				conn, err := net.Dial("udp", p.clientAddr)
				if err != nil {
					t.Fatal(err)
				}
				defer conn.Close()
				if err = conn.SetDeadline(time.Now().Add(8 * time.Second)); err != nil {
					t.Fatal(err)
				}
				payload := []byte("echo-mode UDP through encrypted ICMP")
				if _, err = conn.Write(payload); err != nil {
					t.Fatal(err)
				}
				b := make([]byte, 2048)
				n, err := conn.Read(b)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(b[:n], payload) {
					t.Fatal("UDP payload changed")
				}
				return
			}
			var conn net.Conn
			if kind == "socks5" {
				conn = socks5Connect(t, p.clientAddr, p.echoAddr, 10*time.Second)
			} else {
				var err error
				conn, err = waitForPort(p.clientAddr, 5*time.Second)
				if err != nil {
					t.Fatal(err)
				}
			}
			defer conn.Close()
			payload := make([]byte, 128*1024)
			fillPattern(payload, 57)
			echoRoundTrip(t, conn, payload, 30*time.Second)
		})
	}
}

func TestE2E_EchoModeTwoClients(t *testing.T) {
	opts := basePairOpts()
	opts.crypto = CHACHA20
	opts.cryptoKey = "echo-e2e-shared-password"
	opts.echoMode = true
	p := startPair(t, opts)
	defer p.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", getFreePort(t))
	c2, err := NewClient(addr, "127.0.0.1", p.echoAddr, 60, p.key, "127.0.0.1", false, false, 0, p.client.cryptoConfig, "", "", "bb")
	if err != nil {
		t.Fatal(err)
	}
	if c2.id == p.client.id {
		c2.id = 1 + (p.client.id % 65534)
	}
	if err = c2.Run(); err != nil {
		t.Fatal(err)
	}
	defer c2.Stop()
	a, err := waitForPort(p.clientAddr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := waitForPort(addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for i := 0; i < 3; i++ {
		echoRoundTrip(t, a, bytes.Repeat([]byte{byte(i + 1)}, 8192), 15*time.Second)
		echoRoundTrip(t, b, bytes.Repeat([]byte{byte(i + 101)}, 8192), 15*time.Second)
	}
}
