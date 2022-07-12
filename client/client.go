package client


import (
	"fmt"
	"os"
	"net"
	"flag"
	"log"
	"context"
	"time"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"example/dist_sched/config"

	pb "example/dist_sched/message"
)

const (
	defaultName = "world"
)

var (
	name = flag.String("name", defaultName, "Name to greet")
)


func findAddr() []net.IP {
	for {
		ips, err := net.LookupIP(*config.DNS)
		if err != nil {
			log.Printf("Could not get IPs: %v\n", err)
			time.Sleep(time.Second)
			continue
		}

		if len(ips) == 9 {
			for i, ip := range ips {
				fmt.Printf("addr %d IN A %s\n", i, ip.String())
			}
			return ips
		} else {
			log.Printf("current %v ips, not enough IPs, retrying\n", len(ips))
			// TODO can use channel instead of sleep
			time.Sleep(time.Second * 5)
		}
	}

}

func MyClient() {

	flag.Parse()
	// Set up a connection to the server.
	ips := findAddr()
	for _, v := range ips {
		conn, err := grpc.Dial(v.String() + ":" + *config.Port, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("failed to connect")
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewGreeterClient(conn)

		// Contact the server and print out its response.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		host, err := os.Hostname()
		if err != nil {
			log.Fatalf("Could not get hostname: %v", err)
		}
		r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name, Hostname: host})
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}

		log.Printf("Greeting: %s", r.GetMessage())

	}
}
