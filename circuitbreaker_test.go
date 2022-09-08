package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

var circuit = NewCircuit[*http.Response]()
var breaker CircuitFunc[*http.Response]

func init() {
	circuit.ConfigureCircuit(Settings{Timeout: 1 * time.Millisecond})
	breaker = circuit.Breaker(getHTTP)
}

func getHTTP(ctx context.Context) (*http.Response, error) {
	return http.Get("https://google.com")
}

func TestBreakerWithFailureThreshold(t *testing.T) {
	circuit.ResetCountFailure()
	ctx := context.Background()
	ctxCanc, cancel := context.WithCancel(ctx)
	cancel()
	var err error
	for i := 1; i < 7; i++ {
		_, err = breaker(ctxCanc)
	}

	if !errors.Is(ErrFailureThreshold, err) {
		t.Errorf("error should be failure threshold")
	}
}

func TestBreakerWithTimeout(t *testing.T) {
	circuit.ResetCountFailure()
	ctx := context.Background()
	var err error
	for i := 1; i < 2; i++ {
		_, err = breaker(ctx)
	}

	if !errors.Is(ErrTimeout, err) {
		t.Errorf("error should be failure timeout")
	}
}

func TestBreakerRecover(t *testing.T) {
	circuit.ResetCountFailure()
	ctx := context.Background()
	var err error
	for i := 1; i < 10; i++ {
		if i == 7 {
			circuit.ConfigureCircuit(Settings{RetryAt: 3 * time.Second, Timeout: 2 * time.Second})
			time.Sleep(3 * time.Second)
		}
		_, err = breaker(ctx)
	}
	if err != nil {
		t.Errorf("error should be failure timeout")
	}
}
