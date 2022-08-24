package scheduler

import (
	"os"
	"runtime/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/health/grpc_health_v1"

	"example/dist_sched/config"
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
	inConns      []int
	inNeighbours int
	expectedIn   int // expected number of inNeighbours, used for waiting
	// number of out neighbours, N+, i.e. ones to which we will broadcast values
	outConns      []int
	outNeighbours int
	stubs         map[int]pb.RatioConsensusClient // stubs of outNeighbours

	k    int
	done bool // this indicates termination, which is different from flag

	conData map[int]map[int]*pb.ConData

	/* for job fetching */
	pi      float64 // capacity
	u       float64 // used
	ratio   float64 // (rho + u) / pi after consensus
	w       float64 // remaining capacity = z * pi - u
	allDone bool    // all schedulers have sent finish to controller

	/* used by central controller */
	me         int
	trial      int
	setup      bool // all scheduler connected and ready for consensus
	startCond  *sync.Cond
	ctlRegStub pb.SchedRegClient
	ctlPlStub  pb.JobPlacementClient

	/* metrics */
	msgRcv  uint64
	msgSent uint64

	/* implementing the gRPC server */
	pb.UnimplementedRatioConsensusServer
	pb.UnimplementedSchedStartServer
	grpc_health_v1.UnimplementedHealthServer
}

const (
	schedulerName = "my-scheduler"
)

var (
	MetricsLogger = log.WithFields(log.Fields{"prefix": "metrics"})
	PlLogger      = log.WithFields(log.Fields{"prefix": "placement"})
)

func New() *Scheduler {
	st := time.Now()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not get hostname: %v", err)
	}

	sched := &Scheduler{
		hostname:      hostname,
		inConns:       make([]int, 0),
		inNeighbours:  0,
		expectedIn:    0,
		stubs:         make(map[int]pb.RatioConsensusClient),
		outConns:      make([]int, 0),
		outNeighbours: 0,
		k:             0,
		conData:       make(map[int]map[int]*pb.ConData),

		setup: false,

		/* placement */
		allDone: false,

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
		"setup time": time.Since(st).Seconds(),
	}).Info("setup done")

	return sched
}

func (sched *Scheduler) reset() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	sched.k = 0
	sched.done = false

	sched.conData = make(map[int]map[int]*pb.ConData)
	sched.setup = false

	/* placement */
	sched.allDone = false

	sched.msgRcv = 0
	sched.msgSent = 0

}

func (sched *Scheduler) Schedule() {

	if *config.CpuProfile != "" {
		f, err := os.Create(*config.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *config.MemProfile != "" {
		f, err := os.Create(*config.MemProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}

	for {

		log.WithFields(log.Fields{
			"name":  sched.hostname,
			"trial": sched.trial,
		}).Info("new trial is starting")

		sched.Consensus()
		sched.Placement()
		sched.reset()

		sched.mu.Lock()
		for !sched.setup {
			sched.startCond.Wait()
		}
		sched.mu.Unlock()

	}

}
