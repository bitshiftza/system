//
// CPU resource.
//
// This collector reports on the following stat metrics:
//
//  - "switches" (counter)
//  - "interrupts" (counter)
//  - "running" (counter)
//  - "blocked" (counter)
//  - "usage" (gauge)
//
package main

import "github.com/statsd/client-interface"
import "github.com/c9s/goprocinfo/linux"
import log "github.com/sirupsen/logrus"
import "time"

// CPU resource.
type CPU struct {
	Path     string
	Interval time.Duration
	Extended bool
	client   statsd.Client
	exit     chan struct{}
}

// NewCPU New CPU resource.
func NewCPU(interval time.Duration, extended bool) *CPU {
	return &CPU{
		Path:     "/proc/stat",
		Extended: extended,
		Interval: interval,
		exit:     make(chan struct{}),
	}
}

// Name of the resource.
func (c *CPU) Name() string {
	return "cpu"
}

// Start resource collection.
func (c *CPU) Start(client statsd.Client) error {
	c.client = client
	go c.Report()
	return nil
}

// Report resource collection.
func (c *CPU) Report() {
	var prevTotal, prevIdle uint64
	prev := new(linux.Stat)
	tick := time.Tick(c.Interval)

	for {
		select {
		case <-tick:
			stat, err := linux.ReadStat(c.Path)

			if err != nil {
				log.Error("cpu: %s", err)
				continue
			}

			_ = c.client.Gauge("percent", int(cpuPercent(&prevIdle, &prevTotal, stat.CPUStatAll)))

			if c.Extended {
				_ = c.client.IncrBy("blocked", int(stat.ProcsBlocked))
				_ = c.client.IncrBy("interrupts", int(stat.Interrupts-prev.Interrupts))
				_ = c.client.IncrBy("switches", int(stat.ContextSwitches-prev.ContextSwitches))
			}

			prev = stat
		case <-c.exit:
			log.Info("cpu: exiting")
			return
		}
	}
}

// Stop resource collection.
func (c *CPU) Stop() error {
	println("stopping cpu")
	close(c.exit)
	return nil
}

// calculate percentage from the previous read
// and adjust the previous values.
func cpuPercent(prevIdle, prevTotal *uint64, s linux.CPUStat) float64 {
	total, idle := totals(s)
	di := idle - *prevIdle
	dt := total - *prevTotal
	*prevIdle = idle
	*prevTotal = total
	return float64(dt-di) / float64(dt) * 100
}

// totals from jiffies.
func totals(s linux.CPUStat) (uint64, uint64) {
	user := s.User - s.Guest
	usernice := s.Nice - s.GuestNice
	idle := s.Idle + s.IOWait
	system := s.System + s.IRQ + s.SoftIRQ
	virt := s.Guest + s.GuestNice
	total := user + usernice + system + idle + s.Steal + virt
	return total, idle
}
