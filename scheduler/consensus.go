package scheduler

import (
	"context"
	"example/dist_sched/config"
	pb "example/dist_sched/message"
	"example/dist_sched/util"
	"math"
	"os"
	"runtime/pprof"
	"sync"
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

		prevRound, _ := sched.conData.Load(sched.k - config.Diameter)
		prevData, _ := prevRound.(*sync.Map).Load(sched.me)
		prevFlag := prevData.(*pb.ConData).GetFlag()

		log.WithFields(log.Fields{
			"k":           sched.k,
			"prev k":      sched.k - config.Diameter,
			"prev k flag": prevFlag,
		}).Debug("checking flag at previous round")

		return prevFlag
	} else {
		return false
	}
}

func (sched *Scheduler) CheckCvg() bool {
	if sched.done.Load() {
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

			kData, _ := sched.conData.Load(sched.k)
			kData.(*sync.Map).Store(sched.me, &pb.ConData{
				P:    myData.GetP(),
				Y:    myData.GetY(),
				Z:    myData.GetZ(),
				Mm:   mu,
				M:    mu,
				Flag: flag,
			})

			// now check for termination
			if sched.getPrevRoundFlag() && flag {
				log.WithFields(log.Fields{
					"name":  sched.hostname,
					"k":     sched.k,
					"data":  sched.MyData(),
					"ratio": sched.MyData().GetY() / sched.MyData().GetZ(),
				}).Debug("termination reached")

				sched.done.Store(true)
			}

		}

	} else { // flag == 1
		if math.Abs(myData.GetMm()-myData.GetM()) >= *config.Tolerance {
			kData, _ := sched.conData.Load(sched.k)
			kData.(*sync.Map).Store(sched.me, &pb.ConData{
				P:    myData.GetP(),
				Y:    myData.GetY(),
				Z:    myData.GetZ(),
				Mm:   myData.GetMm(),
				M:    myData.GetM(),
				Flag: false,
			})

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

func (sched *Scheduler) sendOne(to int, stream pb.RatioConsensus_SendConDataClient, data *pb.ConDataRequest, done chan int, info chan sendTime) {
	t := time.Now()
	for {
		log.WithFields(log.Fields{
			"to":   to,
			"k":    sched.k,
			"data": data,
		}).Debug("sending data in sendOne")

		s := uint64(proto.Size(data))
		atomic.AddUint64(&sched.msgSent, s)

		if err := stream.Send(data); err != nil {
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

	if *config.Mode == "dev" {
		sched.mu.Lock()
		log.WithFields(log.Fields{
			"me":        sched.me,
			"to":        to,
			"trial":     sched.trial,
			"iteration": sched.k,
			"took":      time.Since(t).Microseconds(),
		}).Debug("time taken to sendOne")
		sched.mu.Unlock()
	}

	done <- to
	info <- sendTime{to: to, time: time.Since(t).Microseconds()}
}

func (sched *Scheduler) MsgXchg() {
	sched.mu.Lock()
	defer sched.mu.Unlock()

	done := make(chan int)
	info := make(chan sendTime)

	// TODO move this to a separate function
	util.StartTrace(sched.me, sched.trial, sched.k)

	data := &pb.ConDataRequest{
		K:    sched.k,
		Me:   int32(sched.me),
		Data: sched.MyData(),
	}

	for _, you := range sched.outConns {
		go sched.sendOne(you, sched.streams[you], data, done, info)
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
	}).Debug("finished waiting for goroutines to finish sendOne")

	for {
		missing := make([]int, 0)
		for _, from := range sched.inConns {
			kData, _ := sched.conData.Load(sched.k)
			if _, ok := kData.(*sync.Map).Load(from); !ok {
				missing = append(missing, from)
			}
		}

		x, _ := sched.conLen.Load(sched.k)
		c := x.(uint64)

		log.WithFields(log.Fields{
			"me":              sched.me,
			"k":               sched.k,
			"missing no":      len(missing),
			"missing from":    missing,
			"received no":     c - 1,
			"expecting total": sched.inNeighbours,
		}).Info("data reception status")

		if c-1 == sched.inNeighbours {
			break
		}

		t := time.Now()
		// for len(sched.CurData())-1 != sched.inNeighbours {
		sched.neighCond.Wait()
		// }

		log.WithFields(log.Fields{
			"took": time.Since(t),
		}).Info("finished waiting for all responses")
	}

	util.StopTrace()

}

func (sched *Scheduler) LocalComp() {
	myData := sched.MyData()
	curData := sched.CurData()

	l, _ := sched.conLen.Load(sched.k)
	log.WithFields(log.Fields{
		"iteration":        sched.k,
		"current data len": l,
	}).Debug("current data")

	sched.CurData().Range(func(key any, value any) bool {
		log.WithFields(log.Fields{
			"key": key.(int),
			"val": value,
		}).Debug("conData internal")
		return true
	})

	newY := myData.GetY() * myData.GetP()
	newZ := myData.GetZ() * myData.GetP()
	newM := myData.GetMm()
	newm := myData.GetM()

	for _, r := range sched.inConns {
		log.WithFields(log.Fields{
			"neighbour": r,
		}).Debug("getting data from")

		t, _ := curData.Load(r)
		rData := t.(*pb.ConData)

		newY += rData.GetY() * rData.GetP()
		newZ += rData.GetZ() * rData.GetP()

		newM = math.Max(newM, rData.GetMm())
		newm = math.Min(newm, rData.GetM())
	}

	newData := &pb.ConData{
		P:    myData.GetP(),
		Y:    newY,
		Z:    newZ,
		M:    newm,
		Mm:   newM,
		Flag: false,
	}

	// sched.k+1 might have been created by receiving response from other nodes
	kData, _ := sched.conData.LoadOrStore(sched.k+1, &sync.Map{})
	kData.(*sync.Map).Store(sched.me, newData)
	c, _ := sched.conLen.LoadOrStore(sched.k+1, uint64(0))
	sched.conLen.Store(sched.k+1, c.(uint64)+1)

	log.WithFields(log.Fields{
		"to":           sched.k + 1,
		"updated data": newData,
	}).Debug("awesome computation done, advancing iteration counter")

	if !sched.done.Load() {
		atomic.AddUint64(&sched.k, 1)
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

	var err error
	for you, stub := range sched.stubs {
		if sched.streams[you], err = stub.SendConData(context.Background()); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"to":    you,
			}).Fatal("failed to create stream")
		}
	}

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
		}).Info("time of this iteration")
	}

	for you, s := range sched.streams {
		if _, err = s.CloseAndRecv(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"to":    you,
			}).Fatal("failed to close stream")
		}
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

func (sched *Scheduler) CurData() *sync.Map {
	kData, ok := sched.conData.Load(sched.k)
	if !ok {
		log.WithFields(log.Fields{
			"iteration": sched.k,
		}).Fatal("failed to load current data")
	}
	return kData.(*sync.Map)
}

func (sched *Scheduler) MyData() *pb.ConData {
	curData := sched.CurData()
	mData, ok := curData.Load(sched.me)
	if !ok {
		log.WithFields(log.Fields{
			"iteration": sched.k,
			"me":        sched.me,
		}).Fatal("failed to load my data")
	}
	return mData.(*pb.ConData)

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
