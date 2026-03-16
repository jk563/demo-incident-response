// Command inject fires requests using the WELCOME discount code to trigger
// the intentional index-out-of-range panic in the order API. It also sends
// read requests to maintain some traffic while the errors are occurring.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/example/demo-incident-response/traffic/internal/client"
)

func main() {
	target := flag.String("target", "http://localhost:8080", "base URL of the order API")
	rate := flag.Float64("rate", 1, "WELCOME requests per second")
	duration := flag.Duration("duration", 2*time.Minute, "how long to inject errors (0 = until interrupted)")
	flag.Parse()

	c := client.New(*target)
	interval := time.Duration(float64(time.Second) / *rate)

	fmt.Printf("inject  target=%s  rate=%.1f req/s  duration=%s\n", *target, *rate, *duration)
	fmt.Println("sending WELCOME discount code (expect 500s)...")

	var sent, failures, errors atomic.Int64

	// Print stats every 5 seconds.
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for range tick.C {
			fmt.Printf("  sent=%-4d  500s=%-4d  errors=%-4d\n",
				sent.Load(), failures.Load(), errors.Load())
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	var deadline <-chan time.Time
	if *duration > 0 {
		deadline = time.After(*duration)
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			printSummary(sent.Load(), failures.Load(), errors.Load())
			return
		case <-deadline:
			printSummary(sent.Load(), failures.Load(), errors.Load())
			return
		case <-tick.C:
			go func() {
				sent.Add(1)
				_, status, err := c.CreateOrder(client.CreateRequest{
					Items:        client.RandomItems(),
					DiscountCode: "WELCOME",
				})
				if err != nil {
					if status >= 500 {
						failures.Add(1)
					} else {
						errors.Add(1)
						fmt.Fprintf(os.Stderr, "  ERR: %v\n", err)
					}
					return
				}
			}()
		}
	}
}

func printSummary(sent, failures, errs int64) {
	fmt.Printf("\nstopping. totals: sent=%d  500s=%d  errors=%d\n", sent, failures, errs)
}
