package scheduler

import (
	"context"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) CheckCvg() bool {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if sched.done {
		return true
	}

	log.Println("checking convergence...")
	curData := sched.MyData()
	if sched.k%*config.Diameter == 0 && sched.k != 0 {
		if math.Abs(curData.GetMm()-curData.GetM()) < *config.Tolerance {
			sched.done = true

			log.WithFields(log.Fields{
				"name":      sched.hostname,
				"iteration": sched.k,
			}).Info("convergence reached!")
		}


		mu := curData.GetY() / curData.GetZ()


		log.WithFields(log.Fields{
			"Mm":   curData.GetMm(),
			"m":    curData.GetM(),
			"diff": math.Abs(curData.Mm - curData.M),
			"Y":    curData.GetY(),
			"Z":    curData.GetZ(),
			"mu":   mu,
			"done": sched.done,
		}).Info("updating M and m")

		sched.conData[sched.k][sched.hostname] = &pb.ConData{
			P:  curData.GetP(),
			Y:  curData.GetY(),
			Z:  curData.GetZ(),
			Mm: mu,
			M:  mu,
		}
	}

	return false

}

func (sched *Scheduler) sendOne(name string, stub pb.RatioConsensusClient, done chan<- int) {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	log.WithFields(log.Fields{
		"to":   name,
		"k":    sched.k,
		"data": sched.MyData(),
	}).Info("sending data in sendOne")

	_, err := stub.SendConData(ctx, &pb.ConDataRequest{
		K:    int32(sched.k),
		Name: sched.hostname,
		Data: sched.MyData()})

	if err != nil {
		log.WithFields(log.Fields{
			"error":              err,
			"sending request to": name,
			"iteration":          sched.k,
		}).Fatal("error requesting conData")
	}

	done <- 1

	// log.WithFields(log.Fields{
	// 	"content": r,
	// }).Info("Received response")

	// if int(r.GetK()) == sched.k {
	// 	sched.conData[r.GetName()] = r.GetData()
	// }
}

func (sched *Scheduler) MsgXchg() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	done := make(chan int)

	for _, name := range sched.outConns {
		go sched.sendOne(name, sched.stubs[name], done)
	}

	log.WithFields(log.Fields{
		"from": sched.outConns,
	}).Info("waiting for goroutine to finish sendOne")

	sched.mu.Unlock()
	for range sched.outConns {
		<-done
	}
	sched.mu.Lock()

	for len(sched.CurData())-1 != sched.inNeighbours {

		missing := make([]string, 0)
		for _, name := range sched.inConns {
			if _, ok := sched.conData[sched.k][name]; !ok {
				missing = append(missing, name)
			}
		}

		log.WithFields(log.Fields{
			"missing no":   sched.inNeighbours - (len(sched.CurData()) - 1),
			"missing from": missing,
			"k":            sched.k,
		}).Info("waiting for all responses")

		sched.cond.Wait()
	}
	log.Info("finished waiting for all responses")

}

func (sched *Scheduler) LocalComp() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	myData := sched.MyData()
	curData := sched.CurData()

	log.WithFields(log.Fields{
		"iteration": sched.k,
		"data":      myData,
	}).Info("current data")

	newY := myData.GetY() * myData.GetP()
	newZ := myData.GetZ() * myData.GetP()
	newM := myData.GetMm()
	newm := myData.GetM()

	for _, r := range sched.inConns {
		newY += curData[r].GetY() * curData[r].GetP()
		newZ += curData[r].GetZ() * curData[r].GetP()

		newM = math.Max(newM, curData[r].GetMm())
		newm = math.Min(newm, curData[r].GetM())
	}

	// sched.k+1 might have been created by receiving response from other nodes
	if _, ok := sched.conData[sched.k+1]; !ok {
		sched.conData[sched.k+1] = make(map[string]*pb.ConData)
	}

	sched.conData[sched.k+1][sched.hostname] = &pb.ConData{
		P:  myData.GetP(),
		Y:  newY,
		Z:  newZ,
		M:  newm,
		Mm: newM,
	}

	log.WithFields(log.Fields{
		"to": sched.k + 1,
		"updated data": sched.conData[sched.k+1][sched.hostname],
	}).Info("awesome computation done, advancing iteration counter")

	if (!sched.done) {
		sched.k++
	}

}

func (sched *Scheduler) Consensus() {
	for !sched.CheckCvg() && sched.k < 50 {

		log.Printf("doing msg exchange...")
		sched.MsgXchg()

		log.Println("doing local computation...")
		sched.LocalComp()
	}

	log.WithFields(log.Fields{
		"iteration": sched.k,
		"data":      sched.MyData(),
		"average consensus": sched.MyData().GetY() / sched.MyData().GetZ(),
	}).Info("consensus done!")
}

func (sched *Scheduler) CurData() map[string]*pb.ConData {
	return sched.conData[sched.k]
}

func (sched *Scheduler) MyData() *pb.ConData {
	return sched.conData[sched.k][sched.hostname]
}
