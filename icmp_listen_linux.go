//go:build linux && !android

package pingtunnel

import (
	"net"

	"github.com/esrrhs/gohome/loggo"
)

// Size these once per shared underlay, not once per forwarded connection.
// Kernel limits may clamp the request; no global sysctl is changed.
const icmpSocketBufferBytes = 4 << 20

func listenICMP(addr string) (*net.IPConn, error) {
	setICMPDatagram(false)
	local, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenIP("ip4:icmp", local)
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadBuffer(icmpSocketBufferBytes); err != nil {
		loggo.Warn("ICMP receive buffer request failed: %v", err)
	}
	if err := conn.SetWriteBuffer(icmpSocketBufferBytes); err != nil {
		loggo.Warn("ICMP send buffer request failed: %v", err)
	}
	return conn, nil
}
