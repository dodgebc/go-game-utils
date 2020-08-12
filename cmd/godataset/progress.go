package main

import (
	"fmt"
	"time"
)

// ProgressUpdate provides a real-time record of progress
type ProgressUpdate struct {
	startTime     time.Time
	lastUpdate    time.Time
	iteration     int
	lastIteration int
	otherKeys     []string
	otherValues   []int
	description   string
}

// NewProgressUpdate starts a progress update
func NewProgressUpdate(description string) *ProgressUpdate {
	pu := &ProgressUpdate{
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		description: description,
	}
	return pu
}

// Update increments the progress update if enough time has gone by
func (pu *ProgressUpdate) Update(n int) {
	pu.iteration += n
	if time.Since(pu.lastUpdate).Seconds() > 0.5 {
		fmt.Printf(
			"%s: %d it\t%.0f it/s",
			pu.description,
			pu.iteration,
			float64(pu.iteration-pu.lastIteration)/time.Since(pu.lastUpdate).Seconds(),
		)
		for i := range pu.otherKeys {
			fmt.Printf("\t%s: %d", pu.otherKeys[i], pu.otherValues[i])
		}
		fmt.Print("\t\r")
		pu.lastUpdate = time.Now()
		pu.lastIteration = pu.iteration
	}
}

// SetOther sets the value of an additional statistic (or creates it)
func (pu *ProgressUpdate) SetOther(key string, value int) {
	for i := range pu.otherKeys {
		if pu.otherKeys[i] == key {
			pu.otherValues[i] = value
			return
		}
	}
	pu.otherKeys = append(pu.otherKeys, key)
	pu.otherValues = append(pu.otherValues, value)
}

// Close ends the progress bar by printing a new line
func (pu *ProgressUpdate) Close() {
	fmt.Printf(
		"%s: %d it\t%.0f it/s",
		pu.description,
		pu.iteration,
		float64(pu.iteration)/time.Since(pu.startTime).Seconds(),
	)
	for i := range pu.otherKeys {
		fmt.Printf("\t%s: %d", pu.otherKeys[i], pu.otherValues[i])
	}
	fmt.Print("\t\r\n")
}
