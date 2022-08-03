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
	NumSchedulers = flag.Int("schednum", 9, "Number of schedulers")
	MaxTrials = 10

	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	MaxIter   = flag.Int("maxiter", 500, "Maximum number of iterations")

	MFile string

	Diameter int

	Network [][]int

	// reversed network
	RNetwork [][]int
)
