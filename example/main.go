package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alaleks/circuitbreaker"
)

var breaker circuitbreaker.CircuitFunc[*http.Response]

func init() {
	var circuit = circuitbreaker.NewCircuit[*http.Response]()
	circuit.ConfigureCircuit(circuitbreaker.Settings{Timeout: 500 * time.Millisecond})
	breaker = circuit.Breaker(getHTTP)
}

func getHTTP(ctx context.Context) (*http.Response, error) {
	return http.Get("https://go.dev/")
}

func main() {

	ctx := context.Background()

	for i := 10; i < 20; i++ {
		res, err := breaker(ctx)
		if err == nil {
			fmt.Println(i, ":", res.StatusCode)
		} else {
			fmt.Println(i, ":", err)
		}
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			res, err := breaker(ctx)
			if err == nil {
				fmt.Println(j, ":", res.StatusCode)
			} else {
				fmt.Println(j, ":", err)
			}
		}(i)
	}
	wg.Wait()

}
