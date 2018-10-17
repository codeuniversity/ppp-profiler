package profiler

type runningAverage struct {
	Value            float64
	measurementCount int
}

func (a *runningAverage) Add(value float64) {
	a.Value = (a.Value*float64(a.measurementCount) + value) / float64(a.measurementCount+1)
	a.measurementCount++
}
