package config


type Distr int

const (
	Normal Distr = iota
	Uniform
	Poisson
	SkewNormal
)


var (
	// randomly generated job distribution
	Distribution = Normal
	Mean = 30.0
	Std  = 4.0
	Skew = -4.0

	// do random placement when no scheduler has any capacity left
	// so that we can finishing placing all jobs
	RandomPlaceWhenNoSpace = false
	K8sPlace = false


)

