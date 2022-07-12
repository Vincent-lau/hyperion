package main

import (
	"example/dist_sched/client"
	"example/dist_sched/scheduler"
	"example/dist_sched/server"
)

func talk() {
	go server.MyServer()
	client.MyClient()

}

func sched() {
	scheduler.Schedule()
}


func main() {
	talk()
	sched()
}
