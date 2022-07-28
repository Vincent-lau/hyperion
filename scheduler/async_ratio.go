/*
	Specific data structure for the asynchronous ratio consensus algorithm
*/

package scheduler

import (
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"math"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) InitMyConData() {

	sched.mu.Lock()
	defer sched.mu.Unlock()

	if _, ok := sched.conData[0]; !ok {
		sched.conData[0] = make(map[string]*pb.ConData)
	}

	// this node's data
	sched.conData[0][sched.hostname] = &pb.ConData{
		P:    1 / (float64(sched.outNeighbours) + 1), // p for this node
		Y:    config.Load[sched.me] + config.Used[sched.me],
		Z:    config.Cap[sched.me],
		Mm:   math.Inf(1),
		M:    math.Inf(-1),
		Flag: false,
	}

	log.WithFields(log.Fields{
		"data": sched.MyData(),
	}).Debug("initialized conData")

}
