package monitors

import (
	"danny.com/server_manage_system/models"
)

type ProcessHeap struct {
	items  []models.ProcessInfo
	sortBy string // "cpu" hoặc "ram"
}

func (h ProcessHeap) Len() int { return len(h.items) }

func (h ProcessHeap) Less(i, j int) bool {
	if h.sortBy == "ram" {
		return h.items[i].RamPercent < h.items[j].RamPercent
	}
	// Mặc định sắp xếp theo CPU
	return h.items[i].CPUPercent < h.items[j].CPUPercent
}

func (h ProcessHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *ProcessHeap) Push(x interface{}) {
	h.items = append(h.items, x.(models.ProcessInfo))
}

func (h *ProcessHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[:n-1]
	return x
}
