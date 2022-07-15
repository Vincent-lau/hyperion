package scheduler

import (
	"context"
	"example/dist_sched/config"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"
)

const (
	defaultName   = "world"
	numSchedulers = 9
)

var (
	name = flag.String("name", defaultName, "Name to greet")
)

func findAddr() []net.IP {
	flag.Parse()
	found := map[string]bool{}
	var s []net.IP
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
				s = append(s, ip)
				found[ip.String()] = true
			}
		}
		if len(found) == numSchedulers {
			return s
		} else {
			log.Printf("Found %d IPs, waiting for %d\n", len(found), numSchedulers-len(found))
			time.Sleep(time.Second * 5)
		}
	}

}

func (sched *Scheduler) MyClient() {
	addrSlice := findAddr()

	for _, addr := range addrSlice {
		conn, err := grpc.Dial(addr.String()+":"+*config.Port, grpc.WithInsecure(),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{}))
		if err != nil {
			log.Fatalf("Could not connect to %v: %v\n", addr, err)
		}
		sched.conns[addr.String()] = conn

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

		sched.stubs[addr.String()] = pb.NewMaxConsensusClient(conn)
	}

}
