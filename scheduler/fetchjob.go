package scheduler

import (
	pb "example/dist_sched/message"
	util "example/dist_sched/util"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) Placement() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	log.Debug("waiting for all schedulers to send finish before starting placement")

	for !sched.allDone {
		sched.startCond.Wait()
	}

	log.Debug("placement started")

	sched.computeW()
	sched.fetchJobs()

}

func (sched *Scheduler) fetchJobs() {
	// TODO use stream API?

	gotJobs := make([]float64, 0)
	sw := sched.w

	fail1 := false

	for sched.w >= 1e-2 {
		req := &pb.JobRequest{
			Size:  sched.w,
			Trial: int32(sched.trial),
		}
		sched.mu.Unlock()
		r, err := util.MakeRPC(req, sched.ctlPlStub.GetJob)
		sched.mu.Lock()

		if int(r.GetSize()) < 0 || err != nil {
			if !fail1 {
				log.WithFields(log.Fields{
					"error": err,
				}).Info("failed to get job the first time")
				fail1 = true
			} else {
				log.WithFields(log.Fields{
					"error": err,
				}).Info("failed to get jobs the second time")
				break
			}
		} else if r.GetSize() == 0 {
			log.WithFields(log.Fields{
				"size": r.GetSize(),
			}).Warn("getting job of size 0")
		} else {
			PlLogger.WithFields(log.Fields{
				"job": r.GetSize(),
			}).Debug("got a job")
			gotJobs = append(gotJobs, r.GetSize())
		}
		sched.w -= r.GetSize()
	}

	util.RetryRPC(&pb.JobRequest{
		Trial: int32(sched.trial),
		Size:  -1,
	}, sched.ctlPlStub.GetJob)

	PlLogger.WithFields(log.Fields{
		"got jobs":  gotJobs,
		"initial w": sw,
		"final w":   sched.w,
	}).Info("fetched jobs")

}

func (sched *Scheduler) computeW() {
	sched.ratio = sched.MyData().GetY() / sched.MyData().GetZ()
	sched.w = sched.ratio*sched.pi - sched.u

	log.WithFields(log.Fields{
		"ratio":          sched.ratio,
		"used":           sched.u,
		"total capacity": sched.pi,
		"w":              sched.w,
	}).Debug("computed w")

}
