package monitors

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"danny.com/server_manage_system/models"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

type job struct {
	proc *process.Process
}

// Hệ số làm mượt EMA
// α=1−e^−Δt​/τ

type processSample struct {
	LastCPUTime float64
	LastSample  time.Time
	EMA         float64
	LastSeen    time.Time //for TTL
}

type TopProcessMonitor struct {
	cpuThreshold float64
	ramThreshold float64

	emaWindow time.Duration

	ttl time.Duration

	workers   int
	batchSize int
	topK      int

	cache  sync.Map // map[int32]processSample
	cursor int
}

func NewTopProcessMonitor(cpuThreshold, ramThreshold float64) *TopProcessMonitor {
	return &TopProcessMonitor{
		cpuThreshold: cpuThreshold,
		ramThreshold: ramThreshold,

		emaWindow: 15 * time.Second,

		ttl: 2 * time.Minute,

		workers:   min(4, runtime.NumCPU()/2),
		batchSize: 20,
		topK:      5,
	}
}

func (t *TopProcessMonitor) Name() string {
	return "TopProcess"
}

// Thiết lập cửa sổ thời gian cho EMA
func (t *TopProcessMonitor) SetEMAWindow(d time.Duration) {
	if d <= 0 {
		return
	}
	t.emaWindow = d
}

// Lấy một mảng process dựa trên con trỏ hiện tại theo batch
func (t *TopProcessMonitor) sampleBatch(process []*process.Process) []*process.Process {
	if len(process) == 0 {
		return nil
	}

	start := t.cursor
	end := t.cursor + t.batchSize

	if end > len(process) {
		end = len(process)
	}

	batch := process[start:end]
	t.cursor = end % len(process)

	return batch
}

func (t *TopProcessMonitor) collectProcess(ctx context.Context, proc *process.Process, out chan<- models.ProcessInfo, totalMem uint64) {
	pid := proc.Pid
	now := time.Now()

	cpuTimes, err := proc.TimesWithContext(ctx)
	if err != nil {
		return
	}

	memInfo, err := proc.MemoryInfoWithContext(ctx)
	if err != nil {
		return
	}

	// Tự tính tổng CPU time thay vì dùng Total() vì đã bị
	// deprecated (không khuyến khích dùng) trong phiên bản mới của thư viện gopsutil
	// Total() là hàm nội bộ, tác giả thư viện không muốn người dùng gọi trực tiếp nữa.
	totalCPUTime := cpuTimes.User + cpuTimes.System

	var prev processSample

	if val, ok := t.cache.Load(pid); ok {
		prev = val.(processSample)
	}

	deltaCPU := totalCPUTime - prev.LastCPUTime
	deltaTime := now.Sub(prev.LastSample).Seconds()

	if !prev.LastSample.IsZero() && deltaTime > 0 {
		rawCPU := (deltaCPU / deltaTime) * 100

		alpha := emaAlpha(deltaTime, t.emaWindow)
		smoothed := ema(prev.EMA, rawCPU, alpha)

		ramPercent := float64(memInfo.RSS) / float64(totalMem) * 100

		if smoothed >= t.cpuThreshold || ramPercent >= t.ramThreshold {
			name, err := proc.NameWithContext(ctx)
			if err != nil || name == "" {
				name = fmt.Sprintf("pid-%d", pid)
			}

			create, _ := proc.CreateTimeWithContext(ctx)
			start := time.Unix(create/1000, 0)

			out <- models.ProcessInfo{
				PID:        pid,
				Name:       name,
				CPUPercent: smoothed,
				RamUsed:    memInfo.RSS,
				RamPercent: ramPercent,
				StartTime:  start,
				RunTime:    time.Since(start),
			}
		}

		prev.EMA = smoothed
	}

	t.cache.Store(pid, processSample{
		LastCPUTime: totalCPUTime,
		LastSample:  now,
		LastSeen:    now,
		EMA:         prev.EMA,
	})

}

func (t *TopProcessMonitor) cleanup() {
	now := time.Now()

	t.cache.Range(func(key, value interface{}) bool {
		s := value.(processSample)
		if now.Sub(s.LastSeen) > t.ttl {
			t.cache.Delete(key)
		}
		return true
	})
}

func (t *TopProcessMonitor) Check(ctx context.Context) (string, error) {
	vmStat, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("[Get Top Process] Could not retrieve virtual memory info: %v\n", err)
	}

	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("[Get Top Process] Could not retrieve process list: %v\n", err)
	}

	batch := t.sampleBatch(processes)
	if len(batch) == 0 {
		return "", nil
	}

	out := make(chan models.ProcessInfo, len(batch))
	jobs := make(chan *process.Process)

	var wg sync.WaitGroup
	for i := 0; i < t.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				t.collectProcess(ctx, p, out, vmStat.Total)
			}
		}()
	}

	go func() {
		for _, p := range batch {
			jobs <- p
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	// Tạo 2 heaps: 1 cho CPU, 1 cho RAM
	cpuHeap := &ProcessHeap{sortBy: "cpu"}
	ramHeap := &ProcessHeap{sortBy: "ram"}
	heap.Init(cpuHeap)
	heap.Init(ramHeap)

	for p := range out {
		// Push vào CPU heap
		heap.Push(cpuHeap, p)
		if cpuHeap.Len() > t.topK {
			heap.Pop(cpuHeap)
		}

		// Push vào RAM heap
		heap.Push(ramHeap, p)
		if ramHeap.Len() > t.topK {
			heap.Pop(ramHeap)
		}
	}

	t.cleanup()

	// Format output với cả 2 heaps
	var result string
	result += "\n\n================ Top 5 CPU Consuming Processes ================"
	result += formatProcessList(reverseHeap(cpuHeap))
	result += "\n\n================ Top 5 Memory Consuming Processes ================"
	result += formatProcessList(reverseHeap(ramHeap))

	return result, nil
}

// Định dạng danh sách tiến trình thành chuỗi bảng
func formatProcessList(list []models.ProcessInfo) string {
	if len(list) == 0 {
		return "\n\nNo high resource processes\n"
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

// Hàm tính EMA
func ema(prev, current, alpha float64) float64 {
	if prev == 0 {
		return current
	}
	return alpha*current + (1-alpha)*prev
}

// Hàm lấy phần tử từ heap theo thứ tự ngược lại
func reverseHeap(h *ProcessHeap) []models.ProcessInfo {
	n := len(h.items)
	if n == 0 {
		return []models.ProcessInfo{}
	}

	out := make([]models.ProcessInfo, n)

	// Pop tất cả elements ra
	for i := n - 1; i >= 0; i-- {
		out[i] = heap.Pop(h).(models.ProcessInfo)
	}

	return out
}

// Hàm tính hệ số alpha cho EMA dựa trên cửa sổ thời gian
func emaAlpha(deltaSeconds float64, window time.Duration) float64 {
	tau := window.Seconds()
	if tau <= 0 {
		return 1
	}
	return 1 - math.Exp(-deltaSeconds/tau)
}
