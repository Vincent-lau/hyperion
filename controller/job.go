package controller


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
