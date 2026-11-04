package pingtunnel

import (
	"fmt"
	"math/rand/v2"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// ConfigurePadding sets an optional per-packet size variation before Run.
// The bytes are inside AEAD, so old protobuf readers can safely ignore field
// 127 and no new cleartext marker or negotiation is introduced. It does not
// hide ICMP or traffic volume. The maximum excludes the three-byte field header.
func (c *CryptoConfig) ConfigurePadding(max int) error {
	if max < 0 || max > 64 {
		return fmt.Errorf("padding-max must be between 0 and 64")
	}
	if max > 0 && (c.Mode == NoEncryption || c.Cipher == nil) {
		return fmt.Errorf("padding requires authenticated encryption")
	}
	c.paddingMax = max
	return nil
}

// Length selection is not a cryptographic secret. AEAD's fresh nonce makes
// the zero-filled padding indistinguishable from other encrypted plaintext.
// No extra random-data buffer, delay, or packet is needed.
func appendLengthPadding(dst []byte, max int) []byte {
	n := rand.IntN(max + 1)
	dst = protowire.AppendTag(dst, 127, protowire.BytesType)
	dst = protowire.AppendVarint(dst, uint64(n))
	return append(dst, make([]byte, n)...)
}

// Only liveness probes are jittered; application data and ARQ timers are not.
// A dedicated timer avoids quantizing this variation to the maintenance tick.
func jitterHeartbeat(base time.Duration) time.Duration {
	spread := base / 10
	if spread < 1 {
		return base
	}
	return base - spread + time.Duration(rand.Int64N(int64(2*spread+1)))
}
