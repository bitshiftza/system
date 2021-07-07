package main

import (
	"github.com/statsd/client-namespace"
)
import (
	log "github.com/sirupsen/logrus"
	"github.com/statsd/client"
	"github.com/tj/docopt"
	. "github.com/tj/go-gracefully"
	"os"
	"time"
)

const Version = "0.3.0"

const Usage = `
  Usage:
    system-stats
      [--statsd-address addr]
      [--memory-interval i]
      [--disk-interval i]
      [--cpu-interval i]
      [--extended]
      [--name name]
    system-stats -h | --help
    system-stats --version

  Options:
    --statsd-address addr   statsd address [default: :8125]
    --memory-interval i     memory reporting interval [default: 10s]
    --disk-interval i       disk reporting interval [default: 30s]
    --cpu-interval i        cpu reporting interval [default: 5s]
    --extended              output additional extended metrics
    --name name             node name defaulting to hostname [default: hostname]
    -h, --help              output help information
    -v, --version           output version
`

func main() {
	args, err := docopt.Parse(Usage, nil, true, Version, false)
	if err != nil {
		log.Fatalf("could not parse options: %v", err)
	}

	log.Info("starting system %s", Version)

	client, err := statsd.Dial(args["--statsd-address"].(string))
	if err != nil {
		log.Fatalf("could not initialize stastd client: %v", err)
	}

	extended := args["--extended"].(bool)

	name := args["--name"].(string)
	if "hostname" == name {
		host, err := os.Hostname()
		if err != nil {
			log.Fatalf("could not get hostname: %v", err)
		}
		name = host
	}

	c := NewCollector(namespace.New(client, name))
	c.Add(NewMemory(interval(args, "--memory-interval"), extended))
	c.Add(NewCPU(interval(args, "--cpu-interval"), extended))
	c.Add(NewDisk(interval(args, "--disk-interval")))

	err = c.Start()
	if err != nil {
		log.Fatalf("could not start collector: %v", err)
	}
	Shutdown()
	err = c.Stop()
	if err != nil {
		log.Fatalf("could not stop collector: %v", err)
	}
}

func interval(args map[string]interface{}, name string) time.Duration {
	d, err := time.ParseDuration(args[name].(string))
	if err != nil {
		log.Fatalf("could not parse duration: %v", err)
	}
	return d
}
