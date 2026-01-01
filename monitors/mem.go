package monitors

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/mem"
)

type MemMonitor struct {
}

func NewMemMonitor() *MemMonitor {
	return &MemMonitor{}
}

func (m *MemMonitor) Name() string {
	return "Memory"
}

func (m *MemMonitor) Check(ctx context.Context) (string, error) {

	vmStart, err := mem.VirtualMemoryWithContext(ctx)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%.2f%%", vmStart.UsedPercent), nil
}
