package main

import (
	"example/dist_sched/client"
	"example/dist_sched/server"
	"example/dist_sched/rand_sched"
	"fmt"
	"time"
)

func talk() {
	go server.MyServer()
	time.Sleep(time.Second * 5)
	client.MyClient()

	for {
		fmt.Println("hello")
		time.Sleep(time.Second * 5)
	}
}

func sched() {
	rand_sched.Schedule()
}


func main() {
	sched()
}
