//go:build linux && !android

package pingtunnel

import (
	"net"
	"os"
	"strings"

	"github.com/esrrhs/gohome/loggo"
	"golang.org/x/net/icmp"
)

// A native Linux raw ICMP socket receives requests that the kernel already
// answers. Sending a second application reply wastes bandwidth and creates
// duplicate responses on the wire. Never change the host's ping policy here.
func kernelEchoReplies(conn icmpPacketConn) bool {
	switch conn.(type) {
	case *net.IPConn, *icmp.PacketConn:
	default:
		return false
	}
	value, err := os.ReadFile("/proc/sys/net/ipv4/icmp_echo_ignore_all")
	if err == nil && strings.TrimSpace(string(value)) == "1" {
		loggo.Warn("Echo mode: kernel ICMP Echo replies are disabled; ordinary ping is also disabled by this host setting. Using application replies for authenticated tunnel packets; host settings were not changed")
	}
	return err == nil && strings.TrimSpace(string(value)) == "0"
}
