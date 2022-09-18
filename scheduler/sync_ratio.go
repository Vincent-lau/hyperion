/*
	Specific data structure for the asynchronous ratio consensus algorithm
*/

package scheduler

import (
	pb "example/dist_sched/message"
	"math"
	"sync"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) InitMyConData(l, u, pi float64) {

	// this node's data
	kData, _ := sched.conData.LoadOrStore(uint64(0), &sync.Map{})
	kData.(*sync.Map).Store(sched.me, &pb.ConData{
		P:    1 / (float64(sched.outNeighbours) + 1), // p for this node
		Y:    l + u,
		Z:    pi,
		Mm:   math.Inf(1),
		M:    math.Inf(-1),
		Flag: false,
	})
	c, _ := sched.conLen.LoadOrStore(uint64(0), uint64(0))
	sched.conLen.Store(uint64(0), c.(uint64)+1)

	sched.u = u
	sched.pi = pi

	log.WithFields(log.Fields{
		"data": sched.MyData(),
	}).Debug("initialized conData")

}
