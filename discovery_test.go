package s4a

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDiscoverTimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// No heartbeat sender — should time out with empty result
	controllers, err := Discover(ctx, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(controllers) != 0 {
		t.Errorf("expected 0 controllers, got %d", len(controllers))
	}
}

func TestDiscoverFindsController(t *testing.T) {
	heartbeatPayload := []byte("24884|120|33|1|TestController|7879047689384705|v1.0.0-test")

	raw := make([]byte, 8+len(heartbeatPayload))
	raw[0] = 0xc8
	raw[1] = EventTypeHeartbeat
	raw[2] = 0x00
	raw[3] = byte(len(heartbeatPayload))
	copy(raw[8:], heartbeatPayload)

	srcAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	dstAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:50000")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch := make(chan []DiscoveredController, 1)
	go func() {
		result, _ := Discover(ctx, 1500*time.Millisecond)
		ch <- result
	}()

	time.Sleep(100 * time.Millisecond)
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		t.Skipf("cannot send test heartbeat: %v", err)
	}
	conn.Write(raw)
	conn.Close()

	controllers := <-ch

	found := false
	for _, c := range controllers {
		if c.Name == "TestController" {
			found = true
			if c.SerialNum != "24884" {
				t.Errorf("SerialNum: got %q, want \"24884\"", c.SerialNum)
			}
			if c.FirmwareVer != "v1.0.0-test" {
				t.Errorf("FirmwareVer: got %q", c.FirmwareVer)
			}
		}
	}
	if !found {
		t.Error("TestController not discovered")
	}
}

func TestDiscoveredControllerString(t *testing.T) {
	dc := DiscoveredController{
		IP:          net.ParseIP("192.168.1.100"),
		Name:        "FrontDoor",
		SerialNum:   "12345",
		FirmwareVer: "v2.0",
	}
	s := dc.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}
