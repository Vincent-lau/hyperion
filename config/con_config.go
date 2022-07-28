package config

import (
	"flag"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

func gen_load() {
	maxCap := 10

	for i := range Load {
		Load[i] = float64(rand.Intn(maxCap + 1))
		Used[i] = float64(rand.Intn(maxCap + 1 - int(Load[i])))
		Cap[i] = float64(maxCap)
	}

}

func init() {
	flag.Parse()
	Network = AdjList()
	RNetwork = RAdjList()
	gen_load()

	log.WithFields(log.Fields{
		"load": Load,
		"used": Used,
		"cap":  Cap,
	}).Info("initial data")

}

var (
	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	MaxIter   = flag.Int("maxiter", 500, "Maximum number of iterations")
	MFile     = flag.String("Matrix file", "data/adj.txt", "Matrix file name")

	NumSchedulers = 200
	Diameter      int
	Load          = make([]float64, NumSchedulers)
	Used          = make([]float64, NumSchedulers)
	Cap           = make([]float64, NumSchedulers)

	Network [][]int

	// reversed network
	RNetwork [][]int
)
