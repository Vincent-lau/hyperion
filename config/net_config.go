package config

import (

	"flag"
)

var (
	Port = flag.String("port", "50051", "The server port")
	DNS = flag.String("dns", "scheduler-headless.kube-system.svc.cluster.local", "DNS server")
)

