package resilience

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/DavidOrtegaFarrerons/infergrid/internal/application"
)

type CircuitBreakerRunner struct {
	inner   application.InferenceRunner
	breaker *CircuitBreaker
}

func NewCircuitBreakerRunner(inferenceRunner application.InferenceRunner, circuitBreaker *CircuitBreaker) *CircuitBreakerRunner {
	return &CircuitBreakerRunner{
		inner:   inferenceRunner,
		breaker: circuitBreaker,
	}
}

func (r *CircuitBreakerRunner) isTransient(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

func (r *CircuitBreakerRunner) Generate(ctx context.Context, prompt string) (string, error) {
	if err := r.breaker.Allow(); err != nil {
		return "", fmt.Errorf("%w: %v", application.ErrInferenceUnavailable, err)
	}

	result, err := r.inner.Generate(ctx, prompt)
	if err != nil {
		if r.isTransient(err) {
			r.breaker.RecordFailure()
			return "", fmt.Errorf("%w: %v", application.ErrInferenceUnavailable, err)
		}
		return "", err

	}

	r.breaker.RecordSuccess()
	return result, nil
}

type state string

const (
	StateOpen     = state("open")
	StateClosed   = state("closed")
	StateHalfOpen = state("half-open")
)

var ErrCircuitOpen = errors.New("circuit breaker is open, please try again later")

type CircuitBreaker struct {
	mu               sync.Mutex
	state            state
	consecutiveFails int
	openedAt         time.Time
	failureThreshold int
	cooldown         time.Duration
}

func NewCircuitBreaker(failureThreshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		cooldown:         cooldown,
	}
}

func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Now().Sub(cb.openedAt) >= cb.cooldown {
			cb.state = StateHalfOpen
			return nil
		}

		return ErrCircuitOpen
	case StateHalfOpen:
		return ErrCircuitOpen
	}

	return nil
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.consecutiveFails = 0
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.trip()
	case StateClosed:
		cb.consecutiveFails++
		if cb.consecutiveFails >= cb.failureThreshold {
			cb.trip()
		}
	}
}

func (cb *CircuitBreaker) trip() {
	cb.state = StateOpen
	cb.openedAt = time.Now()
	cb.consecutiveFails = 0
}
