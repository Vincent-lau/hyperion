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
	log.SetLevel(log.DebugLevel)
}

func main() {
	var load float64 = 0.0
	var used float64 = 0.0
	var cap float64 = 0.0
	for i := range config.Load {
		load += config.Load[i]
		used += config.Used[i]

		cap += config.Cap[i]
	}

	log.WithFields(log.Fields{
		"loads":                config.Load,
		"number of schedulers": *config.NumSchedulers,
		"expected consensus":   (used + load) / cap,
	}).Debug("expected ratio")

	ctl := controller.New()
	ctl.AsServer()

}
