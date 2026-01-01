package monitors

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/net"
)

type NetMonitor struct {
}

func NewNetMonitor() *NetMonitor {
	return &NetMonitor{}
}

func (n *NetMonitor) Name() string {
	return "Network"
}

func (n *NetMonitor) Check(ctx context.Context) (string, error) {

	netStart, err := net.IOCountersWithContext(ctx, false)

	if err != nil {
		return "", fmt.Errorf("[Network Monitor] Could not retrieve Network info: %v\n", err)
	}

	if len(netStart) == 0 {
		return "", fmt.Errorf("Net percent empty")
	}

	return fmt.Sprintf("Send: %d KB, Recv: %d KB", netStart[0].BytesSent/1024, netStart[0].BytesRecv/1024), nil
}
