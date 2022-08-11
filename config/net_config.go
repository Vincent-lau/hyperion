package config

import (
	"flag"
)

var (
	SchedPort = flag.String("port", "50051", "The server port")
	CtlPort   = flag.String("ctlport", "50052", "The controller port")
	PlacementPort = "50053"
	SchedDNS  = flag.String("scheduler dns", "scheduler-headless.dist-sched.svc.cluster.local", "DNS server")
	CtlDNS    = flag.String("controller dns", "controller-svc.dist-sched.svc.cluster.local", "controller DNS server")

	StartPort = flag.String("startport", "50053", "The start port")

	LivenessPort = flag.String("liveness port", "2379", "The liveness port")
)
