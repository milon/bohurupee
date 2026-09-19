package oauth

import "time"

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// FrozenClock is a test clock that only moves when Advance is called.
type FrozenClock struct {
	t time.Time
}

func NewFrozenClock(t time.Time) *FrozenClock {
	return &FrozenClock{t: t}
}

func (c *FrozenClock) Now() time.Time { return c.t }

func (c *FrozenClock) Advance(d time.Duration) {
	c.t = c.t.Add(d)
}
