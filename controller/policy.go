package controller

import (
	pb "example/dist_sched/message"
	"math/rand"
	"time"

	"github.com/Workiva/go-datastructures/queue"
	log "github.com/sirupsen/logrus"
)

// this file implements different policies for job assignment

func (ctl *Controller) RandomInAvail(in *pb.JobRequest) (*pb.JobReply, error) {

	k := getQueueIdx(in.GetSize())

	// we only randomly choose from queues that are not empty
	indices := make([]int, 0)
	for i := k; i < len(ctl.jobQueue); i++ {
		if !ctl.jobQueue[i].Empty() {
			indices = append(indices, i)
		}
	}

	r := &pb.JobReply{Size: -1}
	if len(indices) == 0 {
		return r, nil
	}

	rk := indices[rand.Intn(len(indices))]

	log.WithFields(log.Fields{
		"calculated index": k,
		"randomised index": rk,
	}).Debug("pick a random queue from all available")

	for ; rk < len(ctl.jobQueue); rk++ {
		q := ctl.jobQueue[rk]
		if q.Len() >= 2 {
			r = choose2(in, q)
			if r.GetSize() != -1 {
				return r, nil
			}
		} else if q.Len() == 1 {
			r = choose1(in, q)
			if r.GetSize() != -1 {
				return r, nil
			}
		}
	}

	return r, nil

}

/*
	We look at queues from large to small, and pick the first one that is available
	We also look at the first two elements of the queue when possilbe
*/
func (ctl *Controller) Large2Small(in *pb.JobRequest) (*pb.JobReply, error) {

	k := getQueueIdx(in.GetSize())
	r := &pb.JobReply{Size: -1}

	for i, q := range ctl.jobQueue {
		if q.Empty() || i < k {
			// queues are ordered from large to small
			// so if the desired index is larger, we want smaller elements than the current queue
			// log.WithFields(log.Fields{
			// 	"queue index":    i,
			// 	"desired index":  k,
			// 	"requested size": in.GetSize(),
			// 	"is empty":       q.Empty(),
			// }).Debug("wrong queue")
			continue
		}
		//  else {
		// 	log.WithFields(log.Fields{
		// 		"queue index":    i,
		// 		"requested size": in.GetSize(),
		// 	}).Debug("right queue")
		// }

		// checking the length does not provide any guarantee, we just use this to speed
		// things up
		if q.Len() >= 2 {
			r = choose2(in, q)
			if r.GetSize() != -1 {
				break
			}
		} else if q.Len() >= 1 {
			r = choose1(in, q)
			if r.GetSize() != -1 {
				break
			}
		}

	}

	return r, nil

}

func choose2(in *pb.JobRequest, q *queue.Queue) *pb.JobReply {
	ps, err := q.Poll(2, time.Millisecond)
	if err != nil || len(ps) < 2 {
		log.WithFields(log.Fields{
			"ps len": len(ps),
			"error":  err,
		}).Debug("not enough elements or queue disposed or timeout")

		if len(ps) > 0 {
			if err := q.Put(ps[0]); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warn("cannot put back elements")
			}
		}
		return &pb.JobReply{Size: -1}
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

	// ! We put elements that is not accepted to the tail of the queue, might not be appropriate
	if pf[li] <= in.GetSize() {
		p = pf[li]
		op = pf[si]
		q.Put(pf[si])
	} else if pf[si] <= in.GetSize() {
		p = pf[si]
		op = pf[li]
		q.Put(pf[li])
	} else {
		if err := q.Put(pf[0], pf[1]); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to put job into queue")
		}
	}

	if p > 0 {
		log.WithFields(log.Fields{
			"found job":      p,
			"other choice":   op,
			"requested size": in.GetSize(),
		}).Debug("two choices job fetched")

		return &pb.JobReply{Size: p}
	} else {
		return &pb.JobReply{Size: -1}
	}

}

func choose1(in *pb.JobRequest, q *queue.Queue) *pb.JobReply {

	ps, err := q.Poll(1, time.Millisecond)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("cannot get even one item from queue")
		return &pb.JobReply{Size: -1}
	}

	p := ps[0].(float64)

	if p <= in.GetSize() {
		log.WithFields(log.Fields{
			"found job":      p,
			"requested size": in.GetSize(),
		}).Debug("one choice job fetched")

		return &pb.JobReply{
			Size: p,
		}
	} else {
		log.WithFields(log.Fields{
			"size":               in.GetSize(),
			"queue element size": p,
		}).Debug("requested size too small, putting head of queue to tail")
		q.Put(p)
	}

	return &pb.JobReply{Size: -1}

}
