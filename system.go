package s4a

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type System struct {
	mu          sync.RWMutex
	controllers map[string]*Controller

	interlocks map[string]string

	multiCard map[string]int

	firstCardOpen map[string]bool

	pendingAuths map[string]map[uint64]struct{}

	doorState map[string]bool
}

func doorRef(controllerAddr string, door uint8) string {
	return fmt.Sprintf("%s:%d", controllerAddr, door)
}

func NewSystem() *System {
	return &System{
		controllers:   make(map[string]*Controller),
		interlocks:    make(map[string]string),
		multiCard:     make(map[string]int),
		firstCardOpen: make(map[string]bool),
		pendingAuths:  make(map[string]map[uint64]struct{}),
		doorState:     make(map[string]bool),
	}
}

func (s *System) AddController(ct *Controller) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.controllers[ct.Addr] = ct
}

func (s *System) RemoveController(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.controllers, addr)
}

func (s *System) Controller(addr string) *Controller {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.controllers[addr]
}

func (s *System) EnableInterlock(ctrlAddrA string, doorA uint8, ctrlAddrB string, doorB uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	refA := doorRef(ctrlAddrA, doorA)
	refB := doorRef(ctrlAddrB, doorB)
	s.interlocks[refA] = refB
	s.interlocks[refB] = refA
}

func (s *System) DisableInterlock(ctrlAddr string, door uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := doorRef(ctrlAddr, door)
	counterpart := s.interlocks[ref]
	delete(s.interlocks, ref)
	delete(s.interlocks, counterpart)
}

func (s *System) EnableMultiCard(ctrlAddr string, door uint8, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := doorRef(ctrlAddr, door)
	s.multiCard[ref] = count
}

func (s *System) DisableMultiCard(ctrlAddr string, door uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := doorRef(ctrlAddr, door)
	delete(s.multiCard, ref)
}

func (s *System) EnableFirstCardOpen(ctrlAddr string, door uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := doorRef(ctrlAddr, door)
	s.firstCardOpen[ref] = true
}

func (s *System) DisableFirstCardOpen(ctrlAddr string, door uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := doorRef(ctrlAddr, door)
	delete(s.firstCardOpen, ref)
}

func (s *System) OpenDoor(ctx context.Context, ctrlAddr string, door uint8, duration time.Duration) error {
	s.mu.Lock()
	ref := doorRef(ctrlAddr, door)

	if counterpart, ok := s.interlocks[ref]; ok {
		if s.doorState[counterpart] {
			s.mu.Unlock()
			return fmt.Errorf("s4a: inter-lock active: door %s is blocked by %s", ref, counterpart)
		}
	}

	if required, ok := s.multiCard[ref]; ok && required > 1 {
		s.mu.Unlock()
		return fmt.Errorf("s4a: door %s requires %d cards for multi-card access", ref, required)
	}

	ct, ok := s.controllers[ctrlAddr]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("s4a: controller %s not found", ctrlAddr)
	}
	s.mu.Unlock()

	if err := ct.OpenDoor(ctx, door, duration); err != nil {
		return err
	}

	s.mu.Lock()
	s.doorState[ref] = true
	s.mu.Unlock()

	if duration > 0 {
		time.AfterFunc(duration, func() {
			s.mu.Lock()
			delete(s.doorState, ref)
			s.mu.Unlock()
		})
	}
	return nil
}

func (s *System) HandleCardSwipe(ctx context.Context, ctrlAddr string, event *Event) (bool, error) {
	s.mu.Lock()

	door, err := parseDoorFromEvent(event)
	if err != nil {
		s.mu.Unlock()
		return false, nil
	}
	ref := doorRef(ctrlAddr, door)

	if s.firstCardOpen[ref] {
		delete(s.firstCardOpen, ref)
		ct, ok := s.controllers[ctrlAddr]
		s.mu.Unlock()
		if ok {
			ct.OpenDoor(ctx, door, 0)
			ct.SetDoorMode(door, DoorUnlocked)
		}
		return true, nil
	}

	if required, ok := s.multiCard[ref]; ok && required > 1 {
		return s.handleMultiCardAuth(ctx, ctrlAddr, door, event, required)
	}

	s.mu.Unlock()
	return false, nil
}

func (s *System) handleMultiCardAuth(ctx context.Context, ctrlAddr string, door uint8, event *Event, required int) (bool, error) {
	ref := doorRef(ctrlAddr, door)

	if s.pendingAuths[ref] == nil {
		s.pendingAuths[ref] = make(map[uint64]struct{})
	}

	cardNum := parseCardNumber(event)
	s.pendingAuths[ref][cardNum] = struct{}{}

	if len(s.pendingAuths[ref]) >= required {
		delete(s.pendingAuths, ref)
		ct, ok := s.controllers[ctrlAddr]
		s.mu.Unlock()
		if ok {
			ct.OpenDoor(ctx, door, 3*time.Second)
		}
		return true, nil
	}

	s.mu.Unlock()
	return true, nil
}

func (s *System) LockDoor(ctx context.Context, ctrlAddr string, door uint8) error {
	s.mu.Lock()
	ref := doorRef(ctrlAddr, door)
	delete(s.firstCardOpen, ref)
	delete(s.doorState, ref)
	ct, ok := s.controllers[ctrlAddr]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("s4a: controller %s not found", ctrlAddr)
	}
	ct.SetDoorMode(door, DoorLocked)
	return ct.ControlDoor(ctx, door, KeepClosed)
}

func (s *System) UnlockDoor(ctx context.Context, ctrlAddr string, door uint8) error {
	s.mu.Lock()
	delete(s.firstCardOpen, doorRef(ctrlAddr, door))
	ct, ok := s.controllers[ctrlAddr]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("s4a: controller %s not found", ctrlAddr)
	}
	ct.SetDoorMode(door, DoorUnlocked)
	return ct.ControlDoor(ctx, door, RestoreAuto)
}

func (s *System) RestoreAuto(ctx context.Context, ctrlAddr string, door uint8) error {
	s.mu.Lock()
	ref := doorRef(ctrlAddr, door)
	delete(s.firstCardOpen, ref)
	delete(s.doorState, ref)
	ct, ok := s.controllers[ctrlAddr]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("s4a: controller %s not found", ctrlAddr)
	}
	ct.SetDoorMode(door, DoorNormal)
	return ct.ControlDoor(ctx, door, RestoreAuto)
}

func (s *System) DoorState(ctrlAddr string, door uint8) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.doorState[doorRef(ctrlAddr, door)]
}

func (s *System) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ct := range s.controllers {
		ct.Close()
	}
	s.controllers = make(map[string]*Controller)
	return nil
}

func parseDoorFromEvent(event *Event) (uint8, error) {
	if event.DoorNo == "" {
		return 0, fmt.Errorf("s4a: no door in event")
	}
	door := uint8(0)
	for _, c := range event.DoorNo {
		if c >= '0' && c <= '9' {
			door = door*10 + uint8(c-'0')
		}
	}
	if door == 0 {
		door = 1
	}
	return door, nil
}

func parseCardNumber(event *Event) uint64 {
	var n uint64
	for _, c := range event.CardData {
		if c >= '0' && c <= '9' {
			n = n*10 + uint64(c-'0')
		}
	}
	return n
}
