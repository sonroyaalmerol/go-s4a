package s4a

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type ControllerInfo struct {
	DeviceID    uint16
	SerialNum   string
	Name        string
	GlobalFlag  string
	FirmwareVer string
	LastSeen    time.Time
	DoorCount   int
	AuthCount   uint32
	LogCount    uint32
	CurrentTime time.Time
}

type Controller struct {
	Addr string
	mu   sync.RWMutex
	c    *Client
	info ControllerInfo

	doorModes map[uint8]DoorMode

	doorsOpen map[uint8]bool
}

type DoorMode int

const (
	DoorNormal DoorMode = iota
	DoorUnlocked
	DoorConditionallyUnlocked
	DoorLocked
)

func NewController(addr string, opts ...ClientOption) (*Controller, error) {
	client, err := NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("s4a: new controller at %s: %w", addr, err)
	}
	return &Controller{
		Addr:      addr,
		c:         client,
		info:      ControllerInfo{},
		doorModes: make(map[uint8]DoorMode),
		doorsOpen: make(map[uint8]bool),
	}, nil
}

func (ct *Controller) Close() error {
	return ct.c.Close()
}

func (ct *Controller) RefreshInfo(ctx context.Context) error {
	resp, err := ct.c.MonitorLog(ctx, 0)
	if err != nil {
		return fmt.Errorf("refresh controller info: %w", err)
	}
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.info.AuthCount = resp.AuthCount
	ct.info.LogCount = resp.LogCount
	ct.info.DoorCount = 1
	ct.info.LastSeen = time.Now()

	if len(resp.CurrentTime) >= 7 {
		ct.info.CurrentTime = time.Date(
			2000+int(resp.CurrentTime[0]),
			time.Month(resp.CurrentTime[1]),
			int(resp.CurrentTime[2]),
			int(resp.CurrentTime[3]),
			int(resp.CurrentTime[4]),
			int(resp.CurrentTime[5]),
			0,
			time.Local,
		)
	}
	if len(resp.DeviceFlag) >= 5 {
		ct.info.SerialNum = string(resp.DeviceFlag)
	}
	return nil
}

func (ct *Controller) Info() ControllerInfo {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.info
}

func (ct *Controller) OpenDoor(ctx context.Context, door uint8, duration uint16) error {
	if err := ct.checkInterlock(door); err != nil {
		return err
	}
	if err := ct.c.OpenDoor(ctx, door, duration); err != nil {
		return fmt.Errorf("open door %d: %w", door, err)
	}
	ct.mu.Lock()
	ct.doorsOpen[door] = true
	ct.mu.Unlock()

	if duration > 0 && duration < 65000 {
		doorDur := time.Duration(duration) * 10 * time.Millisecond
		time.AfterFunc(doorDur, func() {
			ct.mu.Lock()
			delete(ct.doorsOpen, door)
			ct.mu.Unlock()
		})
	}
	return nil
}

func (ct *Controller) Authorize(ctx context.Context, right *AuthRight) error {
	return ct.c.Authorize(ctx, right)
}

func (ct *Controller) RevokeAuth(ctx context.Context, cardHigh, cardLow uint32) error {
	return ct.c.RevokeAuth(ctx, cardHigh, cardLow)
}

func (ct *Controller) ClearAuth(ctx context.Context) error {
	return ct.c.ClearAuth(ctx)
}

func (ct *Controller) SyncTime(ctx context.Context) error {
	return ct.c.SetTime(ctx, time.Now())
}

func (ct *Controller) SetDoorMode(door uint8, mode DoorMode) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.doorModes[door] = mode
}

func (ct *Controller) GetDoorMode(door uint8) DoorMode {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	if mode, ok := ct.doorModes[door]; ok {
		return mode
	}
	return DoorNormal
}

func (ct *Controller) IsDoorOpen(door uint8) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.doorsOpen[door]
}

func (ct *Controller) checkInterlock(door uint8) error {
	if mode := ct.GetDoorMode(door); mode == DoorLocked {
		return fmt.Errorf("s4a: door %d is locked", door)
	}
	return nil
}

func (ct *Controller) Client() *Client {
	return ct.c
}
