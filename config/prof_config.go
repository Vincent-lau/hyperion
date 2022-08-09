package config


import (
	"flag"
)


func init() {
	flag.Parse()
}


var (
	CpuProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	MemProfile = flag.String("memprofile", "", "write memory profile to file")
)

