package monitors

import (
	"context"
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/disk"
)

type DiskMonitor struct {
}

func NewDiskMonitor() *DiskMonitor {
	return &DiskMonitor{}
}

func (d *DiskMonitor) Name() string {
	return "Disk"
}

func (d *DiskMonitor) Check(ctx context.Context) (string, error) {
	path := "/"

	if runtime.GOOS == "windows" {
		path = "C:\\"
	}

	diskStart, err := disk.UsageWithContext(ctx, path)

	if err != nil {
		return "", fmt.Errorf("[Disk Monitor] Could not retrieve Disk info: %v\n", err)
	}

	return fmt.Sprintf("Total: %.2f GB, Used: %.2f GB, Free: %.2f GB", float64(diskStart.Total)/1024/1024/1024, float64(diskStart.Used)/1024/1024/1024, float64(diskStart.Free)/1024/1024/1024), nil
}
