package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
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

	finSched map[int]bool
	trial    int

	pb.UnimplementedSchedRegServer
	grpc_health_v1.UnimplementedHealthServer
}

func (ctl *Controller) AsServer() {
	ctl.healthSrv()

	lis, err := net.Listen("tcp", ":"+*config.CtlPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSchedRegServer(s, ctl)

	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve")
	}

}

func New() *Controller {

	return &Controller{
		schedulers: make([]string, 0),
		schedIP:    make(map[string]string),
		readySched: make(map[int]bool),
		schedStub:  make(map[string]pb.SchedStartClient),
		network:    config.Network,
		rnetwork:   config.RNetwork,
		load:       make([]float64, *config.NumSchedulers),
		used:       make([]float64, *config.NumSchedulers),
		cap:        make([]float64, *config.NumSchedulers),
		finSched:   make(map[int]bool),
		trial:      0,
	}
}

func (ctl *Controller) connSched(in *pb.RegRequest) {
	for {
		conn, err := grpc.Dial(in.GetIp()+":"+*config.StartPort,
			grpc.WithTransportCredentials(insecure.NewCredentials()))

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"name":  in.GetName(),
			}).Fatal("Could not connect to scheduler")
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
		InNeighbours: int32(len(ctl.rnetwork[in.GetMe()])),
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

	if int(in.GetTrial()) != ctl.trial || len(ctl.finSched) == *config.NumSchedulers {
		log.WithFields(log.Fields{
			"from":            in.GetMe(),
			"sched trial":     in.GetTrial(),
			"ctl trial":       ctl.trial,
			"number finished": len(ctl.finSched),
		}).Debug("wrong trial or already finished")
		return &pb.EmptyReply{}, nil
	}

	ctl.finSched[int(in.GetMe())] = true

	if len(ctl.finSched) == *config.NumSchedulers {
		log.WithFields(log.Fields{
			"from": in.GetMe(),
		}).Debug("all schedulers finished, sleep before a new trial")
		// we should lock while we sleep here because we don't want to do anything else
		ctl.mu.Unlock()
		time.Sleep(10 * time.Second)
		ctl.mu.Lock()
		go ctl.newTrial()
	} 
	// else {
	// 	log.WithFields(log.Fields{
	// 		"from":     in.GetMe(),
	// 		"finished": len(ctl.finSched),
	// 		"expected": *config.NumSchedulers,
	// 	}).Debug("not all schedulers finished")
	// }

	return &pb.EmptyReply{}, nil

}

func (ctl *Controller) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (ctl *Controller) Watch(in *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return errors.New("not implemented")
}

/* private functions */

func (ctl *Controller) gen_load() {
	maxCap := 10

	for i := range ctl.load {
		ctl.load[i] = float64(rand.Intn(maxCap + 1))
		ctl.used[i] = float64(rand.Intn(maxCap + 1 - int(ctl.load[i])))
		ctl.cap[i] = float64(maxCap)
	}

}

func (ctl *Controller) newTrial() {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if ctl.trial > config.MaxTrials {
		log.WithFields(log.Fields{
			"trial": ctl.trial,
		}).Info("max trials reached")
		return
	}

	ctl.trial++
	ctl.finSched = make(map[int]bool)
	ctl.gen_load()

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
		go func(i int, s string, trial int, l, u, pi float64) {
			_, err := ctl.schedStub[s].Start(context.Background(),
				&pb.StartRequest{
					Trial: int32(trial),
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

func (ctl *Controller) healthSrv() {

	lis, err := net.Listen("tcp", ":"+*config.LivenessPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve health server")
	}

	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, ctl)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("health server listening")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve health server")
		}
	}()

}
