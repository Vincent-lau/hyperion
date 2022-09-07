package scheduler

import (
	"context"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"fmt"
	"math"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

func (sched *Scheduler) timeToCheck() bool {
	return sched.k%config.Diameter == 0 && sched.k != 0
}

func (sched *Scheduler) getPrevRoundFlag() bool {
	if sched.k >= config.Diameter {

		log.WithFields(log.Fields{
			"k":           sched.k,
			"prev k":      sched.k - config.Diameter,
			"prev k flag": sched.conData[sched.k-config.Diameter][sched.me].GetFlag(),
		}).Debug("checking flag at previous round")

		return sched.conData[sched.k-config.Diameter][sched.me].GetFlag()
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

			sched.conData[sched.k][sched.me] = &pb.ConData{
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
			sched.conData[sched.k][sched.me] = &pb.ConData{
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

type sendTime struct {
	to   int
	time int64
}

func (sched *Scheduler) sendOne(to int, stub pb.RatioConsensusClient, data *pb.ConDataRequest, done chan int, info chan sendTime) {
	t := time.Now()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		log.WithFields(log.Fields{
			"to":   to,
			"k":    sched.k,
			"data": data,
		}).Debug("sending data in sendOne")

		s := uint64(proto.Size(data))
		atomic.AddUint64(&sched.msgSent, s)

		_, err := stub.SendConData(ctx, data)

		if err != nil {
			log.WithFields(log.Fields{
				"error":              err,
				"sending request to": to,
				"iteration":          sched.k,
			}).Warn("error send conData")

			time.Sleep(time.Second)

		} else {
			break
		}
	}

	log.WithFields(log.Fields{
		"me":        sched.me,
		"to":        to,
		"trial":     sched.trial,
		"iteration": sched.k,
		"took":      time.Since(t).Microseconds(),
	}).Debug("time taken to sendOne")
	done <- to
	info <- sendTime{to: to, time: time.Since(t).Microseconds()}
}

func (sched *Scheduler) MsgXchg() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	done := make(chan int)
	info := make(chan sendTime)

	if *config.Trace != "" {
		s := fmt.Sprintf("trace/trace-s%d-t%d-r%d.out", sched.me, sched.trial, sched.k)
		if err := os.MkdirAll("trace", os.ModePerm); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("error creating trace directory")
		}
		f, err := os.Create(s)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error creating trace file")
		}

		if err := trace.Start(f); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("error starting trace")
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Fatal("failed to close trace file")
			}
		}()
	}

	data := &pb.ConDataRequest{
		K:    int32(sched.k),
		Me:   int32(sched.me),
		Data: sched.MyData(),
	}

	for _, you := range sched.outConns {
		go sched.sendOne(you, sched.stubs[you], data, done, info)
	}

	t := time.Now()

	sendTimes := make([]sendTime, 0)
	sched.mu.Unlock()
	for range sched.outConns {
		<-done
		sendTimes = append(sendTimes, <-info)
	}
	sched.mu.Lock()

	log.WithFields(log.Fields{
		"to":        sched.outConns,
		"took":      time.Since(t).Microseconds(),
		"sendTimes": sendTimes,
	}).Debug("finished waiting for goroutine to finish sendOne")

	for len(sched.CurData())-1 != sched.inNeighbours {

		missing := make([]int, 0)
		for _, from := range sched.inConns {
			if _, ok := sched.conData[sched.k][from]; !ok {
				missing = append(missing, from)
			}
		}

		log.WithFields(log.Fields{
			"me":           sched.me,
			"missing no":   sched.inNeighbours - (len(sched.CurData()) - 1),
			"missing from": missing,
			"k":            sched.k,
		}).Debug("waiting for all responses")

		t := time.Now()

		// for len(sched.CurData())-1 != sched.inNeighbours {
		sched.neighCond.Wait()
		// }

		log.WithFields(log.Fields{
			"took": time.Since(t).Microseconds(),
		}).Debug("finished waiting for all responses")

	}

	if *config.Trace != "" {
		trace.Stop()

	}

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
		sched.conData[sched.k+1] = make(map[int]*pb.ConData)
	}

	sched.conData[sched.k+1][sched.me] = &pb.ConData{
		P:    myData.GetP(),
		Y:    newY,
		Z:    newZ,
		M:    newm,
		Mm:   newM,
		Flag: false,
	}

	log.WithFields(log.Fields{
		"to":           sched.k + 1,
		"updated data": sched.conData[sched.k+1][sched.me],
	}).Debug("awesome computation done, advancing iteration counter")

	if !sched.done {
		sched.k++
	}

}

func (sched *Scheduler) LoopConsensus() {
	if *config.CpuProfile != "" {
		f, err := os.Create(*config.CpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *config.MemProfile != "" {
		f, err := os.Create(*config.MemProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}

	for {

		log.WithFields(log.Fields{
			"name":  sched.hostname,
			"trial": sched.trial,
			"me":    sched.me,
		}).Info("new trial is starting")

		sched.Consensus()
		sched.reset()

		sched.mu.Lock()
		for !sched.setup {
			sched.startCond.Wait()
		}
		sched.mu.Unlock()

	}

}

func (sched *Scheduler) Consensus() {

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
		MetricsLogger.WithFields(log.Fields{
			"iteration":          sched.k - 1,
			"xchg time per iter": t1.Sub(t).Microseconds(),
			"comp time per iter": t2.Sub(t1).Microseconds(),
			"time per iteration": ts[len(ts)-1],
		}).Debug("time of this iteration")
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

	MetricsLogger.WithFields(log.Fields{
		"avg time per iter": float64(tot) / float64(len(ts)),
		"total time":        tot,
		"total iter":        sched.k,
	}).Info("final consensus time")

	MetricsLogger.WithFields(log.Fields{
		"msg rcv total":  sched.msgSent,
		"msg sent total": sched.msgRcv,
	}).Info("consensus message exchanged")

	sched.sendFin()

}

func (sched *Scheduler) CurData() map[int]*pb.ConData {
	return sched.conData[sched.k]
}

func (sched *Scheduler) MyData() *pb.ConData {
	return sched.conData[sched.k][sched.me]
}

func (sched *Scheduler) sendFin() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		sched.mu.Lock()
		me := int32(sched.me)
		trial := int32(sched.trial)
		sched.mu.Unlock()

		_, err := sched.ctlRegStub.FinConsensus(ctx, &pb.FinRequest{
			Me:    me,
			Trial: trial,
		})

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Warn("failed to send fin consensus")

			time.Sleep(time.Second)
		} else {
			log.Debug("fin consensus sent")
			break
		}
	}

}
