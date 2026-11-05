package pingtunnel

import "fmt"

// Bound allocations requested by an authenticated peer before constructing its
// FrameMgr. These are per connection, not per tunnel or per authenticated user.
const maxTCPBufferBytes = 16 << 20

func validateTCPResources(bufferBytes, windowFrames int) error {
	if bufferBytes < FRAME_MAX_SIZE || bufferBytes > maxTCPBufferBytes {
		return fmt.Errorf("tcp_buffer_bytes must be between %d and %d", FRAME_MAX_SIZE, maxTCPBufferBytes)
	}
	if windowFrames <= 0 || windowFrames > DEFAULT_TCP_WIN {
		return fmt.Errorf("tcp_window_frames must be between 1 and %d", DEFAULT_TCP_WIN)
	}
	return nil
}
