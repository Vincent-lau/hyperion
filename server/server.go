package server

import (
	"context"
	"example/dist_sched/config"
	"flag"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "example/dist_sched/message"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

func MyServer() {
	flag.Parse()
	lis, err := net.Listen("tcp", ":" + *config.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v from %v", in.GetName(), in.GetHostname())
	return &pb.HelloReply{Message: "Hello " + in.GetName() + in.GetHostname()}, nil
}
