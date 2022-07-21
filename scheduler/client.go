package scheduler

import (
	"context"
	"example/dist_sched/config"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"
)

func (sched *Scheduler) AsClient() {
	ctlAddr := findCtlAddr()

	conn, err := grpc.Dial(ctlAddr.String()+":"+*config.CtlPort, grpc.WithInsecure())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not connect to controller")
	}

	sched.ctlStub = pb.NewSchedRegClient(conn)
	sched.regWithCtl()

	neighbours := sched.getNeighbours()

	sched.connectNeigh(neighbours)

	sched.waitForFinish()
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func findCtlAddr() net.IP {
	for {
		ips, err := net.LookupIP(*config.CtlDNS)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Debug("Could not get IPs")
			time.Sleep(time.Second * 5)
		} else {
			return ips[0]
		}
	}
}

func (sched *Scheduler) regWithCtl() {

	host, err := os.Hostname()
	myIP := getOutboundIP()

	if err != nil {
		log.Fatal("Could not get hostname")
	}

	sched.hostname = host

	var r *pb.RegReply
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
		defer cancel()
		r, err = sched.ctlStub.Reg(ctx, &pb.RegRequest{
			Name: host,
			Ip:   myIP.String(),
		})

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Info("Could not register with controller")
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	sched.me = int(r.GetYou())
	log.WithFields(log.Fields{
		"me": sched.me,
	}).Info("My number")

}

func (sched *Scheduler) getNeighbours() []string {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := sched.ctlStub.GetNeighbours(ctx, &pb.NeighboursRequest{
			Me: int32(sched.me),
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Info("could not get neighbours")
			time.Sleep(time.Second)
		} else {
			log.WithFields(log.Fields{
				"neighbours": r.GetNeigh(),
			}).Info("got all neighbours")
			return r.GetNeigh()
		}
	}
}

func (sched *Scheduler) connectNeigh(neighbours []string) {
	for _, n := range neighbours {
		conn, err := grpc.Dial(n+":"+*config.SchedPort, grpc.WithInsecure(),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{}))
		if err != nil {
			log.WithFields(log.Fields{
				"neighbour address": n,
				"error":             err,
			}).Fatalf("Could not connect to neighbour")
		}

		stub := pb.NewRatioConsensusClient(conn)

		var r *pb.HelloReply
		for {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()
			r, err = stub.SayHello(ctx, &pb.HelloRequest{Name: sched.hostname})
			if err != nil {
				log.WithFields(log.Fields{
					"neighbour addr": n,
					"error": err,
				}).Warn("could not greet")
				time.Sleep(time.Second * 3)
			} else {
				break
			}
		}

		sched.outConns = append(sched.outConns, r.GetName())
		sched.stubs[r.GetName()] = stub
	}
	sched.outNeighbours = len(sched.outConns)

	log.WithFields(log.Fields{
		"number of neighbours": sched.outNeighbours,
	}).Info("connected to all neighbours")

}

func (sched *Scheduler) waitForFinish() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := sched.ctlStub.FinSetup(ctx, &pb.SetupRequest{
			Me: int32(sched.me),
		})
		if err != nil || !r.GetFinished() {
			log.WithFields(log.Fields{
				"error":     err,
				"finished":  r.GetFinished(),
			}).Info("not finished yet")
			time.Sleep(time.Second * 3)
		} else {
			log.WithFields(log.Fields{
				"in nieghbours": sched.inNeighbours,
			}).Info("all schedulers are done")
			break
		}
	}
}

// func findAddr() []net.IP {
// 	flag.Parse()
// 	found := map[string]bool{}
// 	var s []net.IP
// 	for {
// 		ips, err := net.LookupIP(*config.DNS)
// 		if err != nil {
// 			log.WithFields(log.Fields{
// 				"error": err,
// 			}).Debug("Could not get IPs")

// 			time.Sleep(time.Second * 5)
// 			continue
// 		}

// 		for _, ip := range ips {
// 			if found[ip.String()] {
// 				continue
// 			} else {
// 				s = append(s, ip)
// 				log.Printf("connected to %v", ip.String())
// 				found[ip.String()] = true
// 			}
// 		}
// 		if len(found) == *config.NumSchedulers {
// 			return s
// 		} else {
// 			log.Printf("Found %d IPs, waiting for %d", len(found), *config.NumSchedulers-len(found))
// 			if len(found) > *config.NumSchedulers {
// 				log.WithFields(log.Fields{
// 					"total number of schedulers": *config.NumSchedulers,
// 					"scheduler ips":              found,
// 				}).Info("Found too many ips")

// 				// HACK: remove ips that are no longer from the DNS service
// 				// should check where this issue comes from

// 				ipSet := make(map[string]bool)
// 				for _, ip := range ips {
// 					ipSet[ip.String()] = true
// 				}
// 				newS := []net.IP{}
// 				for _, ip := range s {
// 					if ipSet[ip.String()] {
// 						newS = append(newS, ip)
// 					}
// 				}

// 				if len(newS) > *config.NumSchedulers {
// 					log.WithFields(log.Fields{
// 						"num of ips": len(newS),
// 						"ips are":    newS,
// 					}).Panic("Still too many ips")
// 				}
// 				return newS
// 			}
// 			time.Sleep(time.Second * 2)
// 		}
// 	}

// }
