package s4a

import (
	"context"
	"fmt"
	"net"
	"time"
)

type DiscoveredController struct {
	IP          net.IP
	SerialNum   string
	Name        string
	GlobalFlag  string
	FirmwareVer string
	TimeoutCfg  string
	TimeoutLeft string
	TimeoutCnt  string
}

func (d DiscoveredController) String() string {
	return fmt.Sprintf("%s (%s) at %s — fw %s", d.Name, d.SerialNum, d.IP, d.FirmwareVer)
}

func Discover(ctx context.Context, listenDuration time.Duration) ([]DiscoveredController, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DefaultEventPort))
	if err != nil {
		return nil, fmt.Errorf("resolve listen address: %w", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	defer conn.Close()

	seen := make(map[string]DiscoveredController)
	deadline := time.After(listenDuration)
	buf := make([]byte, 4096)

	for {
		select {
		case <-deadline:
			return collectResults(seen), nil
		case <-ctx.Done():
			return collectResults(seen), nil
		default:
		}

		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return collectResults(seen), ctx.Err()
			}
			continue
		}

		evt, err := ParseEvent(buf[:n])
		if err != nil || evt.Type != EventTypeHeartbeat {
			continue
		}

		key := remoteAddr.IP.String()
		if _, exists := seen[key]; !exists {
			seen[key] = DiscoveredController{
				IP:          cloneIP(remoteAddr.IP),
				SerialNum:   evt.HBControllerFlag,
				Name:        evt.HBControllerName,
				GlobalFlag:  evt.HBGlobalFlag,
				FirmwareVer: evt.HBFirmwareVersion,
				TimeoutCfg:  evt.HBTimeoutConfig,
				TimeoutLeft: evt.HBTimeoutRemain,
				TimeoutCnt:  evt.HBTimeoutCount,
			}
		}
	}
}

func collectResults(seen map[string]DiscoveredController) []DiscoveredController {
	result := make([]DiscoveredController, 0, len(seen))
	for _, c := range seen {
		result = append(result, c)
	}
	return result
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}
