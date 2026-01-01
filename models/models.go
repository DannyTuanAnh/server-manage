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

var (
	Stat      = make(map[string]MonitorEvent)
	StatMutex sync.Mutex
)
