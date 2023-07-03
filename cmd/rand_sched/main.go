package main

import (
	"flag"
	"github.com/Vincent-lau/hyperion/internal/rand_sched"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})

}

func main() {

	rand_sched.Schedule()

	log.Debug("done")
	for {
		time.Sleep(10 * time.Second)
	}

}
