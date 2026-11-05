package pingtunnel

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDecryptIntoPreservesEchoCiphertext(t *testing.T) {
	for _, mode := range []EncryptionMode{AES128, AES256, CHACHA20} {
		t.Run(mode.String(), func(t *testing.T) {
			cfg, err := NewCryptoConfig(mode, "receiver-buffer-test-password")
			if err != nil {
				t.Fatal(err)
			}
			cfg.EchoMode = true
			cfg.echoID = 123
			payload := bytes.Repeat([]byte("authenticated payload"), 40)
			wire, err := cfg.Encrypt(payload)
			if err != nil {
				t.Fatal(err)
			}
			original := append([]byte(nil), wire...)
			scratch := make([]byte, 0, len(payload))
			for i := 0; i < 3; i++ {
				plain, err := cfg.DecryptInto(scratch, wire)
				if err != nil || !bytes.Equal(plain, payload) {
					t.Fatalf("decode: %v", err)
				}
				if &plain[0] != &scratch[:cap(scratch)][0] {
					t.Fatal("receiver scratch was not reused")
				}
				if !bytes.Equal(wire, original) {
					t.Fatal("exact Echo reply ciphertext was overwritten")
				}
			}
			wire[len(wire)-1] ^= 1
			if _, err := cfg.DecryptInto(scratch, wire); err == nil {
				t.Fatal("accepted unauthenticated payload")
			}
		})
	}
}

// A second endpoint may open its raw socket while the first is receiving.
func TestICMPModeConcurrentAccess(t *testing.T) {
	old := icmpDatagram.Load()
	defer setICMPDatagram(old)
	var wg sync.WaitGroup
	ip := &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			setICMPDatagram(false)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 10000; i++ {
			_ = icmpDstAddr(ip)
		}
	}()
	wg.Wait()
}

func TestEchoModeKernelReplyStillDeliversAndRejectsReplay(t *testing.T) {
	cfg := echoConfig(t)
	request := echoWirePacket(t, cfg, false, 42, []byte("kernel replied already"))
	var exit atomic.Bool
	raw := &echoScript{fakeICMPConn: &fakeICMPConn{responses: [][]byte{request, request}}, exit: &exit}
	wrapped, err := wrapEchoConn(raw, cfg, true, 42, 123)
	if err != nil {
		t.Fatal(err)
	}
	wrapped.(*echoRequestConn).kernelReplies = true
	recv := make(chan *Packet, 2)
	var wg sync.WaitGroup
	wg.Add(1)
	recvICMP(&wg, &exit, wrapped, recv, cfg)
	if len(recv) != 1 || raw.writeCount() != 0 {
		t.Fatalf("delivered=%d extra replies=%d", len(recv), raw.writeCount())
	}
	if got := <-recv; !bytes.Equal(got.my.Data, []byte("kernel replied already")) {
		t.Fatal("payload changed")
	}
}

func BenchmarkEchoSendDiscard(b *testing.B) {
	cfg, err := NewCryptoConfig(CHACHA20, "benchmark-echo-password")
	if err != nil {
		b.Fatal(err)
	}
	cfg.EchoMode = true
	conn, err := wrapEchoConn(discardICMP{}, cfg, false, 42, 123)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 888)
	dst := &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendICMP(123, i, conn, dst, "", "benchmark", uint32(MyMsg_DATA), data,
			SEND_PROTO, RECV_PROTO, 42, 1, DEFAULT_TCP_BUF, DEFAULT_TCP_WIN, DEFAULT_TCP_RESEND, 0, 0, 60, cfg)
	}
}

// Unlike the recording fake, this sink does not allocate or retain packets.
type discardICMP struct{}

func (discardICMP) WriteTo(b []byte, _ net.Addr) (int, error) { return len(b), nil }
func (discardICMP) ReadFrom([]byte) (int, net.Addr, error)    { return 0, nil, fakeTimeoutError{} }
func (discardICMP) SetReadDeadline(_ time.Time) error         { return nil }
func (discardICMP) Close() error                              { return nil }
