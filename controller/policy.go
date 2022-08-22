package controller

import (
	pb "example/dist_sched/message"
	"math/rand"

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
			r = powerOf2(in, q)
			if r.GetSize() != -1 {
				return r, nil
			}
		} else if q.Len() == 1 {
			r = getOne(in, q)
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

		// We put elements that is not accepted to the tail of the queue, might not be
		// the best way
		/* power of 2 choices */
		if q.Len() >= 2 {
			r = powerOf2(in, q)
			if r.GetSize() != -1 {
				break
			}
		} else {
			r = getOne(in, q)
			if r.GetSize() != -1 {
				break
			}
		}
	}

	return r, nil

}

func powerOf2(in *pb.JobRequest, q *queue.Queue) *pb.JobReply {
	ps, err := q.Get(2)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to get job from queue")
	}

	if len(ps) < 2 {
		log.WithFields(log.Fields{
			"ps len": len(ps),
		}).Warn("not enough elements after checking, maybe went before us")
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
			"found job":      p,
			"other choice":   op,
			"requested size": in.GetSize(),
		}).Debug("power of two choices job fetched")

		return &pb.JobReply{Size: p}
	} else {
		return &pb.JobReply{Size: -1}
	}

}

func getOne(in *pb.JobRequest, q *queue.Queue) *pb.JobReply {

	ps, err := q.Get(1)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("cannot get item from queue")
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
