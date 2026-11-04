package pingtunnel

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

// The scripted underlay stops the synchronous receiver after its final packet.
type echoScript struct {
	*fakeICMPConn
	exit *atomic.Bool
}

func (s *echoScript) ReadFrom(b []byte) (int, net.Addr, error) {
	if s.respIdx >= len(s.responses) {
		s.exit.Store(true)
		return 0, nil, fakeTimeoutError{}
	}
	return s.fakeICMPConn.ReadFrom(b)
}

func echoConfig(t *testing.T) *CryptoConfig {
	t.Helper()
	c, err := NewCryptoConfig(CHACHA20, "echo-test-password")
	if err != nil {
		t.Fatal(err)
	}
	c.EchoMode = true
	return c
}

func echoWirePacket(t *testing.T, c *CryptoConfig, server bool, key int, data []byte) []byte {
	t.Helper()
	raw := &fakeICMPConn{}
	conn, err := wrapEchoConn(raw, c, server, key, 123)
	if err != nil {
		t.Fatal(err)
	}
	sproto, rproto := SEND_PROTO, RECV_PROTO
	if server {
		sproto, rproto = RECV_PROTO, -1
	}
	sendICMP(123, 7, conn, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}, "127.0.0.1:2083", "test", uint32(MyMsg_DATA), data,
		sproto, rproto, key, 0, 0, 0, 0, 0, 0, 60, c)
	if raw.writeCount() != 1 {
		t.Fatalf("writes: %d", raw.writeCount())
	}
	return raw.written[0]
}

func receiveEcho(t *testing.T, c *CryptoConfig, server bool, packets ...[]byte) ([]*Packet, [][]byte) {
	t.Helper()
	var exit atomic.Bool
	raw := &echoScript{fakeICMPConn: &fakeICMPConn{responses: packets}, exit: &exit}
	conn, err := wrapEchoConn(raw, c, server, 42, 123)
	if err != nil {
		t.Fatal(err)
	}
	recv := make(chan *Packet, len(packets))
	var wg sync.WaitGroup
	wg.Add(1)
	recvICMP(&wg, &exit, conn, recv, c)
	wg.Wait()
	close(recv)
	var out []*Packet
	for p := range recv {
		out = append(out, p)
	}
	return out, raw.written
}

func TestEchoModeWireAndExactReplies(t *testing.T) {
	c := echoConfig(t)
	for _, server := range []bool{false, true} {
		for _, data := range [][]byte{nil, []byte("odd"), bytes.Repeat([]byte{0x99}, 888)} {
			request := echoWirePacket(t, c, server, 42, data)
			if !validEchoRequest(request) {
				t.Fatal("invalid Echo Request")
			}
			packets, replies := receiveEcho(t, c, !server, request, request)
			if len(packets) != 1 || len(replies) != 2 {
				t.Fatalf("deliveries=%d replies=%d", len(packets), len(replies))
			}
			if !bytes.Equal(packets[0].my.Data, data) {
				t.Fatal("data changed")
			}
			for _, reply := range replies {
				want := append([]byte(nil), request...)
				finalizeICMPEcho(want, RECV_PROTO, int(binary.BigEndian.Uint16(request[4:6])), int(binary.BigEndian.Uint16(request[6:8])))
				if !bytes.Equal(reply, want) {
					t.Fatal("Echo Reply checksum or payload differs from exact reply")
				}
				if reply[0] != byte(RECV_PROTO) || reply[1] != 0 || !bytes.Equal(reply[4:], request[4:]) {
					t.Fatal("reply did not echo id, sequence and every payload byte")
				}
				check := append([]byte(nil), reply...)
				check[0] = byte(SEND_PROTO)
				finalizeICMPEcho(check, SEND_PROTO, int(binary.BigEndian.Uint16(check[4:6])), int(binary.BigEndian.Uint16(check[6:8])))
				if !bytes.Equal(check, request) {
					t.Fatal("reply geometry changed")
				}
			}
		}
	}
}

func TestEchoModeRejectsInvalidAndReflectedPackets(t *testing.T) {
	c := echoConfig(t)
	good := echoWirePacket(t, c, false, 42, []byte("secret"))
	wrongKey := echoWirePacket(t, c, false, 43, []byte("secret"))
	reflected := echoWirePacket(t, c, true, 42, []byte("secret"))
	corrupt := append([]byte(nil), good...)
	corrupt[len(corrupt)-1] ^= 1
	finalizeICMPEcho(corrupt, SEND_PROTO, 1, 1) // valid checksum, invalid AEAD
	badChecksum := append([]byte(nil), good...)
	badChecksum[2] ^= 1
	reply := append([]byte(nil), good...)
	finalizeICMPEcho(reply, RECV_PROTO, 1, 1)
	badCode := append([]byte(nil), good...)
	badCode[1] = 1
	wrongID := append([]byte(nil), good...)
	finalizeICMPEcho(wrongID, SEND_PROTO, 124, 1)
	legacy := *c
	legacy.EchoMode = false
	legacyPacket := echoWirePacket(t, &legacy, false, 42, []byte("secret"))
	packets, replies := receiveEcho(t, c, true, nil, []byte{8}, make([]byte, 7), wrongKey, reflected, corrupt, badChecksum, reply, badCode, wrongID, legacyPacket)
	if len(packets) != 0 || len(replies) != 0 {
		t.Fatalf("invalid input produced %d deliveries and %d replies", len(packets), len(replies))
	}
}

