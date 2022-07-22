package main

import (
	"example/dist_sched/config"
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
	log.SetLevel(log.InfoLevel)
}

func main() {
	var s float64 = 0.0
	for _, v := range config.Load {
		s += v
	}

	log.WithFields(log.Fields{
		"loads": config.Load,
		"number of schedulers": *config.NumSchedulers,
		"expected ratio": s / float64(*config.NumSchedulers),
	}).Info("expected ratio")

	ctl := controller.New()
	ctl.AsServer()

}
