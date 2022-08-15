package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"example/dist_sched/util"
	"time"

	log "github.com/sirupsen/logrus"
)



func (ctl *Controller) Placement() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	ctl.populateQueue()
	plStart = time.Now()

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
	small := float64(config.MaxCap) * 0.3
	medium := float64(config.MaxCap) * 0.7
	large := float64(config.MaxCap) * 1

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


	if in.GetSize() < 0 {
		ctl.mu.Lock()
		defer ctl.mu.Unlock()

		ctl.fetched++
		if ctl.fetched == *config.NumSchedulers {
			elementsLeft := make([]float64, 0)
			for _, q := range ctl.jobQueue {
				for !q.Empty() {
					v, err := q.Get(1)
					if err != nil {
						log.WithFields(log.Fields{
							"error": err,
						}).Error("failed to get job from queue")
					}
					elementsLeft = append(elementsLeft, v[0].(float64))
				}
			}
			PlLogger.WithFields(log.Fields{
				"prefix":        "placement",
				"trial":         ctl.trial,
				"left elements": elementsLeft,
				"time taken": 	time.Since(plStart).Microseconds(),
			}).Info("all jobs fetched, queue elements left")

			go ctl.newTrial()
		}


		return &pb.JobReply{}, nil
	}

	log.WithFields(log.Fields{
		"size":        in.GetSize(),
		"smallQueue":  ctl.jobQueue[2].Len(),
		"mediumQueue": ctl.jobQueue[1].Len(),
		"largeQueue":  ctl.jobQueue[0].Len(),
	}).Debug("got request")

	for i, q := range ctl.jobQueue {
		k := getQueueIdx(in.GetSize())
		if q.Empty() || i < k {
			// queues are ordered from large to small
			// so if the desired index is larger, we want smaller elements than the current queue
			continue
		}

		// We put elements that is not accepted to the tail of the queue, might not be
		// the best way
		/* power of 2 choices */
		if q.Len() >= 2 {
			ps, err := q.Get(2)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Fatal("failed to get job from queue")
			}
			pf := make([]float64, 2)
			pf[0] = ps[0].(float64)
			pf[1] = ps[1].(float64)
			var li int
			p := -1.0
			op := -1.0

			if pf[0] > pf[1] {
				li = 0
			} else {
				li = 1
			}
			si := 1 - li

			if pf[li] <= in.GetSize() {
				p = pf[li]
				op = pf[si]
				q.Put(pf[si])
			} else if pf[si] <= in.GetSize() {
				p = pf[si]
				op = pf[li]
				q.Put(pf[li])
			} else {
				err := q.Put(pf[0], pf[1])
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("failed to put job into queue")
				}
			}

			if p > 0 {
				log.WithFields(log.Fields{
					"prefix":       "placement",
					"found job":    p,
					"other choice": op,
				}).Debug("power of two choices job fetched")
				return &pb.JobReply{Size: p}, nil
			}

		} else {

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

	}
	// TODO here we signal no more jobs when the head of the queue cannot satisfy
	// the requirement, it might not be the case
	return &pb.JobReply{
		Size: -1,
	}, nil
}
