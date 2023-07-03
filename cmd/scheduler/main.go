package main

import (
	"flag"
	"github.com/Vincent-lau/hyperion/internal/configs"
	"github.com/Vincent-lau/hyperion/internal/scheduler"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()
	// log.SetReportCaller(true)

	if *config.Mode == "DEV" {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	} else if *config.Mode == "PROD" {
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
