package controller

import (
	"math"
	"math/rand"

	config "github.com/Vincent-lau/hyperion/internal/configs"

	log "github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/stat"
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
	if config.K8sPlace {
		ctl.getJobsFromK8s(numJobs, avail)
	} else {
		ctl.genJobs(config.Distribution, numJobs, avail)
	}
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

	ctl.dmdMean = stat.Mean(ctl.jobDemand, nil)
	ctl.dmdStd = stat.StdDev(ctl.jobDemand, nil)

	log.WithFields(log.Fields{
		"jobDemand": ctl.jobDemand,
		"mean":      ctl.dmdMean,
		"std":       ctl.dmdStd,
	}).Debug("got jobs from actual pod queue")
}

// generate synthetic jobs, only with numerical values
func (ctl *Controller) genJobs(distribution config.Distr, numJobs int, avail float64) {

	log.WithFields(log.Fields{
		"avail": avail,
	}).Debug("available capacity")

	if config.DistrStatic {
		ctl.dmdMean = config.Mean
		ctl.dmdStd = config.Std
	} else {
		// set the mean and std of the job demand based on the available capacity and
		// the total number of jobs requested, so that our total demand is less than
		// the available capacity
		// TODO tune 0.6 and 0.4
		if distribution == config.Normal {
			ctl.dmdMean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.6
			ctl.dmdStd = ctl.dmdMean * 0.4
		} else if distribution == config.Poisson {
			ctl.dmdMean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.3
			ctl.dmdStd = math.Sqrt(config.Mean)
		} else if distribution == config.SkewNormal {
			ctl.dmdMean = avail / math.Max(float64(numJobs), float64(*config.NumSchedulers)) * 0.6
			ctl.dmdStd = config.Mean * 0.4
		}
	}

	PlLogger.WithFields(log.Fields{
		"configured job mean": config.Mean,
		"configured job std":  config.Std,
		"number of jobs":      numJobs,
	}).Info("job mean and std for random generation")

	generated := 0.0
	c := 0

	poi := distuv.Poisson{Lambda: ctl.dmdMean}

	for generated+config.Std < avail && c < numJobs {
		var v float64
		if distribution == config.Normal {
			v = rand.NormFloat64()*float64(ctl.dmdStd) + float64(ctl.dmdMean)
		} else if distribution == config.Uniform {
			v = rand.Float64() * float64(config.MaxCap)
		} else if distribution == config.Poisson {
			v = poi.Rand()
		} else if distribution == config.SkewNormal {
			v = skewNorm(config.Skew)*float64(ctl.dmdStd) + float64(ctl.dmdMean)
		} else {
			panic("unknown distribution")
		}

		v = math.Round(v)
		if v <= 0 || v >= config.MaxCap {
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
