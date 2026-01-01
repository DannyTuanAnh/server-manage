package monitors

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUMonitor struct {
}

func NewCPUMonitor() *CPUMonitor {
	return &CPUMonitor{}
}

func (c *CPUMonitor) Name() string {
	return "CPU"
}

func (c *CPUMonitor) Check(ctx context.Context) (string, error) {

	cpuStart, err := cpu.PercentWithContext(ctx, 1*time.Second, false)

	if err != nil {
		return "", fmt.Errorf("[CPU Monitor] Could not retrieve CPU info: %w", err)
	}

	if len(cpuStart) == 0 {
		return "", fmt.Errorf("Cpu percent empty")
	}

	return fmt.Sprintf("%.2f%%", cpuStart[0]), nil
}
