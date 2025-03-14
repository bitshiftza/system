//
// Disk resource.
//
// This collector reports on the following stat metrics:
//
//  - "disk.percent" (gauge)
//  - "disk.free" (gauge)
//  - "disk.used" (gauge)
//
package main

import (
	"github.com/statsd/client-interface"
	log "github.com/sirupsen/logrus"
	"github.com/deniswernert/go-fstab"
	"github.com/c9s/goprocinfo/linux"
	"time"
)

// Disk resource.
type Disk struct {
	Interval time.Duration
	client   statsd.Client
	exit     chan struct{}
}

// NewDisk New disk resource.
func NewDisk(interval time.Duration) *Disk {
	return &Disk{
		Interval: interval,
		exit:     make(chan struct{}),
	}
}

// Name of resource.
func (d *Disk) Name() string {
	return "disk"
}

// Start resource collection.
func (d *Disk) Start(client statsd.Client) error {
	d.client = client
	go d.Report()
	return nil
}

// paths returns the mount-point paths.
func (d *Disk) paths() ([]string, error) {
	mounts, err := fstab.ParseSystem()
	if err != nil {
		return nil, err
	}

	paths := []string{}
	for _, mount := range mounts {
		paths = append(paths, mount.File)
	}

	return paths, nil
}

// Report resources.
func (d *Disk) Report() {
	tick := time.Tick(d.Interval)

	paths, err := d.paths()
	if err != nil {
		log.Errorf("disk: failed to read fstab: %s", err)
		log.Errorf("disk: will not report")
		return
	}

	log.Infof("disk: discovered %v", paths)

	for {
		select {
		case <-tick:
			for _, path := range paths {
				stat, err := linux.ReadDisk(path)

				if err != nil {
					log.Errorf("disk: %s %s", path, err)
					continue
				}

				d.client.Gauge(path+".percent", int(percentDisk(stat.Used, stat.All)))
				d.client.Gauge(path+".free", int(stat.Free))
				d.client.Gauge(path+".used", int(stat.Used))
			}

		case <-d.exit:
			log.Infof("disk: exiting")
			return
		}
	}
}

// Stop resource collection.
func (d *Disk) Stop() error {
	println("stopping disk")
	return nil
}

// calculate percentage.
func percentDisk(a, b uint64) uint64 {
	return uint64(float64(a) / float64(b) * 100)
}
