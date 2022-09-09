package controller

import (
	"example/dist_sched/config"
	"math"
	"math/rand"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/stat/distuv"
	v1 "k8s.io/api/core/v1"
)

func (ctl *Controller) GenParam() {
	ctl.genUsed()
	ctl.genLoad()
}

func (ctl *Controller) loadFromJobs() {
	if len(ctl.jobDemand) < *config.NumSchedulers {
		splitSz := *config.NumSchedulers / len(ctl.jobDemand)
		if *config.NumSchedulers%len(ctl.jobDemand) != 0 {
			splitSz++
		}

		i := 0
		k := 0
		for i < len(ctl.jobDemand) {
			n := ctl.jobDemand[i]
			for j := 0; j < splitSz; j++ {
				if k >= len(ctl.load) || n <= 0 {
					ctl.load[k-1] += n
					i++
					for i < len(ctl.jobDemand) {
						ctl.load[k-1] += ctl.jobDemand[i]
						i++
					}
					break
				}
				var m float64
				if j == splitSz-1 {
					m = n
				} else {
					m = rand.Float64() * n
				}

				ctl.load[k] = m
				k++
				n -= m
			}
			i++
		}

	} else {
		groupSz := int(len(ctl.jobDemand) / *config.NumSchedulers)
		j := 0
		for i := range ctl.load {
			l := 0.0
			for k := 0; k < groupSz; k++ {
				l += ctl.jobDemand[j]
				j++
			}
			ctl.load[i] = l
		}
		for j < len(ctl.jobDemand) {
			ctl.load[len(ctl.load)-1] += ctl.jobDemand[j]
		}

	}

	sj := 0.0
	sl := 0.0
	for i := range ctl.jobDemand {
		sj += ctl.jobDemand[i]
	}
	for i := range ctl.load {
		sl += ctl.load[i]
	}

	log.WithFields(log.Fields{
		"load":     ctl.load,
		"load sum": sl,
		"jobs sum": sj,
		"used":     ctl.used,
	}).Debug("fitted jobs into workload")

}

func (ctl *Controller) genLoad() {

	avail := 0.0
	for i := range ctl.used {
		avail += ctl.cap[i] - ctl.used[i]
	}

	numJobs := int(*config.JobFactor * float64(*config.NumSchedulers))
	// ctl.genJobs("normal", numJobs, avail)
	ctl.getJobsFromK8s(numJobs, avail)
	ctl.loadFromJobs()
}

func (ctl *Controller) genUsed() {
	for i := range ctl.load {
		ctl.used[i] = 0 // TODO change to non-zero val later on
		ctl.cap[i] = config.MaxCap
	}
}

// get pods from the k8s cluster, using k8s api
func (ctl *Controller) getJobsFromK8s(numJobs int, avail float64) {
	log.Debug("getting jobs from actual pod queue")
	podChan := make(chan *v1.Pod)

	go ctl.jobsFromPodQueue(podChan)

	t := 0.0

	for i := 0; i < numJobs; i++ {
		p := <-podChan
		d := getJobDemand(p)

		t += d
		if t >= avail {
			log.WithFields(log.Fields{
				"total demand": t,
				"avail":        avail,
			}).Debug("avail exceeded")
			break
		}

		ctl.jobDemand = append(ctl.jobDemand, d)
		ctl.jobPod = append(ctl.jobPod, p)
	}

	log.WithFields(log.Fields{
		"jobDemand": ctl.jobDemand,
	}).Debug("got jobs from actual pod queue")
}

