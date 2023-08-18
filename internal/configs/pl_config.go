package config

type Distr int

type HeuFunc int

const (
	Random HeuFunc = iota
	Large2small
)

const (
	Normal Distr = iota
	Uniform
	Poisson
	SkewNormal
)

const (
	// randomly generated job distribution
	Distribution = Normal
	// whehter generate jobs based on static data here or based on number of jobs
	DistrStatic  = false 
                         
	Mean         = 30.0
	Std          = 4.0
	Skew         = -4.0

	// do random placement when no scheduler has any capacity left
	// so that we can finishing placing all jobs
	RandomPlaceWhenNoSpace = false
	K8sPlace               = false // get jobs from actual k8s workload and place them onto actual nodes

	// heuristic function, see policy.go
	Heuristic = Large2small
)
