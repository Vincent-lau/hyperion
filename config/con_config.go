package config

import (
	"flag"
)

var (
	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	Diameter  = flag.Int("diameter", 4, "Network diameter")

	Load = []float64{1, 2, 3, 4, 5}
	Used = []float64{0, 0, 0, 0, 0}
	Cap  = []float64{10, 10, 10, 10, 10}


	Network = [][]int{
		{1, 2},
		{2, 4},
		{4},
		{0},
		{2, 3},
	}

	NumSchedulers = flag.Int("num_schedulers", 5, "Number of schedulers")
)
