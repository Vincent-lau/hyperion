package main

import (
	"example/dist_sched/config"
	"example/dist_sched/scheduler"
	"flag"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()
	// log.SetReportCaller(true)

	if *config.Mode == "dev" {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	} else if *config.Mode == "prod" {
		log.SetLevel(log.InfoLevel)
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		panic("unknown environment")
	}

}

func main() {

	sched := scheduler.New()
	sched.Schedule()

	for {
		time.Sleep(10 * time.Second)
	}
}
