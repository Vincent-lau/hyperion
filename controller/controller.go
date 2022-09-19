package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/Workiva/go-datastructures/queue"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

var (
	PlLogger = log.WithFields(log.Fields{
		"prefix": "placement",
		"trial":  0,
	})
	plStart time.Time
)

type Controller struct {
	mu         sync.Mutex
	schedulers []string
	readySched map[int]bool      // schedulers that have finished connection
	schedIP    map[string]string // scheduler name -> IP addr
	schedStub  map[string]pb.SchedStartClient
	network    [][]int
	rnetwork   [][]int
	load       []float64
	used       []float64
	cap        []float64
	jobDemand  []float64

	finSched map[int]bool
	trial    uint64

	/* for placement */
	jobQueue []*queue.Queue
	fetched  int32
	placed   chan int

	/* interfacing with k8s api */
	jobPod    []*v1.Pod // mapping from job id to pod
	nodeMap   map[string]*v1.Node
	clientset *kubernetes.Clientset

	/* for profiling */
	tq *queue.Queue

	pb.UnimplementedSchedRegServer
	pb.UnimplementedJobPlacementServer
	grpc_health_v1.UnimplementedHealthServer
}

func New() *Controller {

	ctl := &Controller{
		schedulers: make([]string, 0),
		schedIP:    make(map[string]string),
		readySched: make(map[int]bool),
		schedStub:  make(map[string]pb.SchedStartClient),
		network:    config.Network,
		rnetwork:   config.RNetwork,
		load:       make([]float64, *config.NumSchedulers),
		used:       make([]float64, *config.NumSchedulers),
		cap:        make([]float64, *config.NumSchedulers),
		jobDemand:  make([]float64, 0),
		jobPod:     make([]*v1.Pod, 0),
		finSched:   make(map[int]bool),
		trial:      0,
		jobQueue:   make([]*queue.Queue, 3),
		nodeMap:    make(map[string]*v1.Node),
		fetched:    0,
		placed:     make(chan int),
		tq:         queue.New(0),
	}

	for i := range ctl.jobQueue {
		ctl.jobQueue[i] = queue.New(0)
	}

	config, err := rest.InClusterConfig()
	// ! essentailly turning off rate limiter
	// default value 5 and 10
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(80, 100)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error getting config")
	}
	clientset, err := kubernetes.NewForConfig(config)
	ctl.clientset = clientset

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error getting config")
	}

	ctl.findNodes()

	return ctl
}

func (ctl *Controller) connSched(in *pb.RegRequest) {
	for {
		conn, err := grpc.Dial(in.GetIp()+":"+*config.StartPort,
			grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"name":  in.GetName(),
			}).Warn("Could not connect to scheduler")
			ctl.mu.Unlock()
			time.Sleep(time.Second)
			ctl.mu.Lock()
		} else {
			ctl.schedStub[in.GetName()] = pb.NewSchedStartClient(conn)
			break
		}
	}

}

func (ctl *Controller) Reg(ctx context.Context, in *pb.RegRequest) (*pb.RegReply, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	log.WithFields(log.Fields{
		"requester data": in,
	}).Debug("Received reg request")

	if !slices.Contains(ctl.schedulers, in.GetName()) {
		ctl.schedulers = append(ctl.schedulers, in.GetName())
		ctl.schedIP[in.GetName()] = in.GetIp()

		ctl.connSched(in)

		return &pb.RegReply{
			You: int32(len(ctl.schedulers) - 1),
		}, nil
	} else {
		log.WithFields(log.Fields{
			"name": in.GetName(),
		}).Debug("scheduler already registered")

		return &pb.RegReply{
			You: int32(slices.Index(ctl.schedulers, in.GetName())),
		}, nil
	}

}

func (ctl *Controller) GetNeighbours(ctx context.Context, in *pb.NeighboursRequest) (*pb.NeighboursReply, error) {

	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if len(ctl.schedulers) < *config.NumSchedulers {
		log.WithFields(log.Fields{
			"expecting": *config.NumSchedulers,
			"got":       len(ctl.schedulers),
		}).Debug("not enough schedulers")

		return nil, errors.New("not enough schedulers, wait more")
	}

	neighbours := make([]string, 0)
	for _, s := range ctl.network[in.GetMe()] {
		neighbours = append(neighbours, ctl.schedIP[ctl.schedulers[s]])
	}

	return &pb.NeighboursReply{
		Neigh:        neighbours,
		InNeighbours: uint64(len(ctl.rnetwork[in.GetMe()])),
	}, nil

}

