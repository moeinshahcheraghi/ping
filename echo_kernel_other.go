//go:build !linux || android

package pingtunnel

func kernelEchoReplies(icmpPacketConn) bool { return false }
