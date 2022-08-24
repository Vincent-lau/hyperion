package scheduler

import (
	"context"
	"errors"
	"example/dist_sched/config"
	"net"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"

	"google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/protobuf/proto"
)

func (sched *Scheduler) healthSrv() {
	lis, err := net.Listen("tcp", ":"+*config.LivenessPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve health server")
	}

	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, sched)

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

func (sched *Scheduler) schedStartSrv() {
	lis, err := net.Listen("tcp", ":"+*config.StartPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
	pb.RegisterSchedStartServer(s, sched)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("start server listening")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve start server")
		}
	}()
}

func (sched *Scheduler) AsServer() {

	sched.healthSrv() // for k8s liveness probing

	sched.schedStartSrv() // for controller to start the scheduler

	// grpc will multiplex the connection over a single TCP connection
	// so tcp is fine here
	lis, err := net.Listen("tcp", ":"+*config.SchedPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{}))
	pb.RegisterRatioConsensusServer(s, sched)

	log.Printf("server listening at %v", lis.Addr())
	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve")
		}
	}()

}

func (sched *Scheduler) Ping(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if !slices.Contains(sched.inConns, int(in.GetMe())) {
		sched.inNeighbours++
		sched.inConns = append(sched.inConns, int(in.GetMe()))
	}

	if sched.expectedIn != 0 && sched.expectedIn == sched.inNeighbours {
		// graph is strongly connected, hence >=1 in neighbours
		log.WithFields(log.Fields{
			"expected in": sched.expectedIn,
		}).Debug("all neighbours connected, broadcasting")
		sched.neighCond.Broadcast()
	}

	log.WithFields(log.Fields{"from": in.GetMe()}).Debug("Received hello")
	return &pb.HelloReply{Me: int32(sched.me)}, nil
}

func (sched *Scheduler) SendConData(ctx context.Context, in *pb.ConDataRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	k := int(in.GetK())
	if _, ok := sched.conData[k]; !ok {
		sched.conData[k] = make(map[int]*pb.ConData)
	}
	sched.conData[k][int(in.GetMe())] = in.GetData()

	log.WithFields(log.Fields{
		"from":               in.GetMe(),
		"scheduler k":        sched.k,
		"received k":         in.GetK(),
		"data":               in.GetData(),
		"expecting total":    sched.inNeighbours,
		"currently received": len(sched.CurData()) - 1,
	}).Debug("received data")

	s := uint64(proto.Size(in))
	sched.msgRcv += s

	if len(sched.CurData())-1 == sched.inNeighbours {
		// received from all inbound neighbours
		log.Debug("received from all inbound neighbours, broadcasting")
		sched.neighCond.Broadcast()
	}

	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (sched *Scheduler) Watch(in *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return errors.New("not implemented")
}

func (sched *Scheduler) StartConsensus(ctx context.Context, in *pb.StartRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	log.WithFields(log.Fields{
		"l":  in.GetL(),
		"u":  in.GetU(),
		"pi": in.GetPi(),
	}).Debug("received start from controller")

	sched.InitMyConData(in.GetL(), in.GetU(), in.GetPi())
	sched.trial = int(in.GetTrial())

	MetricsLogger = MetricsLogger.WithFields(log.Fields{
		"trial": in.GetTrial(),
	})
	PlLogger = PlLogger.WithFields(log.Fields{
		"trial": in.GetTrial(),
	})

	sched.startCond.Broadcast()
	sched.setup = true

	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) StartPlace(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	log.Debug("received start placement from controller")

	sched.startCond.Broadcast()
	sched.allDone = true

	return &pb.EmptyReply{}, nil

}
