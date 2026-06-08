package sessions

import (
	"sync"
	"time"
)

type CallState int

const (
	CallIdle    CallState = iota
	CallActive
	CallEnded
)

// ActiveCall tracks an ongoing call session.
type ActiveCall struct {
	mu        sync.Mutex
	state     CallState
	startTime time.Time
}

func NewCall() *ActiveCall {
	return &ActiveCall{
		state:     CallActive,
		startTime: time.Now(),
	}
}

func (c *ActiveCall) End() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = CallEnded
}

func (c *ActiveCall) IsActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state == CallActive
}

func (c *ActiveCall) Duration() time.Duration {
	return time.Since(c.startTime)
}
