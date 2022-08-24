package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"example/dist_sched/util"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

func (ctl *Controller) Placement() {
	ctl.populateQueue()
	plStart = time.Now()

	go ctl.bcastPl()
}

func (ctl *Controller) bcastPl() {
	log.Debug("broadcasting placement start")

	for _, s := range ctl.schedulers {
		go func(s string) {
			util.RetryRPC(&pb.EmptyRequest{}, ctl.schedStub[s].StartPlace)
		}(s)
	}
}

func getQueueIdx(size float64) int {
	small := config.Mean - config.Std  // x < mu - std
	medium := config.Mean + config.Std // mu-std < x < mu + std

	if size >= medium {
		return 0
	} else if size >= small && size < medium {
		return 1
	} else if size < small {
		return 2
	} else {
		panic("invalid size")
	}
}

// put jobs into multiple queues
func (ctl *Controller) populateQueue() {
	done := make(chan int)
	for _, j := range ctl.jobs {
		go func(s float64) {
			i := getQueueIdx(s)
			if err := ctl.jobQueue[i].Put(s); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("failed to put job into queue")
			}
			done <- 1
		}(j)
	}

	for range ctl.jobs {
		<-done
	}

	PlLogger.WithFields(log.Fields{
		"smallQueue":  ctl.jobQueue[2],
		"mediumQueue": ctl.jobQueue[1],
		"largeQueue":  ctl.jobQueue[0],
	}).Debug("populated queue")

}

func (ctl *Controller) finPl() {
	PlLogger.WithFields(log.Fields{
		"time taken": time.Since(plStart).Microseconds(),
	}).Info("placement time")

	elementsLeft := make([]float64, 0)
	jobSched := make([]float64, 0)
	s := make(map[float64]int)

	t := time.Now()
	for _, q := range ctl.jobQueue {
		vs := q.Dispose()
		for _, v := range vs {
			elementsLeft = append(elementsLeft, v.(float64))
			if _, ok := s[v.(float64)]; !ok {
				s[v.(float64)] = 1
			} else {
				s[v.(float64)]++
			}
		}
	}

	for _, j := range ctl.jobs {
		if _, ok := s[j]; ok && s[j] > 0 {
			s[j]--
		} else {
			jobSched = append(jobSched, j)
		}
	}

	PlLogger.WithFields(log.Fields{
		"left elements":       elementsLeft,
		"scheduled elements":  jobSched,
		"each job fetch time": ctl.tq.Dispose(),
		"drain time":          time.Since(t).Microseconds(),
	}).Info("all jobs fetched, queue elements left")

	go ctl.newTrial()

}

func (ctl *Controller) GetJob(ctx context.Context, in *pb.JobRequest) (*pb.JobReply, error) {
	if ctl.trial != int(in.GetTrial()) {
		log.WithFields(log.Fields{
			"sched trial": in.GetTrial(),
			"ctl trial":   ctl.trial,
		}).Debug("wrong trial")
		return nil, errors.New("wrong trial")
	}
	t := time.Now()

	if in.GetSize() < 0 {
		v := atomic.AddInt32(&ctl.fetched, 1)

		if atomic.AddInt32(&v, int32(-*config.NumSchedulers)) == 0{
			ctl.finPl()
		}
		return &pb.JobReply{}, nil
	}

	log.WithFields(log.Fields{
		"smallQueue":  ctl.jobQueue[2].Len(),
		"mediumQueue": ctl.jobQueue[1].Len(),
		"largeQueue":  ctl.jobQueue[0].Len(),
	}).Debug("got request, current queue status")

	r, _ := ctl.Large2Small(in)

	ctl.tq.Put(time.Since(t).Microseconds())

	// TODO here we signal no more jobs when the head of the queue cannot satisfy
	// if r.GetSize() < 0 {
	// 	// the requirement, it might not be the case
	// 	log.WithFields(log.Fields{
	// 		"requested size": in.GetSize(),
	// 		"smallQueue":     ctl.jobQueue[2],
	// 		"mediumQueue":    ctl.jobQueue[1],
	// 		"largeQueue":     ctl.jobQueue[0],
	// 	}).Debug("no job found, ask scheduler to stop")
	// }

	return r, nil

}
