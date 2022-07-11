package main

import (
	"example/dist_sched/client"
	"example/dist_sched/server"
	"fmt"
	"time"
)

func main() {

	go server.MyServer()
	time.Sleep(time.Second * 5)
	client.MyClient()

	for {
		fmt.Println("hello")
		time.Sleep(time.Second * 5)

	}
}
