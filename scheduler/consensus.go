package scheduler

import (
	"context"
	pb "example/dist_sched/message"
	"log"
	"math"
	"time"
)


func (sched *Scheduler) getOne(name string, stub pb.MaxConsensusClient, replyChan chan *pb.NumReply) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	sched.mu.Unlock()
	r, err := stub.GetMax(ctx, &pb.EmptyRequest{})
	replyChan <- r
	sched.mu.Lock()

	if err != nil {
		log.Fatalf("could not exchange msg: %v", err)
	}

	log.Printf("Received response from %v: %v\n", name, r.GetNum())

}

func (sched *Scheduler) MsgExchg() []*pb.NumReply {
	replyChan := make(chan *pb.NumReply)

	for name, stub := range sched.stubs {
		go sched.getOne(name, stub, replyChan)
	}

	var s []*pb.NumReply
	for r := range replyChan {
		s = append(s, r)
		log.Printf("current len of s is %v and in neighbour is %v \n", len(s), sched.inNeighbour)
		if len(s) == sched.inNeighbour {
			break
		}
	}
	return s

}

func (sched *Scheduler) CheckCvg() bool {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if sched.it > 10 {
		sched.done = true
		return true
	} else {
		return false
	}
}

func (sched *Scheduler) LocalComp(reply []*pb.NumReply) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	for _, r := range reply {
		if r.GetIt() < int32(sched.it) {
			log.Printf("%v counter vs current counter %v too old, ignore\n", r.GetIt(), sched.it)
		} else {
			sched.curMax = int(math.Max(float64(sched.curMax), float64(r.GetNum())))
		}
	}

	log.Println("awesome computation done!")

}

func (sched *Scheduler) Consensus() {
	for !sched.CheckCvg() {
		sched.mu.Lock()
		sched.it++
		sched.mu.Unlock()

		reply := sched.MsgExchg()
		sched.LocalComp(reply)
	}
	log.Printf("consensus done! Max number is %v\n", sched.curMax)

	for {
		time.Sleep(time.Second * 10)
	}

}
