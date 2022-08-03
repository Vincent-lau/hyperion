package scheduler

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/health/grpc_health_v1"

	pb "example/dist_sched/message"
)

type Scheduler struct {
	mu sync.Mutex
	/* cond is used for the following purpose
	1. wait for all in neighbours to connect to me
	2. wait for all in neighbours to send their message to me
	*/
	neighCond *sync.Cond

	hostname string
	// name of inNeighbours, N-, ones that we will receive data from
	inConns      []string
	inNeighbours int
	expectedIn   int // expected number of inNeighbours, used for waiting
	// number of out neighbours, N+, i.e. ones to which we will broadcast values
	outConns      []string
	outNeighbours int
	stubs         map[string]pb.RatioConsensusClient // stubs of outNeighbours

	k    int
	done bool // this indicates termination, which is different from flag

	conData map[int]map[string]*pb.ConData

	/* used by central controller */
	me        int
	trial     int
	setup     bool
	startCond *sync.Cond
	ctlStub   pb.SchedRegClient
	pb.UnimplementedSchedStartServer

	/* metrics */
	msgRcv  int
	msgSent int

	/* implementing the gRPC server */
	pb.UnimplementedRatioConsensusServer
	grpc_health_v1.UnimplementedHealthServer
}

const (
	schedulerName = "my-scheduler"
)

var (
	MetricsLogger = log.WithFields(log.Fields{"prefix": "metrics"})
)

func New() *Scheduler {
	st := time.Now()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not get hostname: %v", err)
	}

	sched := &Scheduler{
		hostname:      hostname,
		inConns:       make([]string, 0),
		inNeighbours:  0,
		expectedIn:    0,
		stubs:         make(map[string]pb.RatioConsensusClient),
		outConns:      make([]string, 0),
		outNeighbours: 0,
		k:             0,
		conData:       make(map[int]map[string]*pb.ConData),

		setup: false,

		// metrics
		msgSent: 0,
		msgRcv:  0,
	}
	sched.neighCond = sync.NewCond(&sched.mu)
	sched.startCond = sync.NewCond(&sched.mu)

	sched.AsServer()
	time.Sleep(time.Second)
	sched.AsClient()

	MetricsLogger.WithFields(log.Fields{
		"test":       "whatever",
		"setup time": time.Since(st).Seconds(),
	}).Info("setup done")

	return sched

}
