package scheduler

import (
	"context"
	"example/dist_sched/config"
	"net"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"
)

func (sched *Scheduler) AsServer() {
	// TODO is tcp fast enough?
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
