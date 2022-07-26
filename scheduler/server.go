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
)

func (sched *Scheduler) HealthSrv() {
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
	}).Info("health server listening")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve health server")
		}
	}()

}

func (sched *Scheduler) AsServer() {

	sched.HealthSrv()

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

// SayHello implements helloworld.GreeterServer
func (sched *Scheduler) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if !slices.Contains(sched.inConns, in.GetName()) {
		sched.inNeighbours++
		sched.inConns = append(sched.inConns, in.GetName())
	}

	log.Printf("Received hello from %v", in.GetName())
	return &pb.HelloReply{Name: sched.hostname}, nil
}

func (sched *Scheduler) SendConData(ctx context.Context, in *pb.ConDataRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	k := int(in.GetK())
	if _, ok := sched.conData[k]; !ok {
		sched.conData[k] = make(map[string]*pb.ConData)
	}
	sched.conData[k][in.GetName()] = in.GetData()

	log.WithFields(log.Fields{
		"from":               in.GetName(),
		"scheduler k":        sched.k,
		"received k":         in.GetK(),
		"data":               in.GetData(),
		"expecting total":    sched.inNeighbours,
		"currently received": len(sched.CurData()) - 1,
	}).Info("Received data")

	if len(sched.CurData())-1 == sched.inNeighbours {
		// received from all inbound neighbours
		sched.cond.Broadcast()
	}

	if sched.timeToCheck() && int(in.GetK()) > sched.k {
		// received an update from a non-terminating node
		sched.cond.Broadcast()
	}

	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func (sched *Scheduler) Watch(in *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	return errors.New("not implemented")
}
