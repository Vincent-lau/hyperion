package scheduler

import (
	"context"
	pb "example/dist_sched/message"
	"log"
	"math"
	"time"
)

func (sched *Scheduler) MsgExchg() {

	// TODO parallel request
	// TODO parallel may require mutex

	for name, stub := range sched.stubs {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := stub.ExchgMax(ctx, &pb.NumRequest{Num: int64(sched.curMax)})
		log.Printf("rpcing to %v\n", name)
		log.Printf("Received response from %v: %v\n", name, r.GetNum())

		sched.curMax = int(math.Max(float64(sched.curMax), float64(r.GetNum())))

		if err != nil {
			log.Fatalf("could not exchange msg: %v", err)
		}

	}

}

func (sched *Scheduler) CheckCvg() bool {
	if sched.Done() {
		return true
	} else {
		sched.done = true
		return false
	}
}

func (sched *Scheduler) LocalComp() {

	log.Println("awesome computation done!")

}

func (sched *Scheduler) Consensus() {

	for !sched.CheckCvg() {
		sched.MsgExchg()
		sched.LocalComp()
	}
	log.Printf("consensus done! Max number is %v\n", sched.curMax)

	for {

		time.Sleep(time.Second * 10)

	}

}
