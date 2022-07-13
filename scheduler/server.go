package scheduler

import (
	"flag"
	"context"
	"example/dist_sched/config"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "example/dist_sched/message"
)

func MyServer() {
	flag.Parse()
	lis, err := net.Listen("tcp", ":" + *config.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &Scheduler{})
	pb.RegisterMaxConsensusServer(s, &Scheduler{})

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// SayHello implements helloworld.GreeterServer
func (sched *Scheduler) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v from %v", in.GetName(), in.GetHostname())
	return &pb.HelloReply{Message: "Hello " + in.GetName() + in.GetHostname()}, nil
}

func (sched *Scheduler) ExchgMax(ctx context.Context, in *pb.NumRequest) (*pb.NumReply, error) {
	log.Printf("Received: %v", in.GetNum())
	return &pb.NumReply{Num: int64(sched.curMax)}, nil
}
