package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// default settings
// you can redefine thoose
const (
	defaultTimeout          = time.Duration(5 * time.Second)
	defaultRetryAt          = time.Duration(30 * time.Second)
	defaultInterval         = time.Duration(0 * time.Millisecond)
	defaultFailureThreshold = 5
)

// returned errors when reached failure threshold or timeout
var (
	ErrFailureThreshold = errors.New("circuit breaker: reached failure threshold")
	ErrTimeout          = errors.New("circuit breaker: reached timeout")
)

type CircuitType interface {
	interface{}
}

type CircuitFunc[T CircuitType] func(context.Context) (T, error)

type Circuit[T CircuitType] struct {
	counterFailure uint      // counts the number of currrent failures
	lastAttempt    time.Time // captures the last time the function was run
	mtx            sync.RWMutex
	Settings
}

type Settings struct {
	TryingRecovery   bool
	FailureThreshold uint
	Timeout          time.Duration
	Interval         time.Duration
	RetryAt          time.Duration
}

// constructor that creates a new Circuit structure
func NewCircuit[T CircuitType]() Circuit[T] {
	return Circuit[T]{
		Settings: Settings{
			FailureThreshold: defaultFailureThreshold,
			Timeout:          defaultTimeout,
			Interval:         defaultInterval,
			RetryAt:          defaultRetryAt,
			TryingRecovery:   true,
		},
		mtx: sync.RWMutex{},
	}
}

// settings configurator
func (c *Circuit[T]) ConfigureCircuit(settings Settings) {

	if !settings.TryingRecovery {
		settings.TryingRecovery = false
	}

	if settings.FailureThreshold == 0 {
		c.FailureThreshold = defaultFailureThreshold
	} else {
		c.FailureThreshold = settings.FailureThreshold
	}

	if settings.Timeout == 0 {
		c.Timeout = defaultTimeout
	} else {
		c.Timeout = settings.Timeout
	}

	if settings.Interval == 0 {
		c.Interval = defaultInterval
	} else {
		c.Interval = settings.Interval
	}

	if settings.RetryAt == 0 {
		c.RetryAt = defaultRetryAt
	} else {
		c.RetryAt = settings.RetryAt
	}

}

// wrapper method for CircuitFunc
// accepts any function matching the signature func(context.Context) (any, error)
// returns a closure
func (c *Circuit[T]) Breaker(cfn CircuitFunc[T]) CircuitFunc[T] {
	return func(ctx context.Context) (T, error) {

		var empty T

		c.mtx.RLock()
		//check that the failure threshold has not been reached
		if c.counterFailure >= c.FailureThreshold {

			//repeat after a certain period of time to recover the circuit
			// only when TryingRecovery is true
			if time.Now().After(c.lastAttempt.Add(c.RetryAt)) && c.TryingRecovery {
				c.ResetCountFailure()
				c.mtx.RUnlock()
				return cfn(ctx)
			}
			c.mtx.RUnlock()

			//if the failure threshold has been reached, we breaker circuit
			return empty, ErrFailureThreshold
		}
		c.mtx.RUnlock()

		c.mtx.Lock()
		defer c.mtx.Unlock()

		// if functions performed concurrently
		// and reached failure threshold
		if c.counterFailure >= c.FailureThreshold {
			return empty, ErrFailureThreshold
		}

		// time interval
		<-time.After(c.Interval)
		//capture last attempt
		c.lastAttempt = time.Now()

		done := make(chan struct {
			res T
			err error
		})

		go func() {
			res, err := cfn(ctx)
			done <- struct {
				res T
				err error
			}{res, err}
		}()

		select {
		case data := <-done:
			if data.err != nil {
				c.counterFailure++ // increment counterFailure
				return data.res, data.err
			}
			c.counterFailure = 0 // reset counterFailure
			return data.res, data.err
		case <-ctx.Done(): // increment counterFailure
			c.counterFailure++
			return empty, ctx.Err()
		case <-time.After(c.Timeout): // increment counterFailure
			c.counterFailure++
			return empty, ErrTimeout
		}

	}
}

// reset counterFailure till 0
func (c *Circuit[T]) ResetCountFailure() {
	c.counterFailure = 0
}
