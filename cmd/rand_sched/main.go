package main

import (
	"example/dist_sched/rand_sched"
	"flag"
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
