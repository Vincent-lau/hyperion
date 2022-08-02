package config

import (
	"flag"
	"fmt"
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

	MFile = fmt.Sprintf("data/adj%d.txt", *NumSchedulers)
	Load = make([]float64, *NumSchedulers)
	Used = make([]float64, *NumSchedulers)
	Cap = make([]float64, *NumSchedulers)

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
	NumSchedulers = flag.Int("schednum", 9, "Number of schedulers")

	Tolerance = flag.Float64("tolerance", 1e-5, "Tolerance for the convergence")
	Delay     = flag.Int("tau", 1, "Delay")
	MaxIter   = flag.Int("maxiter", 500, "Maximum number of iterations")

	MFile string

	Diameter int
	Load     []float64
	Used     []float64
	Cap      []float64

	Network [][]int

	// reversed network
	RNetwork [][]int
)
