package config


import (

	"flag"
)

var (
	Port = flag.String("port", "50051", "The server port")
	DNS = flag.String("dns", "nine-pod-headless.default.svc.cluster.local", "DNS server")
)

