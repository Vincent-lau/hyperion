package config

import (
	"flag"
)

func init() {
	
	Network = AdjList()
	RNetwork = RAdjList()
}

var (
	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	Diameter  = flag.Int("diameter", 8, "Network diameter")

	Load = []float64{1, 9, 3, 6, 9, 7, 6, 5, 2}
	Used = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0}
	Cap  = []float64{10, 10, 10, 10, 10, 10, 10, 10, 10}


	Network [][]int

	// reversed network
	RNetwork [][]int

	NumSchedulers = flag.Int("num_schedulers", 9, "Number of schedulers")
)
