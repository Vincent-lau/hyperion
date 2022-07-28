package config


import (
	"flag"
)


func init() {
	flag.Parse()
}


var (
	Cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

