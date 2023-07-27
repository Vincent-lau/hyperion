/*
	Specific data structure for the asynchronous ratio consensus algorithm
*/

package scheduler

import (
	"math"
	"os"
	"time"

	config "github.com/Vincent-lau/hyperion/internal/configs"
	pb "github.com/Vincent-lau/hyperion/internal/message"

	linuxproc "github.com/c9s/goprocinfo/linux"
	log "github.com/sirupsen/logrus"
)

const (
	CPU_INTERVAL = time.Second * 1
)

func (sched *Scheduler) InitMyConData(l, u, pi float64) {

	if _, ok := sched.conData[0]; !ok {
		sched.conData[0] = make(map[int]*pb.ConData)
	}

	// myu, mypi := sched.k8sCpuUsage()
	myu, mypi := fakeCpuUsage()
	log.WithFields(log.Fields{
		"my used k8s":     myu,
		"my capacity k8s": mypi,
	}).Debug("cpu usage")

	// this node's data
	sched.conData[0][sched.me] = &pb.ConData{
		P:    1 / (float64(sched.outNeighbours) + 1), // p for this node
		Y:    l + myu,
		Z:    mypi,
		Mm:   math.Inf(1),
		M:    math.Inf(-1),
		Flag: false,
	}

	sched.u = myu
	sched.pi = mypi

	host, _ := os.Hostname()
	log.WithFields(log.Fields{
		"data":        sched.MyData(),
		"my used":     myu,
		"my capacity": mypi,
		"host":        host,
		"trial":       sched.trial,
	}).Debug("initialized conData")

}

func (sched *Scheduler) k8sCpuUsage() (float64, float64) {
	u, pi := sched.MyCpu()
	return float64(u), float64(pi)
}

// purely for simulation
// assume there is no load on the node
func fakeCpuUsage() (float64, float64) {
	return 0.0, config.MaxCap
}

func localCpuUsage() (float64, float64) {
	prev := readCPUStats()
	time.Sleep(CPU_INTERVAL)
	curr := readCPUStats()

	pct := cpuPct(curr, prev)

	info, err := linuxproc.ReadCPUInfo("/proc/cpuinfo")
	if err != nil {
		log.Fatal(err)
	}

	mCores := float64(len(info.Processors) * 1000)
	used := pct * mCores
	pi := mCores

	return used, pi
}

func readCPUStats() linuxproc.CPUStat {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("stat read fail")
	}
	return stat.CPUStatAll
}

func cpuPct(curr, prev linuxproc.CPUStat) float64 {
	PrevIdle := prev.Idle + prev.IOWait
	Idle := curr.Idle + curr.IOWait

	PrevNonIdle := prev.User + prev.Nice + prev.System + prev.IRQ + prev.SoftIRQ + prev.Steal
	NonIdle := curr.User + curr.Nice + curr.System + curr.IRQ + curr.SoftIRQ + curr.Steal

	PrevTotal := PrevIdle + PrevNonIdle
	Total := Idle + NonIdle
	// fmt.Println(PrevIdle, Idle, PrevNonIdle, NonIdle, PrevTotal, Total)

	//  differentiate: actual value minus the previous one
	totald := Total - PrevTotal
	idled := Idle - PrevIdle

	cpupct := (float64(totald) - float64(idled)) / float64(totald)

	return cpupct

}
