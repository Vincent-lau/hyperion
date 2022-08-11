package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"example/dist_sched/util"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

const (
	small  = 3  // [0, 3)
	medium = 6  // [3, 6)
	large  = 11 // [6, 11)
)

func (ctl *Controller) Placement() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	ctl.populateQueue()
	go ctl.bcastPl()
}

func (ctl *Controller) bcastPl() {
	log.Debug("broadcasting placement start")

	for _, s := range ctl.schedulers {
		go func(s string) {
			util.MakeRPC(&pb.EmptyRequest{}, ctl.schedStub[s].StartPlace)
		}(s)
	}
}

func getQueueIdx(size float64) int {
	if size >= medium && size < large {
		return 0
	} else if size >= small && size < medium {
		return 1
	} else if size < small {
		return 2
	} else {
		panic("invalid size")
	}
}

// generate jobs such that it mataches the generated workload
func (ctl *Controller) genJobs() []float64 {
	// TODO for now just use the workload, might want split the workload a bit in the future
	rand.Shuffle(len(ctl.load), func(i, j int) {
		ctl.load[i], ctl.load[j] = ctl.load[j], ctl.load[i]
	})

	log.WithFields(log.Fields{
		"jobs": ctl.load,
	}).Debug("generated jobs")

	return ctl.load
}

// put jobs into multiple queues
func (ctl *Controller) populateQueue() {
	jobs := ctl.genJobs()
	done := make(chan int)
	for _, j := range jobs {
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

	for range jobs {
		<-done
	}

	log.WithFields(log.Fields{
		"smallQueue":  ctl.jobQueue[2],
		"mediumQueue": ctl.jobQueue[1],
		"largeQueue":  ctl.jobQueue[0],
	}).Debug("populated queue")

}

func (ctl *Controller) GetJob(ctx context.Context, in *pb.JobRequest) (*pb.JobReply, error) {
	if ctl.trial != int(in.GetTrial()) {
		log.WithFields(log.Fields{
			"sched trial": in.GetTrial(),
			"ctl trial":   ctl.trial,
		}).Debug("wrong trial")
		return nil, errors.New("wrong trial")
	}

	// TODO remove lock when we don't look at the queue elements
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if in.GetSize() < 0 {
		ctl.fetched++
		if ctl.fetched == *config.NumSchedulers {
			log.WithFields(log.Fields{
				"small queue":  ctl.jobQueue[2],
				"medium queue": ctl.jobQueue[1],
				"large queue":  ctl.jobQueue[0],
			}).Debug("all jobs fetched, queue elements left")
		}
		return &pb.JobReply{}, nil
	}



	log.WithFields(log.Fields{
		"size": in.GetSize(),
		"smallQueue":  ctl.jobQueue[2],
		"mediumQueue": ctl.jobQueue[1],
		"largeQueue":  ctl.jobQueue[0],
	}).Debug("got request")

	for i, q := range ctl.jobQueue {
		k := getQueueIdx(in.GetSize())
		if q.Empty() || i < k {
			// queues are ordered from large to small
			// so if the desired index is larger, we want smaller elements than the current queue
			continue
		}

		// TODO power of 2 choices
		// if q.Len() >= 2 {
		// 	p1, err := q.Get()
		// 	p2, err := q.Get()

		// 	if p1 > p2 {
		// 		if p1.(float64)	 <= in.GetSize() {

		// 		}

		// 	} else {

		// 	}

		// } else {

		// }

		ps, err := q.Get(1)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("cannot get item from queue")
		}

		p := ps[0].(float64)

		if p <= in.GetSize() {

			log.WithFields(log.Fields{
				"requested size":       in.GetSize(),
				"actual returned size": p,
			}).Debug("found job")

			return &pb.JobReply{
				Size: p,
			}, nil
		} else {
			log.WithFields(log.Fields{
				"size":               in.GetSize(),
				"queue element size": p,
			}).Debug("requested size too small, putting head of queue to tail")
			q.Put(p)
		}

	}


	return &pb.JobReply{
		Size: -1,
	}, nil
}
