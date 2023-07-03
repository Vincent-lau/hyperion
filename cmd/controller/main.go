package main

import (
	"example/dist_sched/config"
	"example/dist_sched/controller"
	"flag"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

func init() {
	flag.Parse()
	// log.SetReportCaller(true)
	rand.Seed(42)

	if *config.Mode == "DEV" {
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
		log.SetLevel(log.DebugLevel)
	} else if *config.Mode == "PROD" {
		log.SetLevel(log.InfoLevel)
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		panic("unknown environment")
	}

}

func main() {

	ctl := controller.New()
	ctl.AsServer()
}
