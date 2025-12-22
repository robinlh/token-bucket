package ratelimit

import "time"

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

type MockClock struct {
	CurrentTime time.Time
}

func (clock *MockClock) Now() time.Time {
	return clock.CurrentTime
}

func (clock *MockClock) Advance(d time.Duration) {
	clock.CurrentTime = clock.CurrentTime.Add(d)
}
