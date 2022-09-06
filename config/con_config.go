package config

import (
	"flag"
	"fmt"
)

func init() {
	flag.Parse()

	MFile = fmt.Sprintf("data/adj%d.txt", *NumSchedulers)

	Network = AdjList()
	RNetwork = RAdjList()

}

var (
	Mode = flag.String("mode", "dev", "Environment to run in")	
	NumSchedulers = flag.Int("schednum", 9, "Number of schedulers")

	JobFactor = flag.Float64("jobfactor", 0.5, "Number of jobs per scheduler")
	MaxTrials = flag.Int("maxtrials", 1, "Maximum number of trials")
	MaxCap = 500.0

	Mean = 10.0
	Std = 4.0
	Skew = -4.0

	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	MaxIter   = flag.Int("maxiter", 500, "Maximum number of iterations")

	MFile string

	Diameter int

	Network [][]int

	// reversed network
	RNetwork [][]int
)
