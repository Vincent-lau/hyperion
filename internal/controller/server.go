package controller

import (
	config "github.com/Vincent-lau/hyperion/internal/configs"
	pb "github.com/Vincent-lau/hyperion/internal/message"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
)

func (ctl *Controller) AsServer() {
	go ctl.healthSrv()
	go ctl.placementSrv()
	ctl.regSrv()
}

func (ctl *Controller) regSrv() {
	lis, err := net.Listen("tcp", ":"+*config.CtlPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSchedRegServer(s, ctl)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("registration server listening")

	if err := s.Serve(lis); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve")
	}
}

func (ctl *Controller) healthSrv() {

	lis, err := net.Listen("tcp", ":"+*config.LivenessPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve health server")
	}

	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, ctl)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("health server listening")

	if err := s.Serve(lis); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve health server")
	}

}

func (ctl *Controller) placementSrv() {

	lis, err := net.Listen("tcp", ":"+config.PlacementPort)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve placement server")
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{}),
	)
	pb.RegisterJobPlacementServer(s, ctl)

	log.WithFields(log.Fields{
		"at": lis.Addr(),
	}).Debug("placement server listening")

	if err := s.Serve(lis); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("failed to serve placement server")
	}

}
