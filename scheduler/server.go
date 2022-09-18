package scheduler

import (
	"context"
	"example/dist_sched/config"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	pb "example/dist_sched/message"

	"google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/protobuf/proto"
)

func (sched *Scheduler) healthSrv() {
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
	}).Debug("health server listening")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve health server")
		}
	}()
}

func (sched *Scheduler) schedStartSrv() {
	lis, err := net.Listen("tcp", ":"+*config.StartPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve start server")
	}

	s := grpc.NewServer()
	pb.RegisterSchedStartServer(s, sched)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("start server listening")

	go func() {
		if err := s.Serve(lis); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatalf("failed to serve start server")
		}
	}()
}

func (sched *Scheduler) AsServer() {

	sched.healthSrv() // for k8s liveness probing

	sched.schedStartSrv() // for controller to start the scheduler

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

func (sched *Scheduler) Ping(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if !slices.Contains(sched.inConns, int(in.GetMe())) {
		atomic.AddUint64(&sched.inNeighbours, 1)
		sched.inConns = append(sched.inConns, int(in.GetMe()))
	}

	if sched.expectedIn != 0 && sched.expectedIn == sched.inNeighbours {
		// graph is strongly connected, hence >=1 in neighbours
		log.WithFields(log.Fields{
			"expected in": sched.expectedIn,
		}).Debug("all neighbours connected, broadcasting")
		sched.neighCond.Broadcast()
	}

	log.WithFields(log.Fields{"from": in.GetMe()}).Debug("Received hello")
	return &pb.HelloReply{Me: int32(sched.me)}, nil
}

func (sched *Scheduler) SendConData(stream pb.RatioConsensus_SendConDataServer) error {
	for {
		in, err := stream.Recv()
		t := time.Now()

		if err == io.EOF {
			return stream.SendAndClose(&pb.EmptyReply{})
		} else if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("failed to receive con data")
		}

		k := in.GetK()

		kData, _ := sched.conData.LoadOrStore(k, &sync.Map{})
		kData.(*sync.Map).Store(int(in.GetMe()), in.GetData())
		c, _ := sched.conLen.LoadOrStore(k, uint64(0))
		sched.conLen.Store(k, c.(uint64)+1)

		s := uint64(proto.Size(in))
		atomic.AddUint64(&sched.msgRcv, s)

		x, _ := sched.conLen.Load(sched.k)

		log.WithFields(log.Fields{
			"from":               in.GetMe(),
			"scheduler k":        sched.k,
			"received k":         in.GetK(),
			"data":               in.GetData(),
			"expecting total":    sched.inNeighbours,
			"currently received": x.(uint64) - 1,
			"elapsed":            time.Since(t),
		}).Debug("received data")

		if x.(uint64)-1 == sched.inNeighbours {
			// received from all inbound neighbours
			log.Debug("received from all inbound neighbours, broadcasting")
			sched.mu.Lock()
			sched.neighCond.Broadcast()
			sched.mu.Unlock()
		}
	}

}

func (sched *Scheduler) StartConsensus(ctx context.Context, in *pb.StartRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	log.WithFields(log.Fields{
		"l":  in.GetL(),
		"u":  in.GetU(),
		"pi": in.GetPi(),
	}).Debug("received start from controller")

	sched.InitMyConData(in.GetL(), in.GetU(), in.GetPi())
	sched.trial = int(in.GetTrial())

	MetricsLogger = MetricsLogger.WithFields(log.Fields{
		"trial": in.GetTrial(),
	})
	PlLogger = PlLogger.WithFields(log.Fields{
		"trial": in.GetTrial(),
	})

	sched.startCond.Broadcast()
	sched.setup = true

	return &pb.EmptyReply{}, nil
}

func (sched *Scheduler) StartPlace(ctx context.Context, in *pb.EmptyRequest) (*pb.EmptyReply, error) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	log.Debug("received start placement from controller")

	sched.startCond.Broadcast()
	sched.allDone = true

	return &pb.EmptyReply{}, nil

}