// generate synthetic jobs, only with numerical values
func (ctl *Controller) genJobs(distribution string, numJobs int, avail float64) {

	log.WithFields(log.Fields{
		"avail": avail,
	}).Debug("available capacity")

	// TODO tune 0.6 and 0.4
	var poi distuv.Poisson
	if distribution == "normal" {
		config.Mean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.6
		config.Std = config.Mean * 0.4
	} else if distribution == "poisson" {
		config.Mean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.3
		config.Std = math.Sqrt(config.Mean)
		poi = distuv.Poisson{Lambda: config.Mean}
	} else if distribution == "skew normal" {
		config.Mean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.6
		config.Std = config.Mean * 0.4
		config.Skew = -4.0 // TODO tune this
	}

	PlLogger.WithFields(log.Fields{
		"mean":           config.Mean,
		"std":            config.Std,
		"number of jobs": numJobs,
	}).Info("job mean and std")

	generated := 0.0
	c := 0
	for generated+config.Std < avail && c < numJobs {
		var v float64
		if distribution == "normal" {
			v = rand.NormFloat64()*float64(config.Std) + float64(config.Mean)
		} else if distribution == "uniform" {
			v = rand.Float64() * float64(config.MaxCap)
		} else if distribution == "poisson" {
			v = poi.Rand()
		} else if distribution == "skew normal" {
			v = skewNorm(config.Skew)*float64(config.Std) + float64(config.Mean)
		} else {
			panic("unknown distribution")
		}

		if v < 0 || v >= config.MaxCap {
			log.WithFields(log.Fields{"v": v}).Debug("generated value out of bounds")
			continue
		}
		ctl.jobDemand = append(ctl.jobDemand, v)
		generated += v
		c++
	}

	PlLogger.WithFields(log.Fields{
		"generated jobDemand":  ctl.jobDemand,
		"number of jobDemand":  len(ctl.jobDemand),
		"total size":           generated,
		"number of schedulers": *config.NumSchedulers,
		"distribution":         distribution,
	}).Info("generated jobs")

}

func skewNorm(a float64) float64 {

	x1 := rand.NormFloat64()
	x2 := rand.NormFloat64()

	x3 := (a*math.Abs(x1) + x2) / math.Sqrt(1+a*a)

	return x3

}

// func (ctl *Controller) randGen() {

// 	for i := range ctl.load {

// 		ctl.load[i] = float64(rand.Intn(int(config.MaxCap)))

// 		ctl.used[i] = float64(rand.Intn(int(config.MaxCap + 1.0 - ctl.load[i])))
// 		ctl.cap[i] = float64(config.MaxCap)
// 	}

// }

// func (ctl *Controller) preComp() {

// 	ctl.load = []float64{6, 10, 9, 6, 7, 1, 1, 6, 8, 2, 2, 10, 10, 9, 2, 8, 3, 5, 1, 9, 8, 7, 3, 6, 10, 4, 4, 2, 2, 4, 6, 9, 9, 6, 5, 7, 2, 1, 5, 2, 10, 3, 2, 1, 5, 6, 7, 4, 6, 3, 9, 10, 4, 10, 1, 4, 4, 3, 6, 7, 6, 10, 4, 2, 7, 9, 9, 10, 8, 10, 1, 10, 10, 3, 8, 8, 3, 9, 3, 3, 5, 1, 9, 2, 8, 4, 4, 7, 10, 4, 3, 9, 8, 5, 9, 3, 2, 8, 1, 9}
// 	ctl.used = []float64{1, 0, 1, 2, 1, 3, 5, 4, 1, 5, 8, 0, 0, 1, 3, 1, 5, 1, 6, 0, 2, 1, 7, 3, 0, 3, 3, 1, 3, 5, 1, 1, 0, 2, 4, 2, 1, 5, 1, 7, 0, 0, 1, 7, 1, 3, 2, 2, 3, 5, 0, 0, 2, 0, 7, 6, 6, 6, 4, 2, 2, 0, 6, 1, 1, 1, 1, 0, 2, 0, 1, 0, 0, 2, 0, 1, 2, 0, 5, 1, 2, 4, 1, 5, 0, 4, 2, 3, 0, 2, 5, 1, 1, 4, 1, 6, 5, 2, 7, 1}
// 	for i := range ctl.load {
// 		ctl.cap[i] = float64(config.MaxCap)
// 	}
// }
