// Command steady generates continuous background traffic against the order API.
// It creates orders with valid discount codes, reads them back, and occasionally
// issues refunds — producing a healthy baseline for the CloudWatch dashboard.
package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/example/demo-incident-response/traffic/internal/client"
)

var validCodes = []string{"SAVE5", "SAVE10", "SAVE15", ""}

func main() {
	target := flag.String("target", "http://localhost:8080", "base URL of the order API")
	rate := flag.Float64("rate", 2, "requests per second")
	flag.Parse()

	c := client.New(*target)
	interval := time.Duration(float64(time.Second) / *rate)

	fmt.Printf("steady  target=%s  rate=%.1f req/s\n", *target, *rate)

	var created, reads, refunds, errors atomic.Int64
	var orderIDs []string

	// Print stats every 5 seconds.
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for range tick.C {
			fmt.Printf("  created=%-4d  reads=%-4d  refunds=%-4d  errors=%-4d\n",
				created.Load(), reads.Load(), refunds.Load(), errors.Load())
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-stop:
			fmt.Printf("\nstopping. totals: created=%d  reads=%d  refunds=%d  errors=%d\n",
				created.Load(), reads.Load(), refunds.Load(), errors.Load())
			return
		case <-tick.C:
			go func() {
				// Mix of operations: 50% create, 35% read, 15% refund.
				r := rand.Float64()
				switch {
				case r < 0.50:
					code := validCodes[rand.IntN(len(validCodes))]
					order, _, err := c.CreateOrder(client.CreateRequest{
						Items:        client.RandomItems(),
						DiscountCode: code,
					})
					if err != nil {
						errors.Add(1)
						fmt.Fprintf(os.Stderr, "  ERR create: %v\n", err)
						return
					}
					created.Add(1)
					orderIDs = append(orderIDs, order.ID)

				case r < 0.85:
					if len(orderIDs) == 0 {
						return
					}
					id := orderIDs[rand.IntN(len(orderIDs))]
					_, _, err := c.GetOrder(id)
					if err != nil {
						errors.Add(1)
						fmt.Fprintf(os.Stderr, "  ERR read: %v\n", err)
						return
					}
					reads.Add(1)

				default:
					if len(orderIDs) == 0 {
						return
					}
					id := orderIDs[rand.IntN(len(orderIDs))]
					_, status, err := c.RefundOrder(id)
					if err != nil && status != 409 { // 409 = already refunded, fine
						errors.Add(1)
						fmt.Fprintf(os.Stderr, "  ERR refund: %v\n", err)
						return
					}
					refunds.Add(1)
				}
			}()
		}
	}
}
