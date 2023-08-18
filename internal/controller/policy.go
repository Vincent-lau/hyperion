package controller

import (
	"github.com/Vincent-lau/hyperion/internal/configs"
	pb "github.com/Vincent-lau/hyperion/internal/message"
	"math/rand"
	"time"

	"github.com/Workiva/go-datastructures/queue"
	log "github.com/sirupsen/logrus"
)

// This file implements different policies for job assignment

func (ctl *Controller) heuPlace(f config.HeuFunc, in *pb.JobRequest) *Job {
	if f == config.Random {
		return ctl.RandomInAvail(in)
	} else if f == config.Large2small {
		return ctl.Large2Small(in)
	} else {
		panic("unknown heuristic function")
	}
}

func (ctl *Controller) RandomInAvail(in *pb.JobRequest) *Job {

	k := getQueueIdx(in.GetSize(), ctl.dmdMean, ctl.dmdStd)

	// we only randomly choose from queues that are not empty
	indices := make([]int, 0)
	for i := k; i < len(ctl.jobQueue); i++ {
		if !ctl.jobQueue[i].Empty() {
			indices = append(indices, i)
		}
	}

	r := &Job{id: -1, size: -1}
	if len(indices) == 0 {
		return r
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
			if r.Size() != -1 {
				return r
			}
		} else if q.Len() == 1 {
			r = choose1(in, q)
			if r.Size() != -1 {
				return r
			}
		}
	}

	return r

}

/*
We look at queues from large to small, and pick the first one that is available
We also look at the first two elements of the queue when possilbe
*/
func (ctl *Controller) Large2Small(in *pb.JobRequest) *Job {

	k := getQueueIdx(in.GetSize(), ctl.dmdMean, ctl.dmdStd)
	r := &Job{id: -1, size: -1}

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

		// checking the length does not provide any guarantee, we just use this to speed
		// things up
		if q.Len() >= 2 {
			r = choose2(in, q)
			if r.Size() != -1 {
				break
			}
		} else if q.Len() >= 1 {
			r = choose1(in, q)
			if r.Size() != -1 {
				break
			}
		}

	}
	return r
}

func choose2(in *pb.JobRequest, q *queue.Queue) *Job {
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
		return &Job{size: -1, id: -1}
	}

	pj := make([]*Job, 2)
	pj[0] = ps[0].(*Job)
	pj[1] = ps[1].(*Job)
	var li int
	p := &Job{size: -1, id: -1}
	op := &Job{size: -1, id: -1}

	if pj[0].Size() > pj[1].Size() {
		li = 0
	} else {
		li = 1
	}
	si := 1 - li

	// ! We put elements that is not accepted to the tail of the queue, might not be appropriate
	if pj[li].Size() <= in.GetSize() {
		p = pj[li]
		op = pj[si]
		q.Put(pj[si])
	} else if pj[si].Size() <= in.GetSize() {
		p = pj[si]
		op = pj[li]
		q.Put(pj[li])
	} else {
		if err := q.Put(pj[0], pj[1]); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to put job into queue")
		}
	}

	if p.Size() > 0 {
		log.WithFields(log.Fields{
			"found job":      p,
			"other choice":   op,
			"requested size": in.GetSize(),
			"to":             in.GetNode(),
		}).Debug("two choices job fetched")

		return &Job{size: p.Size(), id: p.Id()}
	} else {
		return &Job{size: -1, id: -1}
	}

}

func choose1(in *pb.JobRequest, q *queue.Queue) *Job {

	ps, err := q.Poll(1, time.Millisecond)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Warn("cannot get even one item from queue")
		return &Job{size: -1, id: -1}
	}

	p := ps[0].(*Job)

	if p.Size() <= in.GetSize() {
		log.WithFields(log.Fields{
			"found job":      p,
			"requested size": in.GetSize(),
			"to":             in.GetNode(),
		}).Debug("one choice job fetched")

		return &Job{
			size: p.Size(),
			id:   p.Id(),
		}
	} else {
		log.WithFields(log.Fields{
			"size":               in.GetSize(),
			"queue element size": p,
		}).Debug("requested size too small, putting head of queue to tail")
		q.Put(p)
	}

	return &Job{size: -1, id: -1}

}
