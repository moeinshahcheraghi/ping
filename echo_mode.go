package pingtunnel

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net"
	"sync/atomic"
)

const echoReplayWindow = 4096

// echoRequestConn carries application data only in Echo Requests. Echo Replies
// copy the authenticated request exactly and are never application messages.
// It leaves the legacy framing and the FrameMgr reliability layer unchanged.
type echoRequestConn struct {
	icmpPacketConn
	server   bool
	key      int
	clientID int
	sequence atomic.Uint32
	// Only the single recvICMP goroutine accesses the bounded replay window.
	received map[[32]byte]struct{}
	ring     [echoReplayWindow][32]byte
	next     int
	count    int
}

func wrapEchoConn(conn icmpPacketConn, config *CryptoConfig, server bool, key, clientID int) (icmpPacketConn, error) {
	if config == nil || !config.EchoMode {
		return conn, nil
	}
	if config.Mode == NoEncryption || config.Cipher == nil {
		return nil, errors.New("echo mode requires authenticated encryption")
	}
	if icmpDatagram {
		return nil, errors.New("echo mode requires raw ICMP sockets; unprivileged datagram ICMP is unsupported")
	}
	var seed [2]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return nil, err
	}
	c := &echoRequestConn{icmpPacketConn: conn, server: server, key: key, clientID: clientID,
		received: make(map[[32]byte]struct{})}
	c.sequence.Store(uint32(binary.BigEndian.Uint16(seed[:])))
	return c, nil
}

func (c *echoRequestConn) WriteTo(packet []byte, dst net.Addr) (int, error) {
	if len(packet) < 8 {
		return 0, errors.New("truncated ICMP echo message")
	}
	// The caller may reuse its pooled packet immediately after WriteTo returns.
	out := append([]byte(nil), packet...)
	// Preserve the client session ID: independent clients can share a server
	// and password. The encrypted envelope authenticates this routing field.
	finalizeICMPEcho(out, SEND_PROTO, int(binary.BigEndian.Uint16(packet[4:6])), int(c.sequence.Add(1)))
	return c.icmpPacketConn.WriteTo(out, dst)
}

func validEchoRequest(packet []byte) bool {
	if len(packet) < 8 || packet[0] != byte(SEND_PROTO) || packet[1] != 0 {
		return false
	}
	var sum uint32
	for i := 0; i+1 < len(packet); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(packet[i : i+2]))
	}
	if len(packet)%2 != 0 {
		sum += uint32(packet[len(packet)-1]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return uint16(sum) == 0xffff
}

// accept is called only after AEAD verification, protobuf decoding and magic
// validation. Direction and key checks precede all reply side effects.
func (c *echoRequestConn) accept(packet []byte, src net.Addr, msg *MyMsg) bool {
	if msg.Key != int32(c.key) || (c.server && msg.Rproto != int32(RECV_PROTO)) || (!c.server && msg.Rproto != -1) {
		return false
	}
	if !c.server && int(binary.BigEndian.Uint16(packet[4:6])) != c.clientID {
		return false
	}
	reply := append([]byte(nil), packet...)
	finalizeICMPEcho(reply, RECV_PROTO, int(binary.BigEndian.Uint16(packet[4:6])), int(binary.BigEndian.Uint16(packet[6:8])))
	// Bypass WriteTo: replies must not be converted into fresh data requests.
	// A failed acknowledgement must not prevent application-data delivery.
	_, _ = c.icmpPacketConn.WriteTo(reply, src)
	digest := sha256.Sum256(packet[8:])
	if _, seen := c.received[digest]; seen {
		return false
	}
	if c.count == echoReplayWindow {
		delete(c.received, c.ring[c.next])
	} else {
		c.count++
	}
	c.ring[c.next] = digest
	c.next = (c.next + 1) % echoReplayWindow
	c.received[digest] = struct{}{}
	return true
}
