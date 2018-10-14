/* See the file "LICENSE.txt" for the full license governing this code. */

// Wrapper type for all functions that depend on current time
// This is useful to use a mock clock implementation for testing

package main

import (
	"time"
)

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type skewClock struct {
	skew time.Duration
}

func (cl *skewClock) Now() time.Time {
	return time.Now().Add(-cl.skew)
}

func newSkewClock(i int64) *skewClock {
	d := time.Now().Sub(time.Unix(i, 0))
	return &skewClock{skew: d}
}

func (cl *skewClock) forward(d time.Duration) {
	cl.skew -= d
}
