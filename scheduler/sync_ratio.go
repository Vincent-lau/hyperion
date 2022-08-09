/*
	Specific data structure for the asynchronous ratio consensus algorithm
*/

package scheduler

import (
	pb "example/dist_sched/message"
	"math"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) InitMyConData(l, u, pi float64) {

	if _, ok := sched.conData[0]; !ok {
		sched.conData[0] = make(map[int]*pb.ConData)
	}

	// this node's data
	sched.conData[0][sched.me] = &pb.ConData{
		P:    1 / (float64(sched.outNeighbours) + 1), // p for this node
		Y:    l + u,
		Z:    pi,
		Mm:   math.Inf(1),
		M:    math.Inf(-1),
		Flag: false,
	}

	log.WithFields(log.Fields{
		"data": sched.MyData(),
	}).Debug("initialized conData")

}
