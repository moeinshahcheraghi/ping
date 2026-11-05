package pingtunnel

import (
	"testing"

	"github.com/esrrhs/gohome/network"
)

func TestConfigureTCPResources(t *testing.T) {
	c := &Client{}
	if err := c.ConfigureTCP(0, 0); err != nil {
		t.Fatal(err)
	}
	if c.tcpmode_buffersize != DEFAULT_TCP_BUF || c.tcpmode_maxwin != DEFAULT_TCP_WIN {
		t.Fatal("zero options must preserve historical defaults")
	}
	if err := c.ConfigureTCP(256<<10, 512); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]int{{-1, 512}, {FRAME_MAX_SIZE - 1, 512}, {maxTCPBufferBytes + 1, 512}, {256 << 10, -1}, {256 << 10, DEFAULT_TCP_WIN + 1}} {
		if err := c.ConfigureTCP(pair[0], pair[1]); err == nil {
			t.Fatalf("accepted invalid resource limits %v", pair)
		}
		if c.tcpmode_buffersize != 256<<10 || c.tcpmode_maxwin != 512 {
			t.Fatal("invalid options changed the previous configuration")
		}
	}
	cfg, err := LoadConfig(writeConfig(t, `{"tcp_buffer_bytes":262144,"tcp_window_frames":512}`))
	if err != nil || cfg.TCPBufferBytes != 256<<10 || cfg.TCPWindowFrames != 512 {
		t.Fatalf("resource config: %+v, %v", cfg, err)
	}
}

func TestServerRejectsUnboundedTCPResources(t *testing.T) {
	for _, size := range []int32{-1, 0, int32(FRAME_MAX_SIZE - 1), maxTCPBufferBytes + 1} {
		s, rec := testServer(t)
		packet := testPacketTCP("invalid-buffer", "127.0.0.1:9")
		packet.my.TcpmodeBuffersize = size
		if s.processDataPacketNewConn("invalid-buffer", packet) != nil || rec.writeCount() != 1 {
			t.Fatalf("buffer %d did not receive a refusal", size)
		}
		if s.isConnError(packet.my.Target) {
			t.Fatal("invalid resource request must be rejected before dialing the target")
		}
	}
}

func BenchmarkTCPResourceAllocation(b *testing.B) {
	for _, profile := range []struct {
		name     string
		buf, win int
	}{{"default", DEFAULT_TCP_BUF, DEFAULT_TCP_WIN}, {"many_connections", 256 << 10, 512}} {
		b.Run(profile.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				fm := network.NewFrameMgr(FRAME_MAX_SIZE, FRAME_MAX_ID, profile.buf, profile.win, DEFAULT_TCP_RESEND, 0, 0)
				if fm.GetSendBufferLeft() != profile.buf {
					b.Fatal("incorrect buffer size")
				}
			}
		})
	}
}
