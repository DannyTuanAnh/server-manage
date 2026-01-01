package processor

import (
	"context"
	"sync"
	"time"

	"danny.com/server_manage_system/models"
)

type MonitorService struct {
	ctx      context.Context
	wg       sync.WaitGroup
	cancel   context.CancelFunc
	interval time.Duration
	monitors []models.Monitor
	Out      chan models.MonitorEvent
}

func NewMonitorService(ctx context.Context, interval time.Duration, monitors ...models.Monitor) *MonitorService {
	ctx, cancel := context.WithCancel(ctx)
	return &MonitorService{
		ctx:      ctx,
		interval: interval,
		cancel:   cancel,
		monitors: monitors,
		Out:      make(chan models.MonitorEvent),
	}
}

func (s *MonitorService) Start() {
	s.wg.Add(1)
	go s.run()
}

func (s *MonitorService) Stop() {
	s.cancel()
}

func (s *MonitorService) Wait() {
	s.wg.Wait()
}

func (s *MonitorService) run() {
	defer s.wg.Done()
	defer close(s.Out)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			var wg sync.WaitGroup

			for _, m := range s.monitors {
				wg.Add(1)

				go func(m models.Monitor) {
					defer wg.Done()

					value, err := m.Check(s.ctx)

					s.Out <- models.MonitorEvent{
						Name:      m.Name(),
						Value:     value,
						Timestamp: time.Now(),
						Err:       err,
					}
				}(m)
			}

			wg.Wait()
		}
	}
}

// func GetTopProcess(ctx context.Context) (string, error) {
// 	vmStat, err := mem.VirtualMemoryWithContext(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("[Get Top Process] Could not retrieve virtual memory info: %v\n", err)
// 	}

// 	totalMemory := vmStat.Total

// 	processes, err := process.ProcessesWithContext(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("[Get Top Process] Could not retrieve process list: %v\n", err)
// 	}

// 	var mu sync.Mutex
// 	var wg sync.WaitGroup
// 	var topProcesses []models.ProcessInfo

// 	for _, p := range processes {
// 		wg.Add(1)

// 		go func(proc *process.Process) {
// 			defer wg.Done()

// 			select {
// 			case <-ctx.Done():
// 				return
// 			default:
// 				name, err := proc.NameWithContext(ctx)
// 				if err != nil {
// 					return
// 				}

// 				cpuPercent, err := proc.CPUPercentWithContext(ctx)
// 				if err != nil {
// 					return
// 				}

// 				//RSS: Resident Set Size => Ram thực sự mà tiến trình đang sử dụng
// 				//VMS: Virtual Memory Size => Tổng bộ nhớ ảo mà hệ điều hành cấp phát cho tiến trình
// 				memInfo, err := proc.MemoryInfoWithContext(ctx)
// 				if err != nil {
// 					return
// 				}

// 				// Tính phần trăm RAM sử dụng của tiến trình
// 				ramPercent := float64(memInfo.RSS) / float64(totalMemory) * 100

// 				//trả về đơn vị là milliseconds
// 				createTime, err := proc.CreateTimeWithContext(ctx)
// 				if err != nil {
// 					return
// 				}

// 				// Tính thời gian chạy của tiến trình, từ milliseconds sang seconds
// 				startTime := time.Unix(createTime/1000, 0)
// 				runningTime := time.Since(startTime)

// 				if cpuPercent > 5 || ramPercent > 5 {
// 					mu.Lock()
// 					topProcesses = append(topProcesses, models.ProcessInfo{
// 						PID:        proc.Pid,
// 						Name:       name,
// 						CPUPercent: cpuPercent,
// 						RamUsed:    memInfo.RSS,
// 						RamPercent: ramPercent,
// 						RunTime:    runningTime,
// 						StartTime:  startTime,
// 					})
// 					mu.Unlock()
// 				}

// 			}
// 		}(p)

// 	}

// 	wg.Wait()

// 	var result string
// 	for _, p := range topProcesses {
// 		result += "\n--------------------"
// 		result += fmt.Sprintf("\nProcess Name: %s\n", p.Name)
// 		result += fmt.Sprintf("CPU: %.2f%%, RAM: %.2f%% (%.2f MB)\n",
// 			p.CPUPercent, p.RamPercent, float64(p.RamUsed)/1024/1024)
// 		result += fmt.Sprintf("Running Time: %v\n", p.RunTime.Round(time.Second))
// 		result += fmt.Sprintf("Start Time: %s\n", p.StartTime.Format("02-01-2006 15:04:05"))

// 	}

// 	return result, nil
// }
