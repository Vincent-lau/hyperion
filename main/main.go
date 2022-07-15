package main

import (
	"example/dist_sched/scheduler"
	"math/rand"
	"time"
)



func main() {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10)
	scheduler := scheduler.New(n)
	scheduler.Consensus()
}
