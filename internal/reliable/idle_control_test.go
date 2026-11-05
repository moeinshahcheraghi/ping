package reliable

import (
	"bytes"
	"testing"
	"time"
)

func TestIdleControlDoesNotEmitEverySecond(t *testing.T) {
	fm := NewFrameMgr(888, 1000000, 256<<10, 512, 400, 0, 0)
	fm.SetIdleControlInterval(5 * time.Second)
	fm.lastPayloadTime.Store(0)
	now := time.Now().UnixNano()
	fm.lastPingTime = now - int64(2*time.Second)
	fm.lastSendHBTime = now - int64(2*time.Second)
	fm.Update()
	frames := fm.GetSendFrames()
	if len(frames) != 0 {
		t.Fatalf("idle connection emitted %d controls after only 2s", len(frames))
	}
	fm.lastPingTime = now - int64(6*time.Second)
	fm.lastSendHBTime = now - int64(6*time.Second)
	fm.Update()
	fm.PutSendFrames(fm.GetSendFrames()) // PING is immediate; HB joins the reliable send window.
	fm.Update()
	if len(fm.GetSendFrames()) == 0 {
		t.Fatal("idle heartbeat was never sent")
	}
}

func TestIdleControlDoesNotDelayPayload(t *testing.T) {
	fm := NewFrameMgr(888, 1000000, 256<<10, 512, 400, 0, 0)
	fm.SetIdleControlInterval(5 * time.Second)
	fm.lastPayloadTime.Store(0)
	fm.lastPingTime = time.Now().UnixNano()
	fm.lastSendHBTime = fm.lastPingTime
	payload := []byte("immediate payload on an idle connection")
	fm.WriteSendBuffer(payload)
	fm.Update()
	frames := fm.GetSendFrames()
	defer fm.PutSendFrames(frames)
	for _, f := range frames {
		if f.Data != nil && f.Data.Type == int32(FrameData_USER_DATA) && bytes.Equal(f.Data.Data, payload) {
			return
		}
	}
	t.Fatal("payload waited for the idle control interval")
}

func TestIdleControlJitterBoundedAndHealthGrace(t *testing.T) {
	fm := NewFrameMgr(888, 1000000, 256<<10, 512, 400, 0, 0)
	fm.SetIdleControlInterval(5 * time.Second)
	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		d := fm.nextIdleInterval()
		if d < 4500*time.Millisecond || d > 5500*time.Millisecond {
			t.Fatalf("interval %s", d)
		}
		seen[d] = true
	}
	if len(seen) < 2 {
		t.Fatal("control interval was not varied")
	}
	fm.lastRecvHBTime = time.Now().Add(-12 * time.Second).UnixNano()
	fm.lastRecvDataTime = fm.lastRecvHBTime
	if fm.IsHBTimeout() {
		t.Fatal("one missing idle heartbeat expired connection health")
	}
}
