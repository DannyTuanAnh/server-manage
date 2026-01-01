package monitors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"danny.com/server_manage_system/models"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type TopProcessMonitor struct {
	cpuThreshold float64
	ramThreshold float64
}

func NewTopProcessMonitor(cpuThreshold, ramThreshold float64) *TopProcessMonitor {
	return &TopProcessMonitor{
		cpuThreshold: cpuThreshold,
		ramThreshold: ramThreshold,
	}
}

func (t *TopProcessMonitor) Name() string {
	return "TopProcess"
}

func (t *TopProcessMonitor) Check(ctx context.Context) (string, error) {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("[Get Top Process] Could not retrieve virtual memory info: %v\n", err)
	}

	totalMemory := vmStat.Total

	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("[Get Top Process] Could not retrieve process list: %v\n", err)
	}

	var (
		mu           sync.Mutex
		wg           sync.WaitGroup
		topProcesses []models.ProcessInfo
	)

	for _, p := range processes {
		wg.Add(1)

		go func(proc *process.Process) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				name, err := proc.NameWithContext(ctx)
				if err != nil {
					return
				}

				cpuPercent, err := proc.CPUPercentWithContext(ctx)
				if err != nil {
					return
				}

				//RSS: Resident Set Size => Ram thực sự mà tiến trình đang sử dụng
				//VMS: Virtual Memory Size => Tổng bộ nhớ ảo mà hệ điều hành cấp phát cho tiến trình
				memInfo, err := proc.MemoryInfoWithContext(ctx)
				if err != nil {
					return
				}

				// Tính phần trăm RAM sử dụng của tiến trình
				ramPercent := float64(memInfo.RSS) / float64(totalMemory) * 100

				//trả về đơn vị là milliseconds
				createTime, err := proc.CreateTimeWithContext(ctx)
				if err != nil {
					return
				}

				// Tính thời gian chạy của tiến trình, từ milliseconds sang seconds
				startTime := time.Unix(createTime/1000, 0)
				runningTime := time.Since(startTime)

				if cpuPercent > t.cpuThreshold || ramPercent > t.ramThreshold {
					mu.Lock()
					topProcesses = append(topProcesses, models.ProcessInfo{
						PID:        proc.Pid,
						Name:       name,
						CPUPercent: cpuPercent,
						RamUsed:    memInfo.RSS,
						RamPercent: ramPercent,
						RunTime:    runningTime,
						StartTime:  startTime,
					})
					mu.Unlock()
				}

			}
		}(p)

	}

	wg.Wait()

	return formatProcessList(topProcesses), nil
}

func formatProcessList(list []models.ProcessInfo) string {
	if len(list) == 0 {
		return "No high resource processes"
	}

	var result string
	result += "\n┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐"
	result += fmt.Sprintf("\n│ %-9s %-25s %8s %14s %14s %15s %19s  │", "PID", "Process Name", "CPU %", "RAM %", "RAM MB", "Running", "Started")
	result += "\n├─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤"
	for _, p := range list {
		result += fmt.Sprintf("\n│ %-9d %-25s %7.2f%% %13.2f%% %14.2f %14s %21s │", p.PID, p.Name, p.CPUPercent, p.RamPercent, float64(p.RamUsed)/1024/1024, p.RunTime.Round(time.Second), p.StartTime.Format("15:04 02-01-2006"))
	}
	result += "\n└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘"
	return result
}
