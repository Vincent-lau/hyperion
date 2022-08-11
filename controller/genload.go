package controller

import "math/rand"

func (ctl *Controller) randGen() {

	maxCap := 10
	for i := range ctl.load {
		ctl.load[i] = float64(rand.Intn(maxCap) + 1)
		ctl.used[i] = float64(rand.Intn(maxCap + 1 - int(ctl.load[i])))
		ctl.cap[i] = float64(maxCap)
	}

}

func (ctl *Controller) preComp() {

	maxCap := 10
	ctl.load = []float64{6, 10, 9, 6, 7, 1, 1, 6, 8, 2, 2, 10, 10, 9, 2, 8, 3, 5, 1, 9, 8, 7, 3, 6, 10, 4, 4, 2, 2, 4, 6, 9, 9, 6, 5, 7, 2, 1, 5, 2, 10, 3, 2, 1, 5, 6, 7, 4, 6, 3, 9, 10, 4, 10, 1, 4, 4, 3, 6, 7, 6, 10, 4, 2, 7, 9, 9, 10, 8, 10, 1, 10, 10, 3, 8, 8, 3, 9, 3, 3, 5, 1, 9, 2, 8, 4, 4, 7, 10, 4, 3, 9, 8, 5, 9, 3, 2, 8, 1, 9}
	ctl.used = []float64{1, 0, 1, 2, 1, 3, 5, 4, 1, 5, 8, 0, 0, 1, 3, 1, 5, 1, 6, 0, 2, 1, 7, 3, 0, 3, 3, 1, 3, 5, 1, 1, 0, 2, 4, 2, 1, 5, 1, 7, 0, 0, 1, 7, 1, 3, 2, 2, 3, 5, 0, 0, 2, 0, 7, 6, 6, 6, 4, 2, 2, 0, 6, 1, 1, 1, 1, 0, 2, 0, 1, 0, 0, 2, 0, 1, 2, 0, 5, 1, 2, 4, 1, 5, 0, 4, 2, 3, 0, 2, 5, 1, 1, 4, 1, 6, 5, 2, 7, 1}
	for i := range ctl.load {
		ctl.cap[i] = float64(maxCap)
	}
}

func (ctl *Controller) GenLoad() {
	ctl.preComp()
}
