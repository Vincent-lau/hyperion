package scheduler

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	pb "example/dist_sched/message"
)

type Scheduler struct {
	mu       sync.Mutex
	cond     *sync.Cond
	hostname string

	// name of inNeighbours, N-, ones that we will receive data from
	inConns      []string
	inNeighbours int
	// number of out neighbours, N+, i.e. ones to which we will broadcast values
	outConns      []string
	outNeighbours int
	stubs         map[string]pb.RatioConsensusClient // stubs of outNeighbours

	k int

	// TODO array of struct or struct of array
	conData map[int]map[string]*pb.ConData
	done    bool

	// used by central controller
	me       int
	ctlStub  pb.SchedRegClient

	// implementing the gRPC server
	pb.UnimplementedRatioConsensusServer
}

const (
	schedulerName = "my-scheduler"
)

func New() *Scheduler {

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Could not get hostname: %v", err)
	}

	sched := &Scheduler{
		hostname:      hostname,
		inConns:       make([]string, 0),
		inNeighbours:  0,
		stubs:         make(map[string]pb.RatioConsensusClient),
		outConns:      make([]string, 0),
		outNeighbours: 0,
		k:             0,

		conData: make(map[int]map[string]*pb.ConData),
		done:    false,
	}
	sched.cond = sync.NewCond(&sched.mu)

	sched.AsServer()
	time.Sleep(time.Second)
	sched.AsClient()

	sched.InitMyConData()

	return sched

}

func (sched *Scheduler) Done() bool {
	return sched.done
}
