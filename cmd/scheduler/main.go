package main

import (
	"example/dist_sched/scheduler"
	"flag"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()
	// log.SetReportCaller(true)
	rand.Seed(time.Now().UnixNano())

	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
	})
	log.SetLevel(log.InfoLevel)

}

func main() {

	sched := scheduler.New()
	sched.Consensus()

	for {
		time.Sleep(10 * time.Second)
	}
}
