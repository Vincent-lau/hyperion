package controller

import "fmt"


type Job struct {
	id int
	size float64
}


func (j *Job) Id() int {
	return j.id
}

func (j *Job) Size() float64 {
	return j.size
}

func (j *Job) String() string {
	return fmt.Sprintf("Job %d: %f", j.id, j.size)
}
