# circuitbreaker

**circuitbreaker** is very simple package that implements corresponding pattern.

## installation

```
go get github.com/alaleks/circuitbreaker
```

## settings
```
type Settings struct {
	TryingRecovery   bool // this parameter implements the ability to restore the state
	FailureThreshold uint
	Timeout          time.Duration
	Interval         time.Duration
	RetryAt          time.Duration
}
```

## example

```
var breaker circuitbreaker.CircuitFunc[*http.Response]

func init() {
	var circuit = circuitbreaker.NewCircuit[*http.Response]()
	circuit.ConfigureCircuit(circuitbreaker.Settings{Timeout: 900 * time.Millisecond})
	breaker = circuit.Breaker(getHTTP)
}

func getHTTP(ctx context.Context) (*http.Response, error) {
	return http.Get("https://google.com")
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


```
