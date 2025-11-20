package clock

//go:generate mockgen -source=$GOFILE -destination=./mocks/${GOFILE} -package=mocks

import "time"

// Clock is an interface that abstracts time.Now() for testability.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real implementation of the Clock interface.
type SystemClock struct{}

// NewSystemClock creates a new SystemClock.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// Now returns the current time.
func (c *SystemClock) Now() time.Time {
	return time.Now()
}
