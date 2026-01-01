package models

import (
	"context"
	"sync"
	"time"
)

type MonitorEvent struct {
	Name      string
	Value     string
	Timestamp time.Time
	Err       error
}

type Monitor interface {
	Check(ctx context.Context) (string, error)
	Name() string
}

type ProcessInfo struct {
	PID        int32
	Name       string
	CPUPercent float64
	RamUsed    uint64
	RamPercent float64
	RunTime    time.Duration
	StartTime  time.Time
}

var (
	Stat      = make(map[string]MonitorEvent)
	StatMutex sync.Mutex
)
