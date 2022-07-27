package config

import (
	"flag"
)

func init() {
	flag.Parse()	
	Network = AdjList()
	RNetwork = RAdjList()
}

var (
	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	MaxIter   = flag.Int("maxiter", 500, "Maximum number of iterations")
	MFile     = flag.String("Matrix file", "data/adj.txt", "Matrix file name")

	Diameter  = flag.Int("diameter", 8, "Network diameter")
	NumSchedulers = flag.Int("num_schedulers", 9, "Number of schedulers")
	Load = []float64{4, 3, 3, 5, 5, 6, 8, 1, 4}
	Used = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0}
	Cap  = []float64{10, 10, 10, 10, 10, 10, 10,10, 10}

	Network [][]int

	// reversed network
	RNetwork [][]int

)
