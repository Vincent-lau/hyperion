package controller

import (
	"context"
	"errors"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"golang.org/x/exp/slices"
)

type Controller struct {
	mu         sync.Mutex
	schedulers []string
	readySched map[int]bool      // schedulers that have finished connection
	schedIP    map[string]string // scheduler name -> IP addr
	network    [][]int

	pb.UnimplementedSchedRegServer
}

func (ctl *Controller) AsServer() {

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
		network:    config.Network,
	}
}

func (ctl *Controller) Reg(ctx context.Context, in *pb.RegRequest) (*pb.RegReply, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	log.WithFields(log.Fields{
		"requester data": in,
	}).Info("Received reg request")

	if !slices.Contains(ctl.schedulers, in.GetName()) {
		ctl.schedulers = append(ctl.schedulers, in.GetName())
		ctl.schedIP[in.GetName()] = in.GetIp()
		return &pb.RegReply{
			You: int32(len(ctl.schedulers) - 1),
		}, nil
	} else {
		log.WithFields(log.Fields{
			"name": in.GetName(),
		}).Info("scheduler already registered")

		return &pb.RegReply{
			You: int32(slices.Index(ctl.schedulers, in.GetName())),
		}, nil
	}

}

func (ctl *Controller) GetNeighbours(ctx context.Context, in *pb.NeighboursRequest) (*pb.NeighboursReply, error) {

	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	if len(ctl.schedulers) < *config.NumSchedulers {
		return nil, errors.New("not enough schedulers, wait more")
	}

	neighbours := make([]string, 0)
	for _, s := range ctl.network[in.GetMe()] {
		neighbours = append(neighbours, ctl.schedIP[ctl.schedulers[s]])
	}

	return &pb.NeighboursReply{
		Neigh: neighbours,
	}, nil

}

func (ctl *Controller) FinSetup(ctx context.Context, in *pb.SetupRequest) (*pb.SetupReply, error) {
	ctl.mu.Lock()
	defer ctl.mu.Unlock()

	/*  we check:
	1. all schedulers are ready
	2. the scheduler has been connected by all its in neighbours
	*/
	allReady := len(ctl.readySched) == *config.NumSchedulers
	allConnected := int(in.GetInNeighbours()) == len(config.RNetwork[int(in.GetMe())])

	if allConnected {
		ctl.readySched[int(in.GetMe())] = true
	}

	log.WithFields(log.Fields{
		"scheduler ready":  in.GetMe(),
		"scheduler status": ctl.readySched,
	}).Info("scheduler ready")


	log.WithFields(log.Fields{
		"controller reply": allReady,
		"to":               in.GetMe(),
		"in neighbours":    in.GetInNeighbours(),
	}).Info("controller reply")

	return &pb.SetupReply{
		Finished: allReady,
	}, nil

}
