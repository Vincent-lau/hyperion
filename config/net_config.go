package config

import (
	"flag"
)

var (
	SchedPort = flag.String("port", "50051", "The server port")
	CtlPort   = flag.String("ctlport", "50052", "The controller port")
	SchedDNS  = flag.String("scheduler dns", "scheduler-headless.kube-system.svc.cluster.local", "DNS server")
	CtlDNS    = flag.String("controller dns", "controller-svc.kube-system.svc.cluster.local", "controller DNS server")

	LivenessPort = flag.String("liveness port", "2379", "The liveness port")
)
