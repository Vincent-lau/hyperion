package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"example/dist_sched/util"
	"math/rand"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

func (ctl *Controller) Placement() {
	ctl.populateQueue()
	plStart = time.Now()

	go ctl.bcastPl()
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

		if atomic.AddInt32(&v, int32(-*config.NumSchedulers)) == 0 {
			ctl.randPlace()
			ctl.finPl()
		}
		return &pb.JobReply{}, nil
	}

	log.WithFields(log.Fields{
		"from":        in.GetNode(),
		"smallQueue":  ctl.jobQueue[2].Len(),
		"mediumQueue": ctl.jobQueue[1].Len(),
		"largeQueue":  ctl.jobQueue[0].Len(),
	}).Debug("got request, current queue status")

	j := ctl.Large2Small(in)

	ctl.tq.Put(time.Since(t).Microseconds())

	if j.Size() > 0 {
		go ctl.placePodToNode(in.GetNode(), j.Id())
	}

	// 	log.WithFields(log.Fields{
	// 		"requested size": in.GetSize(),
	// 		"smallQueue":     ctl.jobQueue[2],
	// 		"mediumQueue":    ctl.jobQueue[1],
	// 		"largeQueue":     ctl.jobQueue[0],
	// 	}).Debug("no job found, ask scheduler to stop")

	// TODO here we signal no more jobs when the head of the queue cannot satisfy

	return &pb.JobReply{Size: j.Size()}, nil

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
	for i, j := range ctl.jobDemand {
		go func(id int, s float64) {
			i := getQueueIdx(s)
			if err := ctl.jobQueue[i].Put(&Job{id: id, size: s}); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("failed to put job into queue")
			}
			done <- 1
		}(i, j)
	}

	for range ctl.jobDemand {
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
	s := make(map[int]bool)

	t := time.Now()
	for _, q := range ctl.jobQueue {
		vs := q.Dispose()
		for _, v := range vs {
			elementsLeft = append(elementsLeft, v.(*Job).Size())
			s[v.(*Job).Id()] = true
		}
	}

	for i, j := range ctl.jobDemand {
		if _, ok := s[i]; ok && s[i] {
			s[i] = false
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

func (ctl *Controller) randPlace() {

	log.WithFields(log.Fields{
		"smallQueue":  ctl.jobQueue[2].Len(),
		"mediumQueue": ctl.jobQueue[1].Len(),
		"largeQueue":  ctl.jobQueue[0].Len(),
	}).Debug("random placement")

	ns := make([]string, 0)
	for k := range ctl.nodeMap {
		ns = append(ns, k)
	}

	log.WithFields(log.Fields{
		"nodes": ns,
	}).Debug("got nodes")

	for _, q := range ctl.jobQueue {
		vs := q.Dispose()
		for _, v := range vs {
			go ctl.placePodToNode(ns[rand.Intn(*config.NumSchedulers)], v.(*Job).Id())
		}
	}
}
