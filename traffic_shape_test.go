package pingtunnel

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func TestPaddingCompatibilityAndExactEcho(t *testing.T) {
	for _, mode := range []EncryptionMode{AES128, AES256, CHACHA20} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			cfg, err := NewCryptoConfig(mode, "padding-test-password")
			if err != nil {
				t.Fatal(err)
			}
			cfg.EchoMode = true
			// The receiving peer has no padding configured: mixed-version operation.
			receiver := *cfg
			if err := cfg.ConfigurePadding(64); err != nil {
				t.Fatal(err)
			}
			payload := bytes.Repeat([]byte{0x81}, 888)
			sizes := map[int]bool{}
			for i := 0; i < 128; i++ {
				packet := echoWirePacket(t, cfg, false, 42, payload)
				sizes[len(packet)] = true
				decoded, replies := receiveEcho(t, &receiver, true, packet)
				if len(decoded) != 1 || len(replies) != 1 || !bytes.Equal(decoded[0].my.Data, payload) || !bytes.Equal(replies[0][8:], packet[8:]) {
					t.Fatal("padding changed delivery or exact Echo reply")
				}
				// The old decoder also accepts the extension, without DiscardUnknown.
				c := receiver
				c.echoID = 123
				plain, err := c.Decrypt(packet[8:])
				if err != nil {
					t.Fatal(err)
				}
				old := &MyMsg{}
				if err := proto.Unmarshal(plain, old); err != nil || !bytes.Equal(old.Data, payload) {
					t.Fatal("old protobuf reader incompatible", err)
				}
			}
			if len(sizes) < 16 {
				t.Fatalf("insufficient size variation: %d", len(sizes))
			}
		})
	}
}

func TestPaddingBounds(t *testing.T) {
	c := echoConfig(t)
	for _, n := range []int{-1, 65, 1000000} {
		if c.ConfigurePadding(n) == nil {
			t.Fatalf("accepted %d", n)
		}
	}
	if (&CryptoConfig{}).ConfigurePadding(1) == nil {
		t.Fatal("accepted plaintext padding")
	}
	for _, n := range []int{0, 1, 32, 64} {
		if err := c.ConfigurePadding(n); err != nil {
			t.Fatal(err)
		}
		plain, _ := proto.Marshal(&MyMsg{Data: []byte("test")})
		if n == 0 {
			continue
		}
		for i := 0; i < 100; i++ {
			padded := appendLengthPadding(append([]byte(nil), plain...), n)
			if len(padded) < len(plain)+3 || len(padded) > len(plain)+3+n {
				t.Fatal("padding outside bounds")
			}
		}
	}
}

func TestHeartbeatJitterBounds(t *testing.T) {
	for _, base := range []time.Duration{time.Second, 3 * time.Second, 10 * time.Second} {
		seen := map[time.Duration]bool{}
		for i := 0; i < 100; i++ {
			d := jitterHeartbeat(base)
			if d < base-base/10 || d > base+base/10 {
				t.Fatal("heartbeat outside liveness bounds")
			}
			seen[d] = true
		}
		if len(seen) < 2 {
			t.Fatal("heartbeat remains periodic")
		}
	}
}

func BenchmarkEncryptedPadding(b *testing.B) {
	for _, n := range []int{0, 32, 64} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			cfg, _ := NewCryptoConfig(CHACHA20, "benchmark-padding-password")
			plain := make([]byte, 950, 1100)
			dst := make([]byte, 0, 1200)
			b.ReportAllocs()
			b.SetBytes(950)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := plain[:950]
				if n > 0 {
					p = appendLengthPadding(p, n)
				}
				if _, err := cfg.EncryptInto(dst[:0], p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
