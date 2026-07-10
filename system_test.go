package s4a

import (
	"testing"
	"time"
)

func TestDoorRef(t *testing.T) {
	ref := doorRef("10.0.0.1:65534", 3)
	if ref != "10.0.0.1:65534:3" {
		t.Errorf("doorRef: got %q", ref)
	}
}

func TestSystemNew(t *testing.T) {
	s := NewSystem()
	if s == nil {
		t.Fatal("NewSystem returned nil")
	}
	if len(s.controllers) != 0 {
		t.Error("expected empty controllers map")
	}
}

func TestSystemInterlockEnableDisable(t *testing.T) {
	s := NewSystem()
	s.EnableInterlock("a:65534", 1, "b:65534", 2)

	refA := doorRef("a:65534", 1)
	refB := doorRef("b:65534", 2)

	if s.interlocks[refA] != refB {
		t.Errorf("interlock A: got %q, want %q", s.interlocks[refA], refB)
	}
	if s.interlocks[refB] != refA {
		t.Errorf("interlock B: got %q, want %q", s.interlocks[refB], refA)
	}

	s.DisableInterlock("a:65534", 1)
	if _, ok := s.interlocks[refA]; ok {
		t.Error("interlock A not removed")
	}
	if _, ok := s.interlocks[refB]; ok {
		t.Error("interlock B not removed")
	}
}

func TestSystemMultiCard(t *testing.T) {
	s := NewSystem()
	s.EnableMultiCard("c:65534", 1, 3)
	ref := doorRef("c:65534", 1)
	if s.multiCard[ref] != 3 {
		t.Errorf("multiCard: got %d, want 3", s.multiCard[ref])
	}
	s.DisableMultiCard("c:65534", 1)
	if _, ok := s.multiCard[ref]; ok {
		t.Error("multiCard not removed")
	}
}

func TestSystemFirstCardOpen(t *testing.T) {
	s := NewSystem()
	s.EnableFirstCardOpen("d:65534", 1)
	ref := doorRef("d:65534", 1)
	if !s.firstCardOpen[ref] {
		t.Error("firstCardOpen not enabled")
	}
	s.DisableFirstCardOpen("d:65534", 1)
	if s.firstCardOpen[ref] {
		t.Error("firstCardOpen not disabled")
	}
}

func TestParseDoorFromEvent(t *testing.T) {
	evt := &Event{DoorNo: "2"}
	door, err := parseDoorFromEvent(evt)
	if err != nil {
		t.Fatalf("parseDoorFromEvent: %v", err)
	}
	if door != 2 {
		t.Errorf("door: got %d, want 2", door)
	}
}

func TestParseDoorFromEventEmpty(t *testing.T) {
	evt := &Event{DoorNo: ""}
	_, err := parseDoorFromEvent(evt)
	if err == nil {
		t.Error("expected error for empty door")
	}
}

func TestParseDoorFromEventDefault(t *testing.T) {
	evt := &Event{DoorNo: "0"}
	door, err := parseDoorFromEvent(evt)
	if err != nil {
		t.Fatalf("parseDoorFromEvent: %v", err)
	}
	if door != 1 {
		t.Errorf("door: got %d, want 1 (default)", door)
	}
}

func TestParseCardNumber(t *testing.T) {
	evt := &Event{CardData: "0008242637"}
	n := parseCardNumber(evt)
	if n != 8242637 {
		t.Errorf("card number: got %d, want 8242637", n)
	}
}

func TestSystemOpenDoorInterlockBlocked(t *testing.T) {
	s := NewSystem()
	s.mu.Lock()
	refA := doorRef("a:65534", 1)
	refB := doorRef("b:65534", 2)
	s.interlocks[refA] = refB
	s.interlocks[refB] = refA
	s.doorState[refB] = true
	s.mu.Unlock()

	err := s.OpenDoor(nil, "a:65534", 1, 3*time.Second)
	if err == nil {
		t.Error("expected inter-lock error")
	}
}

func TestSystemOpenDoorMultiCardBlocked(t *testing.T) {
	s := NewSystem()
	s.mu.Lock()
	ref := doorRef("a:65534", 1)
	s.multiCard[ref] = 3
	s.mu.Unlock()

	err := s.OpenDoor(nil, "a:65534", 1, 3*time.Second)
	if err == nil {
		t.Error("expected multi-card error")
	}
}

func TestHandleCardSwipeMultiCardQueue(t *testing.T) {
	s := NewSystem()
	s.EnableMultiCard("x:65534", 1, 2)

	evt1 := &Event{CardData: "111111", DoorNo: "1"}
	evt2 := &Event{CardData: "222222", DoorNo: "1"}

	handled1, _ := s.HandleCardSwipe(nil, "x:65534", evt1)
	if !handled1 {
		t.Error("first card should be queued")
	}
	ref := doorRef("x:65534", 1)
	if len(s.pendingAuths[ref]) != 1 {
		t.Errorf("pending auths: got %d, want 1", len(s.pendingAuths[ref]))
	}

	handled2, _ := s.HandleCardSwipe(nil, "x:65534", evt2)
	if !handled2 {
		t.Error("second card should be handled (queue cleared)")
	}
	if _, ok := s.pendingAuths[ref]; ok {
		t.Error("pending auths should be cleared after reaching count")
	}
}

func TestHandleCardSwipeFirstCardOpen(t *testing.T) {
	s := NewSystem()
	s.EnableFirstCardOpen("y:65534", 1)
	ref := doorRef("y:65534", 1)

	evt := &Event{CardData: "123456", DoorNo: "1"}
	handled, _ := s.HandleCardSwipe(nil, "y:65534", evt)

	if !handled {
		t.Error("first-card-open should handle the swipe")
	}
	if s.firstCardOpen[ref] {
		t.Error("firstCardOpen should be disabled after first swipe")
	}
}
