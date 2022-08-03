package main

import (
	"example/dist_sched/controller"
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
	log.SetLevel(log.DebugLevel)
}

func main() {

	ctl := controller.New()
	ctl.AsServer()

}
