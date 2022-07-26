package scheduler

import (
	"context"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

func (sched *Scheduler) timeToCheck() bool {
	return sched.k%*config.Diameter == 0 && sched.k != 0
}

func (sched *Scheduler) CheckCvg() bool {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if sched.done {

		for {
			if len(sched.conData[sched.k+1]) > 1 {
				// we have received a response from another node, so continue the consensus
				// and we have already put our data in the k+1 in LocalComp
				sched.k++

				log.WithFields(log.Fields{
					"current k": sched.k,
					"my data":   sched.MyData(),
				}).Info("Have reached convergence but exiting due to other schedulers not terminating")

				sched.done = false
				break
			} else {
				// TODO need some kind of controller notification in case all have terminated
				// otherwise wait forever when all terminated

				log.Info("waiting for other schedulers to reach convergence")

				sched.cond.Wait()
			}
		}

	}

	log.Println("checking convergence...")
	curData := sched.MyData()
	if sched.timeToCheck() {
		if math.Abs(curData.GetMm()-curData.GetM()) < *config.Tolerance {
			sched.done = true

			log.WithFields(log.Fields{
				"name":      sched.hostname,
				"iteration": sched.k,
				"data":      sched.MyData(),
				"ratio":     sched.MyData().GetY() / sched.MyData().GetZ(),
			}).Info("convergence reached for this node!")
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
		}).Info("so updating M and m")

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

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		log.WithFields(log.Fields{
			"to":   name,
			"k":    sched.k,
			"data": sched.MyData(),
		}).Info("sending data in sendOne")

		data := &pb.ConDataRequest{
			K:    int32(sched.k),
			Name: sched.hostname,
			Data: sched.MyData(),
		}


		sched.mu.Unlock()
		_, err := stub.SendConData(ctx,data)
		sched.mu.Lock()

		if err != nil {
			log.WithFields(log.Fields{
				"error":              err,
				"sending request to": name,
				"iteration":          sched.k,
			}).Warn("error send conData")

			sched.mu.Unlock()
			time.Sleep(time.Second * 2)
			sched.mu.Lock()

		} else {
			break
		}
	}

	done <- 1

}

func (sched *Scheduler) MsgXchg() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	done := make(chan int)

	for _, name := range sched.outConns {
		go sched.sendOne(name, sched.stubs[name], done)
	}

	log.WithFields(log.Fields{
		"to": sched.outConns,
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
		"to":           sched.k + 1,
		"updated data": sched.conData[sched.k+1][sched.hostname],
	}).Info("awesome computation done, advancing iteration counter")

	if !sched.done {
		sched.k++
	}

}

func (sched *Scheduler) Consensus() {
	for !sched.CheckCvg() && sched.k < *config.MaxIter {

		log.Printf("doing msg exchange...")
		sched.MsgXchg()

		log.Println("doing local computation...")
		sched.LocalComp()
	}

	log.WithFields(log.Fields{
		"iteration":         sched.k,
		"data":              sched.MyData(),
		"average consensus": sched.MyData().GetY() / sched.MyData().GetZ(),
	}).Info("consensus done!")
}

func (sched *Scheduler) CurData() map[string]*pb.ConData {
	return sched.conData[sched.k]
}

func (sched *Scheduler) MyData() *pb.ConData {
	return sched.conData[sched.k][sched.hostname]
}
