package util

import (
	"example/dist_sched/config"
	"fmt"
	"os"
	"runtime/trace"

	log "github.com/sirupsen/logrus"
)

func StartTrace(me int, trial uint64, k uint64) {
	if *config.Trace != "" {
		s := fmt.Sprintf("trace/trace-s%d-t%d-r%d.out", me, trial, k)
		if err := os.MkdirAll("trace", os.ModePerm); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("error creating trace directory")
		}
		f, err := os.Create(s)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error creating trace file")
		}

		if err := trace.Start(f); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error starting trace")
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Fatal("failed to close trace file")
			}
		}()
	}
}

func StopTrace() {
	if *config.Trace != "" {
		trace.Stop()
	}
}
