//
// Memory resource.
//
// This collector reports on the following meminfo metrics:
//
//  - "percent" (gauge)
//  - "active" (gauge)
//  - "total" (gauge)
//  - "free" (gauge)
//  - "swap.percent" (gauge)
//  - "swap.total" (gauge)
//  - "swap.free" (gauge)
//
package main

import "github.com/statsd/client-interface"
import "github.com/c9s/goprocinfo/linux"
import log "github.com/sirupsen/logrus"
import "math"
import "time"

// Memory resource.
type Memory struct {
	Path     string
	Interval time.Duration
	Extended bool
	client   statsd.Client
	exit     chan struct{}
}

// NewMemory New memory resource.
func NewMemory(interval time.Duration, extended bool) *Memory {
	return &Memory{
		Path:     "/proc/meminfo",
		Extended: extended,
		Interval: interval,
		exit:     make(chan struct{}),
	}
}

// Name of the resource.
func (m *Memory) Name() string {
	return "memory"
}

// Start resource collection.
func (m *Memory) Start(client statsd.Client) error {
	m.client = client
	go m.Report()
	return nil
}

// Report resource.
func (m *Memory) Report() {
	tick := time.Tick(m.Interval)
	for {
		select {
		case <-tick:
			stat, err := linux.ReadMemInfo(m.Path)

			if err != nil {
				log.Error("memory: %s", err)
				continue
			}

			_ = m.client.Gauge("percent", percent(stat))
			_ = m.client.Gauge("swap.percent", swapPercent(stat))

			if m.Extended {
				_ = m.client.Gauge("total", bytes(stat.MemTotal))
				_ = m.client.Gauge("used", bytes(used(stat)))
				_ = m.client.Gauge("free", bytes(stat.MemFree))
				_ = m.client.Gauge("active", bytes(stat.Active))
				_ = m.client.Gauge("swap.total", bytes(stat.SwapTotal))
				_ = m.client.Gauge("swap.free", bytes(stat.SwapFree))
			}

		case <-m.exit:
			log.Info("mem: exiting")
			return
		}
	}
}

// Stop resource collection.
func (m *Memory) Stop() error {
	close(m.exit)
	return nil
}

// calculate swap percentage.
func swapPercent(s *linux.MemInfo) int {
	total := s.SwapTotal
	used := total - s.SwapFree
	p := float64(used) / float64(total) * 100

	if math.IsNaN(p) {
		return 0
	}

	return int(p)
}

// calculate percentage.
func percent(s *linux.MemInfo) int {
	total := s.MemTotal
	p := float64(used(s)) / float64(total) * 100

	if math.IsNaN(p) {
		return 0
	}

	return int(p)
}

// used memory.
func used(s *linux.MemInfo) uint64 {
	return s.MemTotal - s.MemFree - s.Buffers - s.Cached
}

// convert to bytes.
func bytes(n uint64) int {
	return int(n * 1000)
}
