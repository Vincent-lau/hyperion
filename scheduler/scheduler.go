package scheduler

import (
	"os"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/health/grpc_health_v1"

	"example/dist_sched/config"
	"example/dist_sched/controller"
	pb "example/dist_sched/message"
)

type Scheduler struct {
	mu sync.Mutex
	/* cond is used for the following purpose
	1. wait for all in neighbours to connect to me
	2. wait for all in neighbours to send their message to me
	*/
	neighCond *sync.Cond

	xchgChan chan uint64

	hostname string
	// name of inNeighbours, N-, ones that we will receive data from
	inConns      []int
	inNeighbours uint64
	expectedIn   uint64 // expected number of inNeighbours, used for waiting
	// number of out neighbours, N+, i.e. ones to which we will broadcast values
	outConns      []int
	outNeighbours uint64
	stubs         map[int]pb.RatioConsensusClient // stubs of outNeighbours
	streams       map[int]pb.RatioConsensus_SendConDataClient

	k    uint64
	done *atomic.Bool // this indicates termination, which is different from flag

	conData sync.Map // iter -> scheduler no -> conData
	conLen  sync.Map // iter -> number of received msg, including self

	/* for job fetching */
	pi      float64      // capacity
	u       float64      // used
	ratio   float64      // (rho + u) / pi after consensus
	w       float64      // remaining capacity = z * pi - u
	allDone *atomic.Bool // all schedulers have sent finish to controller

	/* used by central controller */
	me         uint64
	trial      uint64
	setup      *atomic.Bool // all scheduler connected and ready for consensus
	startCond  *sync.Cond
	ctlRegStub pb.SchedRegClient
	ctlPlStub  pb.JobPlacementClient

	/* metrics */
	msgRcv  uint64
	msgSent uint64

	/* interface with k8s */
	onNode string

	/* implementing the gRPC server */
	pb.UnimplementedRatioConsensusServer
	pb.UnimplementedSchedStartServer
	grpc_health_v1.UnimplementedHealthServer
}

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
	node, err := controller.MyNode(hostname)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("could not get node")
	}

	log.WithFields(log.Fields{
		"node": node.Name,
	}).Debug("found my node")

	sched := &Scheduler{
		hostname:      hostname,
		inConns:       make([]int, 0),
		inNeighbours:  0,
		expectedIn:    0,
		stubs:         make(map[int]pb.RatioConsensusClient),
		streams:       make(map[int]pb.RatioConsensus_SendConDataClient),
		outConns:      make([]int, 0),
		outNeighbours: 0,
		k:             0,
		done:          &atomic.Bool{},
		conData:       sync.Map{},
		conLen:        sync.Map{},

		setup: &atomic.Bool{},

		/* placement */
		allDone: &atomic.Bool{},

		// metrics
		msgSent: 0,
		msgRcv:  0,

		/* interface with k8s */
		onNode: node.Name,
	}
	sched.done.Store(false)
	sched.setup.Store(false)
	sched.allDone.Store(false)

	// ! 30 is arbitrary, we should use a size that is as large as the number of iterations
	// used for now
	sched.xchgChan = make(chan uint64, 30)
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

	atomic.StoreUint64(&sched.k, 0)
	sched.done.Store(false)

	sched.conData.Range(func(key, value any) bool {
		sched.conData.Delete(key)
		return true
	})
	sched.conLen.Range(func(key, value any) bool {
		sched.conLen.Delete(key)
		return true
	})

	sched.setup.Store(false)

	/* placement */
	sched.allDone.Store(false)

	atomic.StoreUint64(&sched.msgRcv, 0)
	atomic.StoreUint64(&sched.msgSent, 0)

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
			"trial": atomic.LoadUint64(&sched.trial),
		}).Info("new trial is starting")

		sched.Consensus()
		sched.Placement()
		sched.reset()

		for !sched.setup.Load() {
			sched.mu.Lock()
			sched.startCond.Wait()
			sched.mu.Unlock()
		}

	}

}
