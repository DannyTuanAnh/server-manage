package main

import (
	"context"
	"fmt"
	"time"

	"danny.com/server_manage_system/models"
	"danny.com/server_manage_system/monitors"
	"danny.com/server_manage_system/processor"
)

func main() {
	ctx := context.Background()
	cpuMonitor := monitors.NewCPUMonitor()
	memMonitor := monitors.NewMemMonitor()
	netMonitor := monitors.NewNetMonitor()
	diskMonitor := monitors.NewDiskMonitor()
	topProcessMonitor := monitors.NewTopProcessMonitor(0, 0)
	topProcessMonitor.SetEMAWindow(20 * time.Second)

	service := processor.NewMonitorService(ctx, 2*time.Second, cpuMonitor, memMonitor, netMonitor, diskMonitor, topProcessMonitor)

	go func() {
		for ev := range service.Out {
			models.StatMutex.Lock()
			models.Stat[ev.Name] = ev
			models.StatMutex.Unlock()
		}
	}()

	go func() {
		time.Sleep(3 * time.Second)

		printTicker := time.NewTicker(3 * time.Second)
		defer printTicker.Stop()

		for range printTicker.C {
			fmt.Println("\n================== System Status =================")

			models.StatMutex.Lock()

			for _, stat := range models.Stat {
				if stat.Err != nil {
					fmt.Printf("[%s] [%s] error: %v\n", stat.Timestamp.Format("02-01-2006 15:04:05"), stat.Name, stat.Err)
					continue
				}
				fmt.Printf("[%s] [%s]: %s\n", stat.Timestamp.Format("02-01-2006 15:04:05"), stat.Name, stat.Value)
			}

			models.StatMutex.Unlock()
		}
	}()

	service.Start()

	// time.Sleep(30 * time.Second)

	service.Wait()
	service.Stop()
}