func (ctl *Controller) FinSetup(ctx context.Context, in *pb.SetupRequest) (*pb.SetupReply, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	/*  we check:
	1. all schedulers are ready
	2. the scheduler has been connected by all its in neighbours
	*/
	allConnected := int(in.GetInNeighbours()) == len(config.RNetwork[int(in.GetMe())])

	if allConnected {
		ctl.readySched[int(in.GetMe())] = true
	} else {
		log.WithFields(log.Fields{
			"connected neighbours": int(in.GetInNeighbours()),
			"expected neighbours":  len(config.RNetwork[int(in.GetMe())]),
		}).Debug("you are not all connected")
	}

	log.WithFields(log.Fields{
		"scheduler ready": in.GetMe(),
	}).Debug("scheduler ready")

	allReady := len(ctl.readySched) == *config.NumSchedulers

	log.WithFields(log.Fields{
		"controller reply": allReady,
		"to":               in.GetMe(),
		"in neighbours":    in.GetInNeighbours(),
	}).Debug("controller reply")

	if allReady {
		go ctl.newTrial()
	}

	return &pb.SetupReply{
		Finished: allReady,
	}, nil
}

func (ctl *Controller) FinConsensus(ctx context.Context, in *pb.FinRequest) (*pb.EmptyReply, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if in.GetTrial() != ctl.trial || len(ctl.finSched) == *config.NumSchedulers {
		log.WithFields(log.Fields{
			"from":        in.GetMe(),
			"sched trial": in.GetTrial(),
			"ctl trial":   ctl.trial,
			"finished":    len(ctl.finSched) == *config.NumSchedulers,
		}).Debug("wrong trial or already finished")
		return &pb.EmptyReply{}, nil
	}

	ctl.finSched[int(in.GetMe())] = true

	if len(ctl.finSched) == *config.NumSchedulers {
		log.WithFields(log.Fields{
			"from": in.GetMe(),
		}).Debug("all schedulers finished")

		go ctl.Placement()
	}

	return &pb.EmptyReply{}, nil

}

/* --private functions-- */

func (ctl *Controller) reset() {
	ctl.fetched = 0
	ctl.jobDemand = make([]float64, 0)
	ctl.jobPod = make([]*v1.Pod, 0)
	ctl.finSched = make(map[int]bool)
	for i := range ctl.jobQueue {
		ctl.jobQueue[i] = queue.New(0)
	}
	ctl.tq = queue.New(0)

}

func (ctl *Controller) newTrial() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if ctl.trial >= *config.MaxTrials {
		log.WithFields(log.Fields{
			"trial": ctl.trial,
		}).Info("max trials reached")
		return
	}

	ctl.trial++
	PlLogger = PlLogger.WithFields(log.Fields{
		"trial": ctl.trial,
	})

	ctl.reset()
	ctl.GenParam()

	log.WithFields(log.Fields{
		"load":  ctl.load,
		"used":  ctl.used,
		"cap":   ctl.cap,
		"trial": ctl.trial,
	}).Info("new trial")

	var load, used, cap float64

	for i := range ctl.load {
		load += ctl.load[i]
		used += ctl.used[i]
		cap += ctl.cap[i]
	}

	log.WithFields(log.Fields{
		"number of schedulers": *config.NumSchedulers,
		"expected consensus":   (used + load) / cap,
	}).Debug("expected ratio")

	ctl.bcastStart()
}

func (ctl *Controller) bcastStart() {
	log.Debug("broadcasting start")

	for i, s := range ctl.schedulers {
		go func(i int, s string, trial uint64, l, u, pi float64) {
			_, err := ctl.schedStub[s].StartConsensus(context.Background(),
				&pb.StartRequest{
					Trial: trial,
					L:     l,
					U:     u,
					Pi:    pi,
				})
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warn("could not broadcast start")
			}
		}(i, s, ctl.trial, ctl.load[i], ctl.used[i], ctl.cap[i])
	}
}
