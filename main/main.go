package main

import (
	"example/dist_sched/scheduler"
)

func talk() {
	go scheduler.MyServer()
	scheduler.MyClient()
}


func main() {
	scheduler := scheduler.New()
	scheduler.Schedule()
}
