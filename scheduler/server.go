package scheduler

import (
	"context"
	"example/dist_sched/config"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "example/dist_sched/message"
)

func (sched *Scheduler) MyServer() {
	flag.Parse()
	lis, err := net.Listen("tcp", ":"+*config.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, sched)
	pb.RegisterMaxConsensusServer(s, sched)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// SayHello implements helloworld.GreeterServer
func (sched *Scheduler) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	sched.inNeighbour++

	log.Printf("Received: %v from %v", in.GetName(), in.GetHostname())
	return &pb.HelloReply{Message: "Hello " + in.GetName() + in.GetHostname()}, nil
}

func (sched *Scheduler) GetMax(ctx context.Context, in *pb.EmptyRequest) (*pb.NumReply, error) {

	return &pb.NumReply{Num: int64(sched.curMax), It: int32(sched.it)}, nil
}


