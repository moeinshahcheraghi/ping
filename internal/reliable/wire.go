package reliable

import "github.com/esrrhs/gohome/network"

// Reuse the upstream protobuf types; no second descriptor or wire format.
type Frame = network.Frame
type FrameData = network.FrameData
type Congestion = network.Congestion
type BBCongestion = network.BBCongestion

const (
	Frame_ACK           = network.Frame_ACK
	Frame_DATA          = network.Frame_DATA
	Frame_PING          = network.Frame_PING
	Frame_PONG          = network.Frame_PONG
	Frame_REQ           = network.Frame_REQ
	FrameData_CLOSE     = network.FrameData_CLOSE
	FrameData_CONN      = network.FrameData_CONN
	FrameData_CONNRSP   = network.FrameData_CONNRSP
	FrameData_HALFCLOSE = network.FrameData_HALFCLOSE
	FrameData_HB        = network.FrameData_HB
	FrameData_USER_DATA = network.FrameData_USER_DATA
)
