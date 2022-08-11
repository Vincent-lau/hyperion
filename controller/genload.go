package controller

import (
	"example/dist_sched/config"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

func (ctl *Controller) randGen() {

	for i := range ctl.load {
		ctl.load[i] = float64(rand.Intn(config.MaxCap))
		ctl.used[i] = float64(rand.Intn(config.MaxCap + 1 - int(ctl.load[i])))
		ctl.cap[i] = float64(config.MaxCap)
	}

}

func (ctl *Controller) preComp() {

	ctl.load = []float64{6, 10, 9, 6, 7, 1, 1, 6, 8, 2, 2, 10, 10, 9, 2, 8, 3, 5, 1, 9, 8, 7, 3, 6, 10, 4, 4, 2, 2, 4, 6, 9, 9, 6, 5, 7, 2, 1, 5, 2, 10, 3, 2, 1, 5, 6, 7, 4, 6, 3, 9, 10, 4, 10, 1, 4, 4, 3, 6, 7, 6, 10, 4, 2, 7, 9, 9, 10, 8, 10, 1, 10, 10, 3, 8, 8, 3, 9, 3, 3, 5, 1, 9, 2, 8, 4, 4, 7, 10, 4, 3, 9, 8, 5, 9, 3, 2, 8, 1, 9}
	ctl.used = []float64{1, 0, 1, 2, 1, 3, 5, 4, 1, 5, 8, 0, 0, 1, 3, 1, 5, 1, 6, 0, 2, 1, 7, 3, 0, 3, 3, 1, 3, 5, 1, 1, 0, 2, 4, 2, 1, 5, 1, 7, 0, 0, 1, 7, 1, 3, 2, 2, 3, 5, 0, 0, 2, 0, 7, 6, 6, 6, 4, 2, 2, 0, 6, 1, 1, 1, 1, 0, 2, 0, 1, 0, 0, 2, 0, 1, 2, 0, 5, 1, 2, 4, 1, 5, 0, 4, 2, 3, 0, 2, 5, 1, 1, 4, 1, 6, 5, 2, 7, 1}
	for i := range ctl.load {
		ctl.cap[i] = float64(config.MaxCap)
	}
}

func (ctl *Controller) GenLoad() {
	ctl.randGen()
}

// generate jobs such that it mataches the generated workload
func (ctl *Controller) genJobs() []float64 {
	jobs := make([]float64, 0)

	rand.Shuffle(len(ctl.load), func(i, j int) {
		ctl.load[i], ctl.load[j] = ctl.load[j], ctl.load[i]
	})

	for _, l := range ctl.load {
		left := l
		for left > 1e-2 {
			n := float64(rand.Intn(int(left)) + 1)
			left -= n
			jobs = append(jobs, n)
		}
	}

	log.WithFields(log.Fields{
		"jobs": jobs,
	}).Debug("generated jobs")

	return jobs
}