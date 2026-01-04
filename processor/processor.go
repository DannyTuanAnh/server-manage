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
