package main

import (
	"example/dist_sched/scheduler"
	"math/rand"
	"log"
	"time"
)

func talk() {
	go scheduler.MyServer()
	scheduler.MyClient()
}


func main() {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(10)
	log.Printf("n = %d\n", n)
	scheduler := scheduler.New(n)
	scheduler.Consensus()
}