func TestEchoModeCryptographicSeparation(t *testing.T) {
	paired := echoConfig(t)
	legacy := *paired
	legacy.EchoMode = false
	for _, mode := range []EncryptionMode{AES128, AES256, CHACHA20} {
		cfg, err := NewCryptoConfig(mode, "echo-test-password")
		if err != nil {
			t.Fatal(err)
		}
		old := *cfg
		cfg.EchoMode = true
		ciphertext, err := cfg.Encrypt([]byte("test"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = old.Decrypt(ciphertext); err == nil {
			t.Fatal("legacy accepted echo-mode ciphertext")
		}
		ciphertext, err = old.Encrypt([]byte("test"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = cfg.Decrypt(ciphertext); err == nil {
			t.Fatal("echo mode accepted legacy ciphertext")
		}
	}
	for _, cfg := range []*CryptoConfig{{EchoMode: true}, {Mode: CHACHA20, EchoMode: true}} {
		if _, err := wrapEchoConn(&fakeICMPConn{}, cfg, false, 42, 1); err == nil {
			t.Fatal("accepted missing AEAD")
		}
	}
	raw := &fakeICMPConn{}
	conn, err := wrapEchoConn(raw, &legacy, false, 42, 1)
	if err != nil || conn != raw {
		t.Fatal("legacy transport changed")
	}
}

func TestEchoModeSequencesAndBoundedReplay(t *testing.T) {
	cfg := echoConfig(t)
	raw := &fakeICMPConn{}
	wrapped, err := wrapEchoConn(raw, cfg, false, 42, 123)
	if err != nil {
		t.Fatal(err)
	}
	c := wrapped.(*echoRequestConn)
	c.sequence.Store(65534)
	packet := make([]byte, 9)
	for i := 0; i < 3; i++ {
		if _, err := c.WriteTo(packet, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}); err != nil {
			t.Fatal(err)
		}
	}
	for i, w := range raw.written {
		if !validEchoRequest(w) || binary.BigEndian.Uint16(w[6:8]) != uint16(65535+i) {
			t.Fatal("invalid sequence wrap")
		}
	}
	msg := &MyMsg{Key: 42, Rproto: -1}
	for i := 0; i < echoReplayWindow+50; i++ {
		w := make([]byte, 12)
		binary.BigEndian.PutUint16(w[4:6], 123)
		binary.BigEndian.PutUint32(w[8:], uint32(i))
		if !c.accept(w, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}, msg) {
			t.Fatal("new authenticated record treated as replay")
		}
	}
	if len(c.received) != echoReplayWindow {
		t.Fatalf("replay state grew to %d", len(c.received))
	}
}

func TestEchoModeClientRouting(t *testing.T) {
	cfg := echoConfig(t)
	packet := echoWirePacket(t, cfg, true, 42, []byte("server response"))
	var exit atomic.Bool
	raw := &echoScript{fakeICMPConn: &fakeICMPConn{responses: [][]byte{packet}}, exit: &exit}
	conn, err := wrapEchoConn(raw, cfg, false, 42, 124)
	if err != nil {
		t.Fatal(err)
	}
	recv := make(chan *Packet, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	recvICMP(&wg, &exit, conn, recv, cfg)
	if len(recv) != 0 || raw.writeCount() != 0 {
		t.Fatal("client accepted or echoed another client's packet")
	}
}

func TestLoadConfigEchoMode(t *testing.T) {
	c, err := LoadConfig(writeConfig(t, `{"echo_mode":true}`))
	if err != nil || !c.EchoMode {
		t.Fatalf("echo mode not loaded: %v", err)
	}
}

func TestEchoModeConcurrentSend(t *testing.T) {
	cfg := echoConfig(t)
	raw := &fakeICMPConn{}
	conn, err := wrapEchoConn(raw, cfg, false, 42, 123)
	if err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 16)
	binary.BigEndian.PutUint16(packet[4:6], 123)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 32; j++ {
				if _, err := conn.WriteTo(packet, &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if raw.writeCount() != 512 {
		t.Fatalf("writes=%d", raw.writeCount())
	}
	seen := make(map[uint16]bool)
	for _, p := range raw.written {
		seq := binary.BigEndian.Uint16(p[6:8])
		if !validEchoRequest(p) || seen[seq] {
			t.Fatal("invalid checksum or reused concurrent sequence")
		}
		seen[seq] = true
	}
	if !bytes.Equal(packet, append(make([]byte, 4), append([]byte{0, 123}, make([]byte, 10)...)...)) {
		t.Fatal("caller buffer modified")
	}
}
