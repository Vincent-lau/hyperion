package scheduler

import (
	"context"
	"example/dist_sched/config"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"
)

func (sched *Scheduler) AsClient() {
	ctlAddr := findCtlAddr()

	conn, err := grpc.Dial(ctlAddr.String()+":"+*config.CtlPort,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	sched.mu.Lock()
	defer sched.mu.Unlock()

	host, err := os.Hostname()
	myIP := getOutboundIP()

	if err != nil {
		log.Fatal("Could not get hostname")
	}

	sched.hostname = host

	var r *pb.RegReply
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		sched.mu.Unlock()
		r, err = sched.ctlStub.Reg(ctx, &pb.RegRequest{
			Name: host,
			Ip:   myIP.String(),
		})
		sched.mu.Lock()

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Debug("Could not register with controller")
			sched.mu.Unlock()
			time.Sleep(time.Second)
			sched.mu.Lock()
		} else {
			break
		}
	}

	sched.me = int(r.GetYou())
	log.WithFields(log.Fields{
		"me": sched.me,
	}).Debug("My number")

}

func (sched *Scheduler) getNeighbours() []string {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		sched.mu.Unlock()
		r, err := sched.ctlStub.GetNeighbours(ctx, &pb.NeighboursRequest{
			Me: int32(sched.me),
		})
		sched.mu.Lock()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Debug("could not get neighbours")

			sched.mu.Unlock()
			time.Sleep(time.Second)
			sched.mu.Lock()
		} else {
			sched.expectedIn = int(r.GetInNeighbours())

			log.WithFields(log.Fields{
				"neighbours": r.GetNeigh(),
				"expected in": r.GetInNeighbours(),
			}).Debug("got neighbours")
			return r.GetNeigh()
		}
	}
}

func (sched *Scheduler) connectNeigh(neighbours []string) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	for _, n := range neighbours {
		conn, err := grpc.Dial(n+":"+*config.SchedPort, 
			grpc.WithTransportCredentials(insecure.NewCredentials()),
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
			sched.mu.Unlock()
			r, err = stub.SayHello(ctx, &pb.HelloRequest{Name: sched.hostname})
			sched.mu.Lock()
			if err != nil {
				log.WithFields(log.Fields{
					"neighbour addr": n,
					"error":          err,
				}).Warn("could not greet")
				sched.mu.Unlock()
				time.Sleep(time.Second * 3)
				sched.mu.Lock()
			} else {
				log.WithFields(log.Fields{
					"to": r.GetName(),
				}).Debug("greeted")
				break
			}
		}

		sched.outConns = append(sched.outConns, r.GetName())
		sched.stubs[r.GetName()] = stub
	}
	sched.outNeighbours = len(sched.outConns)

	log.WithFields(log.Fields{
		"number of out neighbours": sched.outNeighbours,
	}).Debug("connected to all neighbours")

	// now wait for all neighbours to connect to me
	for sched.expectedIn != sched.inNeighbours {
		log.WithFields(log.Fields{
			"expected in": sched.expectedIn,
			"in neighbours": sched.inNeighbours,
		}).Debug("waiting for all neighbours to connect to me")
		sched.neighCond.Wait()
	}

	log.Debug("all neighbours connected to me")

}

func (sched *Scheduler) waitForFinish() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		sched.mu.Unlock()
		r, err := sched.ctlStub.FinSetup(ctx, &pb.SetupRequest{
			Me:           int32(sched.me),
			InNeighbours: int64(sched.inNeighbours),
		})
		sched.mu.Lock()

		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"finished": r.GetFinished(),
			}).Warn("error sending finish to controller")
			sched.mu.Unlock()
			time.Sleep(time.Second)
			sched.mu.Lock()
		} else {
			break
		}
	}

	for !sched.setup {
		sched.startCond.Wait()
	}

	log.WithFields(log.Fields{
		"in nieghbours": sched.inNeighbours,
	}).Debug("received finish setup from controller, all schedulers are connected")

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
// 		if len(found) == config.NumSchedulers {
// 			return s
// 		} else {
// 			log.Printf("Found %d IPs, waiting for %d", len(found), config.NumSchedulers-len(found))
// 			if len(found) > config.NumSchedulers {
// 				log.WithFields(log.Fields{
// 					"total number of schedulers": config.NumSchedulers,
// 					"scheduler ips":              found,
// 				}).Debug("Found too many ips")

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

// 				if len(newS) > config.NumSchedulers {
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
