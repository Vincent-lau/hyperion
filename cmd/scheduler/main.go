package main

import (
	"example/dist_sched/scheduler"
	"flag"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	environment = "dev"
)

func init() {
	flag.Parse()
	// log.SetReportCaller(true)
	rand.Seed(time.Now().UnixNano())

	if environment == "dev" {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	} else if environment == "prod" {
		log.SetLevel(log.InfoLevel)
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		panic("unknown environment")
	}

}

func main() {

	sched := scheduler.New()
	sched.LoopConsensus()

	for {
		time.Sleep(10 * time.Second)
	}
}
