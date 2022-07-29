package scheduler

import (
	"context"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"math"
	"os"
	"runtime/pprof"
	"time"

	log "github.com/sirupsen/logrus"
)

var metricsLogger = log.WithFields(log.Fields{"prefix": "metrics"})

func (sched *Scheduler) timeToCheck() bool {
	return sched.k%config.Diameter == 0 && sched.k != 0
}

func (sched *Scheduler) getPrevRoundFlag() bool {
	if sched.k >= config.Diameter {

		log.WithFields(log.Fields{
			"k":           sched.k,
			"prev k":      sched.k - config.Diameter,
			"prev k flag": sched.conData[sched.k-config.Diameter][sched.hostname].GetFlag(),
		}).Debug("checking flag at previous round")

		return sched.conData[sched.k-config.Diameter][sched.hostname].GetFlag()
	} else {
		return false
	}
}

func (sched *Scheduler) CheckCvg() bool {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	if sched.done {
		return true
	}

	log.Debug("checking convergence...")
	myData := sched.MyData()

	if !myData.GetFlag() {
		flag := false
		if sched.timeToCheck() {
			if math.Abs(myData.GetMm()-myData.GetM()) < *config.Tolerance {
				log.WithFields(log.Fields{
					"name":      sched.hostname,
					"iteration": sched.k,
					"data":      sched.MyData(),
					"ratio":     sched.MyData().GetY() / sched.MyData().GetZ(),
				}).Debug("flag raised for this node")
				flag = true
			}
			mu := myData.GetY() / myData.GetZ()

			log.WithFields(log.Fields{
				"Mm":   myData.GetMm(),
				"m":    myData.GetM(),
				"diff": math.Abs(myData.Mm - myData.M),
				"Y":    myData.GetY(),
				"Z":    myData.GetZ(),
				"mu":   mu,
			}).Debug("updating M and m")

			sched.conData[sched.k][sched.hostname] = &pb.ConData{
				P:    myData.GetP(),
				Y:    myData.GetY(),
				Z:    myData.GetZ(),
				Mm:   mu,
				M:    mu,
				Flag: flag,
			}

			// now check for termination
			if sched.getPrevRoundFlag() && flag {
				log.WithFields(log.Fields{
					"name":  sched.hostname,
					"k":     sched.k,
					"data":  sched.MyData(),
					"ratio": sched.MyData().GetY() / sched.MyData().GetZ(),
				}).Debug("termination reached")

				sched.done = true
			}

		}

	} else { // flag == 1
		if math.Abs(myData.GetMm()-myData.GetM()) >= *config.Tolerance {
			sched.conData[sched.k][sched.hostname] = &pb.ConData{
				P:    myData.GetP(),
				Y:    myData.GetY(),
				Z:    myData.GetZ(),
				Mm:   myData.GetMm(),
				M:    myData.GetM(),
				Flag: false,
			}
			log.WithFields(log.Fields{
				"at":   sched.k,
				"data": sched.MyData(),
			}).Debug("flip!")
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
		}).Debug("sending data in sendOne")

		data := &pb.ConDataRequest{
			K:    int32(sched.k),
			Name: sched.hostname,
			Data: sched.MyData(),
		}

		sched.mu.Unlock()
		_, err := stub.SendConData(ctx, data)
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
	}).Debug("waiting for goroutine to finish sendOne")

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
		}).Debug("waiting for all responses")

		for len(sched.CurData())-1 != sched.inNeighbours {
			sched.neighCond.Wait()
		}

	}
	log.Debug("finished waiting for all responses")

}

func (sched *Scheduler) LocalComp() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	myData := sched.MyData()
	curData := sched.CurData()

	log.WithFields(log.Fields{
		"iteration": sched.k,
		"data":      myData,
	}).Debug("current data")

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
		P:    myData.GetP(),
		Y:    newY,
		Z:    newZ,
		M:    newm,
		Mm:   newM,
		Flag: false,
	}

	log.WithFields(log.Fields{
		"to":           sched.k + 1,
		"updated data": sched.conData[sched.k+1][sched.hostname],
	}).Debug("awesome computation done, advancing iteration counter")

	if !sched.done {
		sched.k++
	}

}

func (sched *Scheduler) Consensus() {
	if *config.CpuProfile != "" {
		f, err := os.Create(*config.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ts := make([]int64, 0)

	for !sched.CheckCvg() && sched.k < *config.MaxIter {
		t := time.Now()

		log.Debug("doing msg exchange...")
		sched.MsgXchg()

		t1 := time.Now()

		log.Debug("doing local computation...")
		sched.LocalComp()

		t2 := time.Now()

		ts = append(ts, time.Since(t).Microseconds())
		metricsLogger.WithFields(log.Fields{
			"iteration":                           sched.k,
			"msg_exchange_time in micro sec ":     t1.Sub(t).Microseconds(),
			"local_computation_time in micro sec": t2.Sub(t1).Microseconds(),
			"time per iteration":                  ts[len(ts)-1],
		}).Info("time of this iteration")
	}

	var tot int64
	for _, t := range ts {
		tot += t
	}

	log.WithFields(log.Fields{
		"iteration":         sched.k,
		"data":              sched.MyData(),
		"average consensus": sched.MyData().GetY() / sched.MyData().GetZ(),
	}).Info("consensus done!")

	metricsLogger.WithFields(log.Fields{
		"average time per iteration": float64(tot) / float64(len(ts)),
		"total time taken":           tot,
	}).Info("consensus time")

}

func (sched *Scheduler) CurData() map[string]*pb.ConData {
	return sched.conData[sched.k]
}

func (sched *Scheduler) MyData() *pb.ConData {
	return sched.conData[sched.k][sched.hostname]
}
