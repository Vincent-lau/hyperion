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
	return atomic.LoadUint64(&sched.k)%config.Diameter == 0 && atomic.LoadUint64(&sched.k) != 0
}

func (sched *Scheduler) getPrevRoundFlag() bool {
	if atomic.LoadUint64(&sched.k) >= config.Diameter {

		prevRound, _ := sched.conData.Load(atomic.LoadUint64(&sched.k) - config.Diameter)
		prevData, _ := prevRound.(*sync.Map).Load(atomic.LoadUint64(&sched.me))
		prevFlag := prevData.(*pb.ConData).GetFlag()

		log.WithFields(log.Fields{
			"k":           atomic.LoadUint64(&sched.k),
			"prev k":      atomic.LoadUint64(&sched.k) - config.Diameter,
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
					"iteration": atomic.LoadUint64(&sched.k),
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

			kData, _ := sched.conData.Load(atomic.LoadUint64(&sched.k))
			kData.(*sync.Map).Store(atomic.LoadUint64(&sched.me), &pb.ConData{
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
					"k":     atomic.LoadUint64(&sched.k),
					"data":  sched.MyData(),
					"ratio": sched.MyData().GetY() / sched.MyData().GetZ(),
				}).Debug("termination reached")

				sched.done.Store(true)
			}

		}

	} else { // flag == 1
		if math.Abs(myData.GetMm()-myData.GetM()) >= *config.Tolerance {
			kData, _ := sched.conData.Load(atomic.LoadUint64(&sched.k))
			kData.(*sync.Map).Store(atomic.LoadUint64(&sched.me), &pb.ConData{
				P:    myData.GetP(),
				Y:    myData.GetY(),
				Z:    myData.GetZ(),
				Mm:   myData.GetMm(),
				M:    myData.GetM(),
				Flag: false,
			})

			log.WithFields(log.Fields{
				"at":   atomic.LoadUint64(&sched.k),
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
			"k":    atomic.LoadUint64(&sched.k),
			"data": data,
		}).Debug("sending data in sendOne")

		s := uint64(proto.Size(data))
		atomic.AddUint64(&sched.msgSent, s)

		if err := stream.Send(data); err != nil {
			log.WithFields(log.Fields{
				"error":              err,
				"sending request to": to,
				"iteration":          atomic.LoadUint64(&sched.k),
			}).Warn("error send conData")

			time.Sleep(time.Second)
		} else {
			break
		}
	}

	log.WithFields(log.Fields{
		"me":        atomic.LoadUint64(&sched.me),
		"to":        to,
		"trial":     atomic.LoadUint64(&sched.trial),
		"iteration": atomic.LoadUint64(&sched.k),
		"took":      time.Since(t).Microseconds(),
	}).Debug("time taken to sendOne")

	done <- to
	info <- sendTime{to: to, time: time.Since(t).Microseconds()}
}

func (sched *Scheduler) MsgXchg() {
	done := make(chan int)
	info := make(chan sendTime)

	util.StartTrace(atomic.LoadUint64(&sched.me), atomic.LoadUint64(&sched.trial), atomic.LoadUint64(&sched.k))

	data := &pb.ConDataRequest{
		K:    atomic.LoadUint64(&sched.k),
		Me:   atomic.LoadUint64(&sched.me),
		Data: sched.MyData(),
	}

	for _, you := range sched.outConns {
		go sched.sendOne(you, sched.streams[you], data, done, info)
	}

	t := time.Now()

	sendTimes := make([]sendTime, 0)
	for range sched.outConns {
		<-done
		sendTimes = append(sendTimes, <-info)
	}

	log.WithFields(log.Fields{
		"to":        sched.outConns,
		"took":      time.Since(t).Microseconds(),
		"sendTimes": sendTimes,
	}).Debug("finished waiting for goroutines to finish sendOne")

	missing := make([]int, 0)
	for _, from := range sched.inConns {
		kData, _ := sched.conData.Load(atomic.LoadUint64(&sched.k))
		if _, ok := kData.(*sync.Map).Load(from); !ok {
			missing = append(missing, from)
		}
	}

	// x, _ := sched.conLen.Load(atomic.LoadUint64(&sched.k))

	log.WithFields(log.Fields{
		"me":              atomic.LoadUint64(&sched.me),
		"k":               atomic.LoadUint64(&sched.k),
		"missing no":      len(missing),
		"missing from":    missing,
		"expecting total": atomic.LoadUint64(&sched.inNeighbours),
	}).Info("data reception status")

	notMyK := make([]uint64, 0)

	xt := time.Now()
	// for len(sched.CurData())-1 != sched.inNeighbours {
	for mk := range sched.xchgChan {
		if mk == atomic.LoadUint64(&sched.k) {
			break
		} else {
			notMyK = append(notMyK, mk)
		}
	}
	log.WithFields(log.Fields{
		"got not my k": notMyK,
	}).Debug("not my k")

	for _, mk := range notMyK {
		sched.xchgChan <- mk
	}
	// }

	log.WithFields(log.Fields{
		"took": time.Since(xt),
	}).Info("finished waiting for all responses")

	util.StopTrace()

}

func (sched *Scheduler) LocalComp() {
	myData := sched.MyData()
	curData := sched.CurData()

	// l, _ := sched.conLen.Load(atomic.LoadUint64(&sched.k))
	// log.WithFields(log.Fields{
	// 	"iteration":        atomic.LoadUint64(&sched.k),
	// 	"current data len": l,
	// }).Info("current data")

	sched.CurData().Range(func(key any, value any) bool {
		log.WithFields(log.Fields{
			"key": key,
			"val": value,
		}).Info("conData internal")
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

	// atomic.LoadUint64(&sched.k)+1 might have been created by receiving response from other nodes
	nextK := atomic.LoadUint64(&sched.k) + 1
	kData, _ := sched.conData.LoadOrStore(nextK, &sync.Map{})
	kData.(*sync.Map).Store(atomic.LoadUint64(&sched.me), newData)
	// c, _ := sched.conLen.LoadOrStore(nextK, uint64(0))
	// sched.conLen.Store(nextK, c.(uint64)+1)

	// // special case here for reception of all messages
	// if c.(uint64) == atomic.LoadUint64(&sched.inNeighbours) {
	// 	log.WithFields(log.Fields{
	// 		"iteration": nextK,
	// 		"now c":     c.(uint64) + 1,
	// 	}).Debug("adding my own data to conData, got all messages")
	// 	sched.xchgChan <- nextK
	// }

	log.WithFields(log.Fields{
		"to":           nextK,
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
			"trial": atomic.LoadUint64(&sched.trial),
			"me":    atomic.LoadUint64(&sched.me),
		}).Info("new trial is starting")

		sched.Consensus()
		sched.reset()

		for !sched.setup.Load() {
			sched.mu.Lock()
			sched.startCond.Wait()
			sched.mu.Unlock()
		}

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

	for !sched.CheckCvg() && atomic.LoadUint64(&sched.k) < *config.MaxIter {
		t := time.Now()

		log.Debug("doing msg exchange...")
		sched.MsgXchg()

		t1 := time.Now()

		log.Debug("doing local computation...")
		sched.LocalComp()

		t2 := time.Now()

		ts = append(ts, time.Since(t).Microseconds())
		MetricsLogger.WithFields(log.Fields{
			"trial":              atomic.LoadUint64(&sched.trial),
			"iteration":          atomic.LoadUint64(&sched.k) - 1,
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
		} else {
			log.WithFields(log.Fields{
				"me":  atomic.LoadUint64(&sched.me),
				"you": you,
			}).Info("stream closed")
		}
	}

	var tot int64
	for _, t := range ts {
		tot += t
	}

	log.WithFields(log.Fields{
		"iteration":         atomic.LoadUint64(&sched.k),
		"data":              sched.MyData(),
		"average consensus": sched.MyData().GetY() / sched.MyData().GetZ(),
	}).Info("consensus done!")

	MetricsLogger.WithFields(log.Fields{
		"trial":             atomic.LoadUint64(&sched.trial),
		"avg time per iter": float64(tot) / float64(len(ts)),
		"total time":        tot,
		"total iter":        atomic.LoadUint64(&sched.k),
	}).Info("final consensus time")

	MetricsLogger.WithFields(log.Fields{
		"trial":          atomic.LoadUint64(&sched.trial),
		"msg rcv total":  atomic.LoadUint64(&sched.msgSent),
		"msg sent total": atomic.LoadUint64(&sched.msgRcv),
	}).Info("consensus message exchanged")

	sched.sendFin()

}

func (sched *Scheduler) CurData() *sync.Map {
	kData, ok := sched.conData.Load(atomic.LoadUint64(&sched.k))
	if !ok {
		log.WithFields(log.Fields{
			"iteration": atomic.LoadUint64(&sched.k),
		}).Fatal("failed to load current data")
	}
	return kData.(*sync.Map)
}

func (sched *Scheduler) MyData() *pb.ConData {
	curData := sched.CurData()
	mData, ok := curData.Load(atomic.LoadUint64(&sched.me))
	if !ok {
		log.WithFields(log.Fields{
			"iteration": atomic.LoadUint64(&sched.k),
			"me":        atomic.LoadUint64(&sched.me),
		}).Fatal("failed to load my data")
	}
	return mData.(*pb.ConData)

}

func (sched *Scheduler) sendFin() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		me := atomic.LoadUint64(&sched.me)
		trial := atomic.LoadUint64(&sched.trial)

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
			log.Info("fin consensus sent")
			break
		}
	}

}
