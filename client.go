package s4a

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	conn    net.PacketConn
	devAddr *net.UDPAddr

	deviceID uint16
	seq      atomic.Uint32

	mu      sync.Mutex
	pending map[uint16]chan *Frame
}

type ClientOption func(*Client)

func WithDeviceID(id uint16) ClientOption {
	return func(c *Client) { c.deviceID = id }
}

func NewClient(controllerAddr string, opts ...ClientOption) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp", controllerAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve controller address: %w", err)
	}
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	c := &Client{
		conn:     conn,
		devAddr:  addr,
		deviceID: DefaultDeviceID,
		pending:  make(map[uint16]chan *Frame),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.readLoop()
	return c, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) nextSeq() uint16 {
	return uint16(c.seq.Add(1))
}

func (c *Client) sendAndWait(ctx context.Context, f *Frame) (*Frame, error) {
	seq := f.Seq
	ch := make(chan *Frame, 1)

	c.mu.Lock()
	c.pending[seq] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
	}()

	raw := make([]byte, 0, FrameSize(len(f.Data)))
	raw, _ = f.AppendBinary(raw)
	if _, err := c.conn.WriteTo(raw, c.devAddr); err != nil {
		return nil, fmt.Errorf("send frame: %w", err)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) readLoop() {
	buf := make([]byte, 4096)
	for {
		c.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := c.conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		if n < 33 {
			continue
		}

		f := &Frame{}
		if err := f.UnmarshalBinary(buf[:n]); err != nil {
			continue
		}

		c.mu.Lock()
		ch, ok := c.pending[f.Seq]
		c.mu.Unlock()

		if ok {
			select {
			case ch <- f:
			default:
			}
		}
	}
}

func (c *Client) OpenDoor(ctx context.Context, door uint8, duration time.Duration) error {
	f := NewOpenDoorRequest(c.deviceID, c.nextSeq(), door, duration)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseOpenDoorResponse(resp)
}

func (c *Client) Authorize(ctx context.Context, right *AuthRight) error {
	f := NewAuthorizeRequest(c.deviceID, c.nextSeq(), right)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseAuthorizeResponse(resp)
}

func (c *Client) ControlDoor(ctx context.Context, door uint8, cmd DoorControl) error {
	f := NewControlDoorRequest(c.deviceID, c.nextSeq(), door, cmd)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseOpenDoorResponse(resp)
}

func (c *Client) RevokeAuth(ctx context.Context, cardNumber uint64) error {
	f := NewRevokeAuthRequest(c.deviceID, c.nextSeq(), cardNumber)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseRevokeAuthResponse(resp)
}

func (c *Client) ClearAuth(ctx context.Context) error {
	f := NewClearAuthRequest(c.deviceID, c.nextSeq())
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseClearAuthResponse(resp)
}

func (c *Client) MonitorLog(ctx context.Context, index uint32) (*MonitorLogResponse, error) {
	f := NewMonitorLogRequest(c.deviceID, c.nextSeq(), index)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return nil, err
	}
	return ParseMonitorLogResponse(resp)
}

func (c *Client) SetTime(ctx context.Context, t time.Time) error {
	f := NewSetTimeRequest(c.deviceID, c.nextSeq(), t)
	resp, err := c.sendAndWait(ctx, f)
	if err != nil {
		return err
	}
	return ParseSetTimeResponse(resp)
}

type EventListener struct {
	conn *net.UDPConn
}

func NewEventListener(listenAddr string) (*EventListener, error) {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve listen address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	return &EventListener{conn: conn}, nil
}

func (l *EventListener) Close() error {
	return l.conn.Close()
}

func (l *EventListener) ListenEvents(ctx context.Context, handler func(*Event) error) error {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		l.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read event: %w", err)
		}

		evt, err := ParseEvent(buf[:n])
		if err != nil {
			continue
		}
		if err := handler(evt); err != nil {
			return err
		}
	}
}
