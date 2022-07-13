package scheduler

import (
	"flag"
	"context"
	"example/dist_sched/config"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	pb "example/dist_sched/message"
)

const (
	defaultName = "world"
	numSchedulers = 9
)

var (
	name = flag.String("name", defaultName, "Name to greet")
)


func findAddr(c chan net.IP) {
	flag.Parse()
	found := map[string]bool{}
	for {
		ips, err := net.LookupIP(*config.DNS)
		if err != nil {
			log.Printf("Could not get IPs: %v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}

		for _, ip := range ips {
			if found[ip.String()] {
				continue
			} else {
				c <- ip
				found[ip.String()] = true
			}
		}
		if len(found) == numSchedulers {
			close(c)
			break
		} else {
			log.Printf("Found %d IPs, waiting for %d\n", len(found), numSchedulers-len(found))
			time.Sleep(time.Second * 5)
		}
	}

}

func MyClient() (map[string]*grpc.ClientConn, map[string]pb.MaxConsensusClient) {
	conns := make(map[string]*grpc.ClientConn)
	stubs := make(map[string]pb.MaxConsensusClient)
	addrChan := make(chan net.IP)
	go findAddr(addrChan)
	for addr := range addrChan {
		conn, err := grpc.Dial(addr.String() + ":" + *config.Port, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Could not connect to %v: %v\n", addr, err)
		}
		conns[addr.String()] = conn

		stub := pb.NewGreeterClient(conn)

		// Contact the server and print out its response.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		host, err := os.Hostname()
		if err != nil {
			log.Fatalf("Could not get hostname: %v", err)
		}
		_, err = stub.SayHello(ctx, &pb.HelloRequest{Name: *name, Hostname: host})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}

		stubs[addr.String()] = pb.NewMaxConsensusClient(conn)
	}

	return conns,stubs
}

